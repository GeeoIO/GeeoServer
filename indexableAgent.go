package main

import "geeo.io/GeeoServer/quad"

// AgentMoveMessage models a basic move message
type AgentMoveMessage struct {
	JSONChangeMessage `json:"JSONChangeMessage,omitempty"`
	ID                *string     `json:"agent_id"`
	Point             *quad.Point `json:"pos,omitempty"`
}

// Agent is the type of agents
// it can be directly serialized to represent an Agent Move JSON message
type Agent struct {
	ID         *string `json:"agent_id"`
	ws         *wsConn
	publicData map[string]interface{}

	Point *quad.Point `json:"pos,omitempty"`
}

// GetPoint returns the position of the agent for quads
func (a *Agent) GetPoint() *quad.Point {
	return a.Point
}

// SetPoint sets the position of the agent for quads
func (a *Agent) SetPoint(p *quad.Point) {
	a.Point = p
}
