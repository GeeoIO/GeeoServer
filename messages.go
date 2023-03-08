package main

import (
	"errors"

	"geeo.io/GeeoServer/quad"
)

// JSONChangeMessage tags update messages sent to clients
type JSONChangeMessage interface{}

// JSONCommand holds WS messages
type JSONCommand struct {
	AgentPosition   *quad.Point            `json:"agentPosition"`
	AgentPublicData map[string]interface{} `json:"publicData"`
	ViewPosition    *quad.Rect             `json:"viewPosition"`
	CreatePOI       *JSONPOI               `json:"createPOI"`
	RemovePOI       *JSONPOI               `json:"removePOI"`
	CreateAirBeacon *JSONAirBeacon         `json:"createAirBeacon"`
	RemoveAirBeacon *JSONAirBeacon         `json:"removeAirBeacon"`
	// TODO add messages and geo events
	//SendMessage        *JSONMessage           `json:"sendMessage"`
	//SendEvent          *JSONEvent             `json:"sendMessage"`
}

func (j *JSONCommand) check() error {

	if j.AgentPosition != nil && !j.AgentPosition.IsValid() {
		return errors.New("Invalid agentPosition")
	}
	if j.ViewPosition != nil {
		// TODO we don't support wrapping around yet
		if j.ViewPosition[0] >= j.ViewPosition[2] ||
			j.ViewPosition[2] <= j.ViewPosition[0] {
			j.ViewPosition[0] = -180
			j.ViewPosition[2] = 180
		}
		if !j.ViewPosition.IsValid() {
			return errors.New("Invalid viewPosition")
		}
	}

	if j.CreatePOI != nil && !j.CreatePOI.Pos.IsValid() {
		return errors.New("Invalid POI position")
	}
	if j.RemovePOI != nil && j.RemovePOI.ID == nil {
		return errors.New("Invalid POI ID")
	}
	if j.CreateAirBeacon != nil && !j.CreateAirBeacon.Pos.IsValid() {
		return errors.New("Invalid AirBeacon position")
	}
	if j.RemoveAirBeacon != nil && j.RemoveAirBeacon.ID == nil {
		return errors.New("Invalid air beacon ID")
	}
	return nil
}

func (j *JSONCommand) clear() {
	j.AgentPosition = nil
	j.AgentPublicData = nil
	j.ViewPosition = nil
	j.CreatePOI = nil
	j.RemovePOI = nil
	j.CreateAirBeacon = nil
	j.RemoveAirBeacon = nil
}

// EnteredLeft is used to provide additional enter/leave information
type EnteredLeft struct {
	Entered bool `json:"entered,omitempty"`
	Left    bool `json:"left,omitempty"`
}

func (e *EnteredLeft) clearEnteredLeft() {
	e.Entered = false
	e.Left = false
}

// JSONPOI holds POI messages
type JSONPOI struct {
	ID         *string                `json:"poi_id"`
	Pos        *quad.Point            `json:"pos,omitempty"`
	PublicData map[string]interface{} `json:"publicData,omitempty"`
	Creator    *string                `json:"creator,omitempty"`
}

func (p *JSONPOI) clear() {
	p.ID = nil
	p.Pos = nil
	p.PublicData = nil
	p.Creator = nil
}

// JSONAirBeacon holds AirBeacon messages
type JSONAirBeacon struct {
	ID         *string                `json:"ab_id"`
	Pos        *quad.Rect             `json:"pos,omitempty"`
	PublicData map[string]interface{} `json:"publicData,omitempty"`
	Creator    *string                `json:"creator,omitempty"`
}

// JSONAgent describes the public view on agents
type JSONAgent struct {
	ID         *string                `json:"agent_id"`
	Pos        *quad.Point            `json:"pos,omitempty"`
	PublicData map[string]interface{} `json:"publicData,omitempty"`
}

func (a *JSONAgent) clear() {
	a.ID = nil
	a.Pos = nil
	a.PublicData = nil
}

// JSONPOIEnteredLeft holds POI enter/leave messages
type JSONPOIEnteredLeft struct {
	JSONChangeMessage `json:"JSONChangeMessage,omitempty"`
	JSONPOI
	EnteredLeft
}

func (poi *POI) enterLeaveMessage(enter bool) JSONChangeMessage {
	message := &JSONPOIEnteredLeft{}
	message.ID = poi.id
	if enter {
		message.Pos = poi.GetPoint()
		message.PublicData = poi.publicData
		message.Creator = poi.creator
		message.Entered = true
	} else {
		message.Left = true
	}

	return message
}

// JSONAgentEnteredLeft is sent through the WS when an agent enters/leaves a view
type JSONAgentEnteredLeft struct {
	JSONChangeMessage `json:"JSONChangeMessage,omitempty"`
	JSONAgent
	EnteredLeft
}

func (a *Agent) enterLeaveMessage(enter bool) JSONChangeMessage {
	message := &JSONAgentEnteredLeft{}
	message.ID = a.ID

	if enter {
		message.Pos = a.GetPoint()
		message.PublicData = a.publicData
		message.Entered = true
	} else {
		message.Left = true
	}
	return message
}
