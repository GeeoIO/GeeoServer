package main

import "geeo.io/GeeoServer/quad"

// AirBeacon models an Air Beacon
type AirBeacon struct {
	id         *string
	publicData map[string]interface{}
	creator    *string // the agent who created the AirBeacon, or null for a system AirBeacon
	rect       *quad.Rect
	el         *quad.Element
}

// GetRect returns the position of the AB for quads
func (ab *AirBeacon) GetRect() *quad.Rect {
	return ab.rect
}

// SetRect sets the position of the AB for quads
func (ab *AirBeacon) SetRect(r *quad.Rect) {
	ab.rect = r
}

// GetNode gets the quad elements that stores us in quads
func (ab *AirBeacon) GetNode() *quad.Element {
	return ab.el
}

// SetNode sets the quad elements that stores us in quads
func (ab *AirBeacon) SetNode(el *quad.Element) {
	ab.el = el
}
