package main

import (
	"errors"
	"fmt"

	jwt "github.com/dgrijalva/jwt-go"
)

var (
	// ErrCantProduce is returned when attempting to produce on a socket that doesn't have that capability
	ErrCantProduce = errors.New("can't produce")

	// ErrCantConsume is returned when attempting to produce on a socket that doesn't have that capability
	ErrCantConsume = errors.New("can't consume")

	// ErrCantUpdatePOI is returned when attempting to produce on a socket that doesn't have that capability
	ErrCantUpdatePOI = errors.New("can't update POI")

	// ErrCantUpdateAirBeacon is returned when attempting to produce on a socket that doesn't have that capability
	ErrCantUpdateAirBeacon = errors.New("can't update AirBeacon")

	// ErrInvalidCapabilities is returned if the set of capabilities is not coherent
	ErrInvalidCapabilities = errors.New("invalid capabilities")

	// ErrInvalidJWTToken is returned if the token isn't valid
	ErrInvalidJWTToken = errors.New("invalid JWT token")
)

// JWTTokenCaps allows specification of Capabilities for this socket
type JWTTokenCaps struct {
	Produce       bool       `json:"produce"`
	Consume       bool       `json:"consume"`
	POI           bool       `json:"createPOI"`
	AirBeacon     bool       `json:"createAirBeacon"`
	SendEvents    bool       `json:"sendEvents"`
	ReceiveEvents bool       `json:"receiveEvents"`
	MaxView       [2]float64 `json:"maxView"`
	MaxAirBeacon  [2]float64 `json:"maxAirBeacon"`
	HTTP          bool       `json:"http"`
}

func (cap *JWTTokenCaps) check() error {
	if !cap.Produce && !cap.Consume && !cap.POI && !cap.AirBeacon {
		return ErrInvalidCapabilities
	}
	return nil
}

// JWTToken describes the format of JWT Tokens
type JWTToken struct {
	jwt.StandardClaims

	AgentID string `json:"agentId"`
	ViewID  string `json:"viewId"`

	Public       map[string]interface{} `json:"publicProperties"`
	Capabilities JWTTokenCaps           `json:"caps"`
}

func parseJWTToken(b64tok string) (*JWTToken, error) {

	var jwttoken JWTToken

	token, err := jwt.ParseWithClaims(b64tok, &jwttoken, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(SecretKey), nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTToken); ok && token.Valid {
		//log.Print(claims)
		if claims.Capabilities.MaxView[0] == 0 {
			claims.Capabilities.MaxView = [2]float64{1, 1} // default to [1,1]
		}
		return claims, nil
	}
	return nil, ErrInvalidJWTToken

}
