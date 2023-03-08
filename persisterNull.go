package main

import "net/http"

type nullPersister struct{}

func newNullPersister() Persister {
	return &nullPersister{}
}

func (p *nullPersister) readPOIsInto(geeodb *GeeoDB) error {
	return nil
}
func (p *nullPersister) readAirBeaconsInto(geeodb *GeeoDB) error {
	return nil
}
func (p *nullPersister) persistPOI(poi *POI) error {
	return nil
}
func (p *nullPersister) removePOI(poi *POI) error {
	return nil
}
func (p *nullPersister) persistAirBeacon(ab *AirBeacon) error {
	return nil
}
func (p *nullPersister) removeAirBeacon(ab *AirBeacon) error {
	return nil
}

func (p *nullPersister) close() {}

func (p *nullPersister) BackupHandleFunc(w http.ResponseWriter, req *http.Request) {
}
func (p *nullPersister) JSONDumpHandleFunc(w http.ResponseWriter, req *http.Request) {
}
