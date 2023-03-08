package main

import (
	"net/http"

	"github.com/sirupsen/logrus"

	"encoding/json"

	"github.com/gorilla/mux"
)

func withTokenAndDB(db *GeeoDB, wsh *WSRouter, fn func(http.ResponseWriter, *http.Request, *JWTToken, *GeeoDB, *WSRouter)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		t := req.Header.Get("X-GEEO-TOKEN")
		if t == "" {
			t = req.URL.Query().Get("token")
		}
		token, err := parseJWTToken(t)
		if err != nil || !token.Capabilities.HTTP {
			message := struct {
				Error   string `json:"error"`
				Message string `json:"message"`
			}{"Can't parse token, or token invalid", err.Error()}
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(message)
			log.Warn("HTTP route: can't parse token, or token without HTTP cap")
			return
		}
		fn(w, req, token, db, wsh)
	}
}

// NewHTTPRouter returns the router for the geeo http api
func NewHTTPRouter(router *mux.Router, db *GeeoDB, wsh *WSRouter) *mux.Router {

	router.HandleFunc("/v1/ping", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-type", "application/json")
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(struct {
			Tag   string
			Build string
		}{Tag, Build})
	})

	router.HandleFunc("/v1/POI", withTokenAndDB(db, wsh, addRemovePOI))

	router.HandleFunc("/v1/airbeacon", withTokenAndDB(db, wsh, addRemoveAirBeacon))

	router.HandleFunc("/v1/log", setLogLevel) // doesn't need additional security, awaits bearer token

	return router
}

func addRemovePOI(w http.ResponseWriter, req *http.Request, token *JWTToken, db *GeeoDB, wsh *WSRouter) {
	w.Header().Set("Content-type", "application/json")

	if !token.Capabilities.POI {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(struct {
			Error string
		}{"Your token doesn't allow POI creation/removal"})
		log.Warn("POI HTTP route: Your token doesn't allow POI creation/removal")

		return
	}

	if req.Method != http.MethodPost && req.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(struct {
			Error string
		}{"Only POST and DELETE are supported by this endpoint"})
		log.Warn("POI HTTP route: Only POST and DELETE are supported by this endpoint")
		return
	}

	cmd := &JSONPOI{}
	err := json.NewDecoder(req.Body).Decode(cmd)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(struct {
			Error string
		}{"Can't parse json body"})
		log.Warn("POI HTTP route: Can't parse json body")
		return
	}
	log.Debug("POI HTTP command: ", cmd)

	switch req.Method {
	case http.MethodPost:
		log.Info("POST /v1/POI: ", *cmd.ID, " created by ", cmd.Creator, " at ", cmd.Pos)
		poi := db.addPOI(*cmd.ID, cmd.Pos, cmd.PublicData, cmd.Creator)

		go func() {
			message := poi.enterLeaveMessage(true)
			wsh.sendMessageToConsumersWithPoint(message, poi.GetPoint())
		}()

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(*poi)
	case http.MethodDelete:
		log.Info("DELETE /v1/POI: ", *cmd.ID)
		db.RLock()
		poi, found := db.pois[*cmd.ID]
		db.RUnlock()
		if !found {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(struct {
				Error string
			}{"POI not found"})
			log.Warn("POI HTTP route: POI not found")
			return
		}
		db.removePOI(poi)

		go func() {
			message := poi.enterLeaveMessage(false)
			wsh.sendMessageToConsumersWithPoint(message, poi.GetPoint())
		}()

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(*poi)
	}
}

func addRemoveAirBeacon(w http.ResponseWriter, req *http.Request, token *JWTToken, db *GeeoDB, wsh *WSRouter) {
	w.Header().Set("Content-type", "application/json")

	if !token.Capabilities.AirBeacon {
		json.NewEncoder(w).Encode(struct {
			Error string
		}{"Your token doesn't allow Air Beacon creation/removal"})
		log.Warn("AirBeacon HTTP route: Your token doesn't allow Air Beacon creation/removal")

		return
	}

	if req.Method != http.MethodPost && req.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(struct {
			Error string
		}{"Only POST and DELETE are supported by this endpoint"})
		log.Warn("AirBeacon HTTP route: Only POST and DELETE are supported by this endpoint")
		return
	}

	cmd := &JSONAirBeacon{}
	err := json.NewDecoder(req.Body).Decode(cmd)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(struct {
			Error string
		}{"Can't parse json body"})
		log.Warn("AirBeacon HTTP route: Can't parse json body")
		return
	}
	log.Debug("Airbeacon HTTP command: ", cmd)

	switch req.Method {
	case http.MethodPost:
		log.Info("POST /v1/airbeacon: ", *cmd.ID, " created by ", cmd.Creator, " at ", cmd.Pos)
		poi := db.addAirBeacon(*cmd.ID, cmd.Pos, cmd.PublicData, cmd.Creator)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(*poi)
	case http.MethodDelete:
		log.Info("DELETE /v1/airbeacon: ", *cmd.ID)
		db.RLock()
		ab, found := db.ab[*cmd.ID]
		db.RUnlock()
		if !found {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(struct {
				Error string
			}{"AirBeacon not found"})
			log.Warn("AirBeacon HTTP route: AirBeacon not found")
			return
		}
		db.removeAirBeacon(*ab.id)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(*ab)
	}
}

func setLogLevel(w http.ResponseWriter, req *http.Request) {
	auth := req.Header.Get("Authorization")

	if WebhookBearerToken == "" || (auth != WebhookBearerToken && req.URL.Query().Get("bearer") != WebhookBearerToken) {
		w.WriteHeader(http.StatusUnauthorized)
		log.Warn("Unauthorized attempt to set log level")
		return
	}

	if req.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		log.Warn("Wrong HTTP method to set log level")
		return
	}

	level := req.URL.Query().Get("level")
	log.Info("Setting log level to ", level)
	switch level {
	case "debug":
		log.SetLevel(logrus.DebugLevel)
	case "info":
		log.SetLevel(logrus.InfoLevel)
	case "warn":
		log.SetLevel(logrus.WarnLevel)
	case "error":
		log.SetLevel(logrus.ErrorLevel)
	default:
		log.Warn("Invalid log level, ", level)
	}
	w.WriteHeader(http.StatusOK)
}
