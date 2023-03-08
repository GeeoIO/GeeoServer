package main

import (
	"net/http"
	"strconv"

	jwt "github.com/dgrijalva/jwt-go"
)

// DevHelperGetToken sends a valid JWT token in dev mode
// /api/dev/token
func DevHelperGetToken(w http.ResponseWriter, req *http.Request) {

	viewID := req.URL.Query().Get("viewId")
	agentID := req.URL.Query().Get("agId")

	caps := map[string]interface{}{
		"produce":         true,
		"consume":         true,
		"createPOI":       true,
		"createAirBeacon": true,
		"sendEvents":      true,
		"maxView":         [2]float64{360, 180},
		"maxAirBeacon":    [2]float64{360, 180},
		"http":            true,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"viewId":  viewID,
		"agentId": agentID,
		"caps":    caps,
	})
	tokenString, err := token.SignedString([]byte(SecretKey))

	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	w.Header().Set("Content-type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.Itoa(len(tokenString)))

	w.Write([]byte(tokenString))
	log.Debugf("New JWT development token (agent: %s, view: %s): %s", agentID, viewID, tokenString)
}
