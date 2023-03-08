# Design document

This document briefly describes the ideas Geeo has implemented. It wasn't made of public viewing but may be useful.

## Objects we'll handle

- WebSockets
- View is a view through which we observe the system. It can move (Moveable), can see but can't be seen
- Consumers (interface) have a View which can move
- Producers (interface) Transient/Moveable/Located
- Located (interface)
- maybe Persistent / Transient / Moveable
- PropertiesHolder (interface) for the 3 following objects
- Agents (vs. Producers ?)
- PoI Persistent
- AirBeacons with WebHooks Persistent and detection radius

We could implement an "adaptive max view" which iteratively gathers the largest view.
But it's harder to handle View deconnections (what is the new largest ?)... A regular cleanup could be implemented
This max view could never grow larger than a system-wide largest View param.

## WebSockets

Are used for consuming or producing, or both. When the connection is established
we must know which roles will be assumed.
HTTP Request headers used to pass a JWT Web Token with authenticated identity,
roles, optional initial position (if producer), optional initial view (if consumer),
and capabilities (add PoI or AirBeacon).

## Producers

Send their location on a WebSocket regularly

## Consumers

Have a View and observe changes happening inside. Notified of Agents add/move/delete

## Events

### new WS connection

- check token and set capabilities accordingly (can subscribe, can publish position, can create POIs, can create AirBeacons)
- if Producer, get initial coordinates from token if present
- if Consumer, get initial Views from token if present

### WS connection closed

- notify Agent removal to Views and AirBeacons within reach
- remove Agent position and/or Views

### View from a Subscriber

A single WebSocket can have 0+ views, each with an ID.

- check capabilities to Consume
- upsert Views
- send Views contents

Note: Limit max number of views

### new position from a Producer

- check capabilities to Produce
- upsert agent position
- find views within reach, notify
- find AirBeacons within reach, notify

### new POI

A POI should have a creator and an ACL

- check capabilities to create POI
- find views within reach, notify if `notifyViews`
- find AirBeacons within reach, notify if `notifyAirBeacons`

### delete POI

- check capabilities to delete POI
- find views within reach, notify if `notifyViews`
- find AirBeacons within reach, notify if `notifyAirBeacons`

### new AirBeacon

An AirBeacon should have a creator and an ACL

- check if `notifyViews`
- check capabilities to create AirBeacons
- find agents within reach, notify AirBeacon of agents
- if `notifyViews`, find views within reach, notify them

### delete AirBeacon

- check if `notifyViews`
- check capabilities to delete AirBeacons
- find agents within reach, notify AirBeacon of agents
- if `notifyViews`, find views within reach, notify them

## Location Events

Allow firing an event from an HTTP/WS route, with a location and radius.
Notify agents within radius and Viewss within BB
