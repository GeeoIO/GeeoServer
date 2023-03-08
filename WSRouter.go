package main

import (
	"encoding/json"
	"errors"
	"expvar"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var (
	// MessageSendInterval determines how often we'll send batches of updates
	MessageSendInterval = 1000 * time.Millisecond
	// ShowOwnAgent determines if messages for my view can include myself if I'm also an agent
	ShowOwnAgent = true // TODO LATER handle false too...

	activeConnections *expvar.Int
)

var (
	// ErrAgentExists is returned if the Agent ID is already used
	ErrAgentExists = errors.New("Agent ID already exists")
	// ErrViewExists is returned if the View ID is already used
	ErrViewExists = errors.New("View ID already exists")
	// ErrInvalidMessage is returned if the json message can't be parsed
	ErrInvalidMessage = errors.New("Invalid message format")
)

// WSRouter holds what a WS handler needs to work
type WSRouter struct {
	db  *GeeoDB
	whw *WebhookWriter
}

// NewWSRouter returns a new WSRouter
func NewWSRouter(db *GeeoDB, whw *WebhookWriter) *WSRouter {
	newwsh := WSRouter{db: db, whw: whw}
	activeConnections = expvar.NewInt("active_connections")

	return &newwsh
}

func (wsh *WSRouter) handle(upgrader websocket.Upgrader) func(http.ResponseWriter, *http.Request) {

	// handle a new websocket Connection
	// if the token is sent and is valid, we'll proceed
	// this function runs in its own goroutine. If it ever ends, the connection is dropped
	return func(res http.ResponseWriter, req *http.Request) {
		var identity string

		activeConnections.Add(1)
		defer func() {
			activeConnections.Add(-1)
		}()

		conn, err := upgrader.Upgrade(res, req, nil)
		if err != nil {
			log.Warn("upgrade error: ", err)
			return
		}

		t := req.Header.Get("X-GEEO-TOKEN")
		if t == "" {
			t = req.URL.Query().Get("token")
		}

		token, err := parseJWTToken(t)
		if err != nil {
			message := struct {
				Error   string `json:"error"`
				Message string `json:"message"`
			}{"Can't parse token, or token invalid", err.Error()}
			conn.WriteJSON(message)
			log.Warn("Can't parse token, or token invalid: ", err.Error())
			return
		}

		// TODO LATER connectionWebHook

		var agent *Agent
		var view *View

		capabilities := token.Capabilities
		// TODO check MaxView and MaxAirBeacon

		wsConn := newWSConn(conn)
		if capabilities.Produce {
			agent = wsh.db.addAgent(token.AgentID, wsConn, token.Public)
			identity = "agent:" + token.AgentID
		}
		if capabilities.Consume {
			view = wsh.db.addView(token.ViewID, wsConn)
			identity = "view:" + token.ViewID
		}
		if capabilities.Produce && capabilities.Consume {
			identity = "agent:" + token.AgentID + "+view:" + token.ViewID
		}
		log.Debug("login: ", identity)

		wsConn.Name = identity

		defer func() {
			if err := recover(); err != nil {
				log.Error(err)
			}
			log.Debug("logout: ", identity)
			if agent != nil {
				wsh.db.removeAgent(*agent.ID)
				wsh.handleAgentLeft(agent)
			}
			if view != nil {
				wsh.db.removeView(token.ViewID)
			}
			wsConn.close()
			// TODO LATER defer deconnectionWebHook
		}()

		// we'll use a single JSONCommand for this socket to limit allocations
		// command.clear() must be called before parsing a new command
		command := JSONCommand{}

		for {
			_, jsonString, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					return
				}
				if websocket.IsCloseError(err, websocket.CloseMessageTooBig) {
					wsConn.writeImmediateJSON(struct {
						Error   string `json:"error"`
						Message string `json:"message"`
					}{"Message too large", "Your message size exceeds geeo's limits"})
					log.Warn(identity, ": Message too large")
					// TODO LATER monitor read error rate
					return
				}
				log.Warn(identity, ": ", err.Error())
				return
			}

			command.clear()
			if err := json.Unmarshal(jsonString, &command); err != nil {
				wsConn.writeImmediateJSON(struct {
					Error   string `json:"error"`
					Message string `json:"message"`
				}{"Can't parse command (" + string(jsonString) + ")", err.Error()})
				log.Warn(identity, ": invalid JSON command")
			}

			if err := command.check(); err != nil {
				wsConn.writeImmediateJSON(struct {
					Error   string `json:"error"`
					Message string `json:"message"`
				}{"Invalid Command (" + string(jsonString) + ")", err.Error()})
				log.Warn(identity, ": invalid command")
			}

			log.Debug(identity, ": ", string(jsonString))

			if command.AgentPosition != nil && capabilities.Produce {
				wsh.handleAgentMove(agent, command.AgentPosition)
				// TODO LATER monitor move rate
			}

			if command.ViewPosition != nil && capabilities.Consume {
				viewSize := command.ViewPosition.Size()
				if viewSize[0] > capabilities.MaxView[0] || viewSize[1] > capabilities.MaxView[1] {
					wsConn.writeImmediateJSON(struct {
						Error string `json:"error"`
					}{"View size error: it can't be larger than what your JWT Token allows"})
					log.Warn(identity, ": View size error")
				} else {
					//view.ws.Flush()
					wsh.handleViewMove(view, command.ViewPosition)
					// TODO LATER monitor move rate
				}
			}

			if command.AgentPublicData != nil && capabilities.Produce {
				wsh.handleAgentPublicData(agent, command.AgentPublicData)
				// TODO LATER monitor change rate
			}

			if command.CreatePOI != nil && capabilities.POI {
				var creator *string
				if agent != nil {
					creator = agent.ID
				}
				wsh.handlePOICreate(command.CreatePOI.ID, command.CreatePOI.Pos, command.CreatePOI.PublicData, creator)
			}

			if command.RemovePOI != nil && capabilities.POI {
				var user *string
				if agent != nil {
					user = agent.ID
				}
				wsh.handlePOIRemove(command.RemovePOI.ID, user)
			}

			if command.CreateAirBeacon != nil && capabilities.AirBeacon {
				abSize := command.CreateAirBeacon.Pos.Size()
				if abSize[0] > capabilities.MaxAirBeacon[0] || abSize[1] > capabilities.MaxAirBeacon[1] {
					wsConn.writeImmediateJSON(struct {
						Error string `json:"error"`
					}{"Air Beacon size error: it can't be larger than what your JWT Token allows"})
					log.Warn(identity, ": AirBeacon size error")
				} else {
					var creator *string
					if agent != nil {
						creator = agent.ID
					}
					wsh.handleAirBeaconCreate(command.CreateAirBeacon.ID, command.CreateAirBeacon.Pos, command.CreateAirBeacon.PublicData, creator)
				}
			}

			if command.RemoveAirBeacon != nil && capabilities.AirBeacon {
				var user *string
				if agent != nil {
					user = agent.ID
				}
				wsh.handleAirBeaconRemove(command.RemoveAirBeacon.ID, user)
			}

			// TODO LATER log errors trying to perform action without capabilities
		}
	}
}
