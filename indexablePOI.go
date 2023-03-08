package main

import "geeo.io/GeeoServer/quad"

// POI is the type of points of interest
type POI struct {
	//JSONChangeMessage `json:"JSONChangeMessage,omitempty"`
	id         *string
	publicData map[string]interface{}
	creator    *string // the agent who created the POI, or null for a system poi

	point *quad.Point
}

// GetPoint gets our position for quads
func (poi *POI) GetPoint() *quad.Point {
	return poi.point
}

// SetPoint sets our position for quads
func (poi *POI) SetPoint(p *quad.Point) {
	poi.point = p
}
