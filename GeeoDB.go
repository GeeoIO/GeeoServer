package main

import (
	"errors"
	"expvar"
	"net/http"

	"sync"

	"geeo.io/GeeoServer/quad"
	set "github.com/deckarep/golang-set"
)

var (
	// ErrNotImplemented is returned for actions without an implementation
	ErrNotImplemented = errors.New("Not Implemented")
	// ErrAgentNotFound is returned when agent can't be found
	ErrAgentNotFound = errors.New("Agent doesn't exist")
	// ErrViewNotFound is returned when View can't be found
	ErrViewNotFound = errors.New("View doesn't exist")
	// ErrAgentExistsAlready is returned when agent exists already
	//ErrAgentExistsAlready = errors.New("Agent already exists")
	// ErrPoiExistsAlready is returned when POI exists already
	//ErrPoiExistsAlready = errors.New("POI already exists")

	numPOIs, numABs, numViews, numAgents *expvar.Int
)

// Persister is the interface you should implement to provide persistence to GeeoDB
type Persister interface {
	readPOIsInto(geeodb *GeeoDB) error
	readAirBeaconsInto(geeodb *GeeoDB) error
	persistPOI(poi *POI) error
	removePOI(poi *POI) error
	persistAirBeacon(ab *AirBeacon) error
	removeAirBeacon(ab *AirBeacon) error
	close()
	BackupHandleFunc(w http.ResponseWriter, req *http.Request)
	JSONDumpHandleFunc(w http.ResponseWriter, req *http.Request)
	// TODO LATER chain of Persisters
}

// JSONMessageAble is the interface for objects which can produce JSON change messages
type JSONMessageAble interface {
	enterLeaveMessage(enter bool) JSONChangeMessage
}

// GeeoDB handles all the insert/update/delete operations
type GeeoDB struct {
	agents    map[string]*Agent
	v         map[string]*View
	ab        map[string]*AirBeacon
	pois      map[string]*POI
	persister Persister

	tree quad.Quad
	sync.RWMutex
}

// NewGeeoDB creates a new GeeoDB
func NewGeeoDB(pers Persister, depth int) *GeeoDB {

	// set min depth for quad tree... that's a global actually
	quad.MinDepth = depth
	newdb := &GeeoDB{
		agents: make(map[string]*Agent),
		v:      make(map[string]*View),
		ab:     make(map[string]*AirBeacon),
		pois:   make(map[string]*POI),

		persister: pers,
		tree:      quad.NewQuad(),
	}

	newdb.persister.readPOIsInto(newdb)
	newdb.persister.readAirBeaconsInto(newdb)

	numPOIs = expvar.NewInt("num_poi")
	numABs = expvar.NewInt("num_airbeacon")
	numAgents = expvar.NewInt("num_agent")
	numViews = expvar.NewInt("num_view")

	numPOIs.Set(int64(len(newdb.pois)))
	numABs.Set(int64(len(newdb.ab)))
	numAgents.Set(int64(len(newdb.agents)))
	numViews.Set(int64(len(newdb.v)))

	return newdb
}

func (db *GeeoDB) addAgent(id string, conn *wsConn, pub map[string]interface{}) *Agent {
	newagent := &Agent{ID: &id, ws: conn, publicData: pub}

	db.Lock()
	defer db.Unlock()
	if _, exists := db.agents[id]; exists {
		log.Error("[BUG] should not attempt to add existing agent ", id)
	}

	db.agents[id] = newagent

	numAgents.Add(1)

	return newagent
}

func (db *GeeoDB) removeAgent(id string) {
	db.Lock()
	defer db.Unlock()

	agent, ok := db.agents[id]
	if ok {
		if agent.GetPoint() != nil {
			db.tree.RemovePoint(agent)
		}
		delete(db.agents, id)
		numAgents.Add(-1)
		return
	}
	log.Warn("should not remove inexisting agent ", id)
}

func (db *GeeoDB) updateAgentPosition(id string, pos *quad.Point) *quad.Point {
	db.Lock()
	defer db.Unlock()

	agent, ok := db.agents[id]
	var oldPosition *quad.Point
	if ok {
		oldPosition = agent.GetPoint()
		if oldPosition == nil {
			agent.SetPoint(pos)
			db.tree.AddPoint(agent)
			return oldPosition
		}
		db.tree.RemovePoint(agent)
		agent.SetPoint(pos)
		db.tree.AddPoint(agent)
		return oldPosition
	}
	log.Error("Agent not found when updating position ", id)
	return nil
}

