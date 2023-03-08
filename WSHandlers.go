package main

import (
	"geeo.io/GeeoServer/quad"
	set "github.com/deckarep/golang-set"
)

func (wsh *WSRouter) handleAgentMove(agent *Agent, pos *quad.Point) {
	// TODO LATER return if move was too small or previous message was not long ago
	oldPosition := wsh.db.updateAgentPosition(*agent.ID, pos)

	enterMessage := agent.enterLeaveMessage(true)

	agentMessageB := AgentMoveMessage{
		ID:    agent.ID,
		Point: agent.Point,
	}

	if oldPosition != nil { // we had a position before

		leaveMessage := agent.enterLeaveMessage(false)

		beforeViews := wsh.db.getRectLikeWithPoint(oldPosition)
		afterViews := wsh.db.getRectLikeWithPoint(pos)

		agentleftview := beforeViews.Difference(afterViews)
		wsh.sendMessageToConsumers(leaveMessage, agentleftview)

		agentmovedinview := beforeViews.Intersect(afterViews)
		wsh.sendMessageToViews(agentMessageB, agentmovedinview) // sent only to Views

		agententeredview := afterViews.Difference(beforeViews)
		wsh.sendMessageToConsumers(enterMessage, agententeredview)

	} else {
		// TODO LATER factor with agententeredview
		// we didn't need to determine a max view size ! rtrees rock
		wsh.sendMessageToConsumersWithPoint(enterMessage, pos)
	}
}

func (wsh *WSRouter) handleViewMove(view *View, pos *quad.Rect) {
	// TODO check if the new pos intersects the old one
	// if not, we need a way to just "refresh" in the protocol

	// TODO LATER return if move was too small

	// TODO implement WS ping pong

	previousPos := wsh.db.updateViewPosition(*view.id, pos)

	viewPointsAfter := wsh.db.getPointLikeIn(pos)

	if previousPos != nil {
		viewPointsBefore := wsh.db.getPointLikeIn(previousPos)
		//viewPOIsBefore := wsh.db.getPOIsIn(previousPos)
		//unionBefore := viewAgentsBefore.Union(viewPOIsBefore)

		removed := viewPointsBefore.Difference(viewPointsAfter)
		added := viewPointsAfter.Difference(viewPointsBefore)

		for _ag := range removed.Iter() {
			if _ag == nil {
				continue
			}
			ag := _ag.(JSONMessageAble)
			message := ag.enterLeaveMessage(false)
			view.ws.writeJSON(message)
		}

		for _ag := range added.Iter() {
			if _ag == nil {
				continue
			}
			ag := _ag.(JSONMessageAble)
			message := ag.enterLeaveMessage(true)
			view.ws.writeJSON(message)
		}
	} else {

		for _ag := range viewPointsAfter.Iter() {
			if _ag == nil {
				continue
			}
			ag := _ag.(JSONMessageAble)
			message := ag.enterLeaveMessage(true)
			view.ws.writeJSON(message)
		}
	}

	// TODO handle AirBeacon and Event
}

func (wsh *WSRouter) handleAgentPublicData(agent *Agent, pub map[string]interface{}) {
	agent.publicData = pub

	message := &JSONAgent{}
	message.ID = agent.ID
	message.Pos = agent.GetPoint()
	message.PublicData = agent.publicData
	wsh.sendMessageToConsumersWithPoint(message, agent.GetPoint())
}

func (wsh *WSRouter) handleAgentLeft(agent *Agent) {
	message := &JSONAgentEnteredLeft{}
	message.ID = agent.ID
	message.Left = true
	wsh.sendMessageToConsumersWithPoint(message, agent.GetPoint())
}

func (wsh *WSRouter) handlePOICreate(id *string, pos *quad.Point, publicData map[string]interface{}, creator *string) {
	_, exists := wsh.db.pois[*id]
	if exists {
		// TODO error instead: poi_id already exists
		log.Warnf("Poi %s already exists", *id)
		return
	}
	poi := wsh.db.addPOI(*id, pos, publicData, creator)
	message := poi.enterLeaveMessage(true)
	wsh.sendMessageToConsumersWithPoint(message, poi.GetPoint())
}

func (wsh *WSRouter) handlePOIRemove(id *string, user *string) {
	poi, exists := wsh.db.pois[*id]
	if !exists {
		return
	}
	if user != nil && *poi.creator != *user { // TODO replace with ACL
		// TODO send error
		return
	}
	wsh.db.removePOI(poi)

	message := poi.enterLeaveMessage(false)
	wsh.sendMessageToConsumersWithPoint(message, poi.GetPoint())
}
func (wsh *WSRouter) handleAirBeaconCreate(id *string, pos *quad.Rect, publicData map[string]interface{}, creator *string) {
	_, exists := wsh.db.ab[*id]
	if exists {
		// TODO error instead: ab_id already exists
		log.Warnf("AirBeacon %s already exists", *id)
		return
	}
	wsh.db.addAirBeacon(*id, pos, publicData, creator)
}

func (wsh *WSRouter) handleAirBeaconRemove(id *string, user *string) {
	ab, exists := wsh.db.ab[*id]
	if !exists {
		// TODO return error
		return
	}
	if user != nil && *ab.creator != *user { // TODO replace with ACL
		// TODO send error
		return
	}
	wsh.db.removeAirBeacon(*id)
}

func (wsh *WSRouter) sendMessageToConsumersWithPoint(message JSONChangeMessage, point *quad.Point) {
	if point == nil {
		return
	}
	views := wsh.db.getRectLikeWithPoint(point)
	wsh.sendMessageToConsumers(message, views)
}

func (wsh *WSRouter) sendMessageToConsumers(message JSONChangeMessage, consumers set.Set) {
	for each := range consumers.Iter() {
		if each == nil { // BUG strange I need it under load
			continue
		}
		switch consumer := each.(type) {
		case *View:
			consumer.ws.writeJSON(message)
		case *AirBeacon:
			if wsh.whw != nil {
				msg := HookMessage{AirBeacon: *consumer.id, Message: message}
				wsh.whw.Write(consumer, msg)
			}
		}
	}
}

func (wsh *WSRouter) sendMessageToViews(message JSONChangeMessage, consumers set.Set) {
	for each := range consumers.Iter() {
		if each == nil { // BUG strange I need it under load
			continue
		}
		if view, ok := each.(*View); ok {
			view.ws.writeJSON(message)
		}
	}
}
