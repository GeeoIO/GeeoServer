package main

import "geeo.io/GeeoServer/quad"

// View is the type of views
type View struct {
	id   *string
	ws   *wsConn
	rect *quad.Rect
	el   *quad.Element
}

// GetRect gets our position for quads
func (v *View) GetRect() *quad.Rect {
	return v.rect
}

// SetRect sets our position for quads
func (v *View) SetRect(r *quad.Rect) {
	v.rect = r
}

// GetNode gets our storage in quads
func (v *View) GetNode() *quad.Element {
	return v.el
}

// SetNode sets our storage in quads
func (v *View) SetNode(el *quad.Element) {
	v.el = el
}
