package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"geeo.io/GeeoServer/quad"

	bolt "go.etcd.io/bbolt"
)

var (
	poisBucket       = []byte("pois")
	airBeaconsBucket = []byte("airBeacons")
)

type boltDBPersister struct {
	db *bolt.DB
}

// We already have a JSON struct for POIs, for sending over websockets
// but in the future, we'll add ACLs for instance, which must be saved, but not sent over WS
type serializedPOI struct {
	ID         *string
	Pos        *quad.Point
	PublicData map[string]interface{}
	Creator    *string
}
type serializedAirBeacon struct {
	ID         *string
	Pos        *quad.Rect
	PublicData map[string]interface{}
	Creator    *string
}

// TODO replace CreateBucketIfNotExists with the simpler Bucket where appropriate

func newBoltDBPersister(dbfilename string) Persister {

	persister := &boltDBPersister{}

	db, err := bolt.Open(dbfilename, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil || db == nil {
		log.Fatal(err)
	}
	err = db.Update(func(tx *bolt.Tx) error {
		_, errPois := tx.CreateBucketIfNotExists(poisBucket)
		if errPois != nil {
			return errPois
		}
		_, errABs := tx.CreateBucketIfNotExists(airBeaconsBucket)
		if errABs != nil {
			return errABs
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	persister.db = db

	return persister
}

func (p *boltDBPersister) readAirBeaconsInto(geeodb *GeeoDB) error {
	log.Info("Loading AirBeacons from file")
	return p.db.View(func(tx *bolt.Tx) error {

		bucket := tx.Bucket(airBeaconsBucket)
		before := time.Now()
		counter := 0

		parseChannel := make(chan []byte)
		// start a goroutine just for adding to rtree
		go func() {
			defer func() { log.Debug("Persister done inserting Air Beacons to RTree") }()
			for v := range parseChannel {
				obj := serializedAirBeacon{}
				json.Unmarshal(v, &obj)
				rect := quad.NewRect(obj.Pos[0], obj.Pos[1], obj.Pos[2], obj.Pos[3])
				geeodb._loadAirBeacon(*obj.ID, &rect, obj.PublicData, obj.Creator)
			}
		}()

		bucket.ForEach(func(id, v []byte) error {
			parseChannel <- v // will wait until goroutine can accept more before continuing
			counter++
			return nil
		})
		close(parseChannel)
		after := time.Now()

		log.Infof("BoltDB: loaded %d AirBeacons in %fs", counter, after.Sub(before).Seconds())
		return nil
	})
}

func (p *boltDBPersister) readPOIsInto(geeodb *GeeoDB) error {
	log.Info("Loading POIs from file")
	return p.db.View(func(tx *bolt.Tx) error {

		bucket := tx.Bucket(poisBucket)
		before := time.Now()
		counter := 0

		parseChannel := make(chan []byte)
		// start a goroutine just for adding to rtree
		go func() {
			defer func() { log.Debug("Persister done inserting POIs to RTree") }()
			for {
				select {
				case v, more := <-parseChannel:
					if !more {
						return
					}
					obj := serializedPOI{}
					json.Unmarshal(v, &obj)
					point := quad.NewPoint(obj.Pos[0], obj.Pos[1])
					geeodb._loadPOI(*obj.ID, &point, obj.PublicData, obj.Creator)
				}
			}
		}()

		bucket.ForEach(func(id, v []byte) error {
			parseChannel <- v // will wait until goroutine can accept more before continuing
			counter++
			return nil
		})
		close(parseChannel)
		after := time.Now()

		log.Infof("BoltDB: loaded %d points of interest in %fs", counter, after.Sub(before).Seconds())
		return nil
	})
}
func (p *boltDBPersister) persistPOI(poi *POI) error {

	// we're using JSON marshalling: it will be easier to upgrade to a new version of JSON schemas
	obj := serializedPOI{poi.id, poi.GetPoint(), poi.publicData, poi.creator}

	bytes, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	return p.db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(poisBucket)
		if err != nil {
			return err
		}

		return bucket.Put([]byte(*poi.id), bytes)
	})
}
func (p *boltDBPersister) removePOI(poi *POI) error {
	return p.db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(poisBucket)
		if err != nil {
			return err
		}

		return bucket.Delete([]byte(*poi.id))
	})
}

func (p *boltDBPersister) persistAirBeacon(ab *AirBeacon) error {

	// we're using JSON marshalling: it will be easier to upgrade to a new version of JSON schemas
	obj := serializedAirBeacon{ab.id, ab.GetRect(), ab.publicData, ab.creator}

	bytes, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	return p.db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(airBeaconsBucket)
		if err != nil {
			return err
		}

		return bucket.Put([]byte(*ab.id), bytes)
	})
}
func (p *boltDBPersister) removeAirBeacon(ab *AirBeacon) error {
	return p.db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(airBeaconsBucket)
		if err != nil {
			return err
		}

		return bucket.Delete([]byte(*ab.id))
	})
}

func (p *boltDBPersister) close() {
	p.db.Close()
}

// BackupHandleFunc outputs a geeodb backup as a route !
func (p *boltDBPersister) BackupHandleFunc(w http.ResponseWriter, req *http.Request) {
	auth := req.Header.Get("Authorization")

	if WebhookBearerToken == "" || (auth != WebhookBearerToken && req.URL.Query().Get("bearer") != WebhookBearerToken) {
		w.WriteHeader(http.StatusUnauthorized)
		log.Warn("Unauthorized attempt to backup DB")
		return
	}

	err := p.db.View(func(tx *bolt.Tx) error {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", `attachment; filename="geeo.db"`)
		w.Header().Set("Content-Length", strconv.Itoa(int(tx.Size())))
		_, err := tx.WriteTo(w)
		return err
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (p *boltDBPersister) JSONDumpHandleFunc(w http.ResponseWriter, req *http.Request) {
	auth := req.Header.Get("Authorization")

	if WebhookBearerToken == "" || (auth != WebhookBearerToken && req.URL.Query().Get("bearer") != WebhookBearerToken) {
		w.WriteHeader(http.StatusUnauthorized)
		log.Warn("Unauthorized attempt to backup DB")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write([]byte(p.JSONDump()))
}

func (p *boltDBPersister) JSONDump() string {
	log.Debug("Dumping DB to JSON")
	pois := p.JSONDumpBucket(poisBucket)
	abs := p.JSONDumpBucket(airBeaconsBucket)
	return "{\"pois\":" + pois + ",\"airbeacons\":" + abs + "}"
}

func (p *boltDBPersister) JSONDumpBucket(bucketName []byte) string {
	log.Debug("Dumping bucket ", string(bucketName))
	res := "["
	counter := 0
	before := time.Now()
	p.db.View(func(tx *bolt.Tx) error {

		bucket := tx.Bucket(bucketName)

		bucket.ForEach(func(id, v []byte) error {
			if counter != 0 {
				res += ","
			}
			res += string(v)
			counter++
			return nil
		})
		return nil
	})
	after := time.Now()
	log.Infof("BoltDB: dumped %d rows in %fs", counter, after.Sub(before).Seconds())
	return res + "]"
}