func (db *GeeoDB) getPointLikeIn(pos *quad.Rect) set.Set {
	db.RLock()
	defer db.RUnlock()

	res := db.tree.GetPointsIn(pos)

	resultset := set.NewThreadUnsafeSet() // no need for thread safety
	for _, each := range res {
		resultset.Add(each)
	}
	return resultset
}

func (db *GeeoDB) addView(id string, conn *wsConn) *View {
	v := &View{id: &id, ws: conn}

	db.Lock()
	defer db.Unlock()

	numViews.Add(1)

	db.v[id] = v
	return v
}

func (db *GeeoDB) removeView(id string) {
	db.Lock()
	defer db.Unlock()

	v, ok := db.v[id]

	if ok {
		if v.GetRect() == nil {
			return
		}
		db.tree.RemoveRect(v)
		delete(db.v, id)
		numViews.Add(-1)
	}
}

func (db *GeeoDB) updateViewPosition(id string, pos *quad.Rect) *quad.Rect {
	db.Lock()
	defer db.Unlock()
	v, ok := db.v[id]

	var oldPosition *quad.Rect
	if ok {
		oldPosition = v.GetRect()
		if oldPosition == nil {
			v.SetRect(pos)
			db.tree.AddRect(v)
			return oldPosition
		}
		db.tree.MoveRect(v, pos)
	}
	return oldPosition
}

func (db *GeeoDB) addAirBeacon(id string, pos *quad.Rect, publicData map[string]interface{}, creator *string) *AirBeacon {

	db.Lock()
	defer db.Unlock()

	ab := &AirBeacon{id: &id, rect: pos, publicData: publicData, creator: creator}
	db.ab[id] = ab
	db.tree.AddRect(ab)
	db.persister.persistAirBeacon(ab)

	numABs.Add(1)

	return ab
}

// _loadPOI is really used to batch load POIs into the db without blocking or checks
func (db *GeeoDB) _loadAirBeacon(id string, pos *quad.Rect, publicData map[string]interface{}, creator *string) *AirBeacon {
	airBeacon := &AirBeacon{
		id:         &id,
		publicData: publicData,
		creator:    creator,
	}
	airBeacon.SetRect(pos)

	db.Lock()
	defer db.Unlock()

	db.ab[id] = airBeacon
	db.tree.AddRect(airBeacon)

	return airBeacon
}

func (db *GeeoDB) removeAirBeacon(id string) {
	db.Lock()
	defer db.Unlock()

	ab, ok := db.ab[id]

	if ok {
		db.tree.RemoveRect(ab)
		db.persister.removeAirBeacon(ab)
		numABs.Add(-1)
	}
}

func (db *GeeoDB) getRectLikeWithPoint(pos *quad.Point) set.Set {

	if pos == nil {
		log.Error("getRectLikeWithPoint for nil. This shouldn't happen")
	}
	db.RLock()
	defer db.RUnlock()

	res := db.tree.GetRectsWithPoint(pos, quad.AcceptAll)
	return res
}
func (db *GeeoDB) getViewsWithPoint(pos *quad.Point) set.Set {

	if pos == nil {
		log.Error("getViewsWithPoint for nil. This shouldn't happen")
	}
	db.RLock()
	defer db.RUnlock()

	res := db.tree.GetRectsWithPoint(pos, func(each quad.RectLike) bool {
		_, ok := each.(*View)
		return ok
	})
	return res
}

// _loadPOI is really used to batch load POIs into the db without blocking or checks
func (db *GeeoDB) _loadPOI(id string, pos *quad.Point, publicData map[string]interface{}, creator *string) *POI {
	poi := &POI{
		id:         &id,
		publicData: publicData,
		creator:    creator,
	}
	poi.SetPoint(pos)

	db.Lock()
	defer db.Unlock()

	db.pois[id] = poi
	db.tree.AddPoint(poi)
	return poi
}

func (db *GeeoDB) addPOI(id string, pos *quad.Point, publicData map[string]interface{}, creator *string) *POI {
	if _, exists := db.pois[id]; exists {
		log.Error("[BUG] should not attempt to add existing poi ", id)
	}

	poi := db._loadPOI(id, pos, publicData, creator)
	db.persister.persistPOI(poi)

	numPOIs.Add(1)

	return poi
}

func (db *GeeoDB) removePOI(poi *POI) {
	db.Lock()
	defer db.Unlock()

	_, ok := db.pois[*poi.id]

	if ok {
		delete(db.pois, *poi.id)
		db.tree.RemovePoint(poi)
		db.persister.removePOI(poi)

		numPOIs.Add(-1)
	} else {
		log.Warn("should not attempt to remove missing poi ", poi.id)
	}
}
