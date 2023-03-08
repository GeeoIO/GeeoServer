# Geeo Server

Geeo is a reactive geo database.

- it's reactive because network clients listen to updates being made in a geo-location way: they specify which map area they're interested in, and will receive updates in real time
- it's a geo database because you can only store geo data in it (Points of Interest, POI)
- it's also a kind of geo-rooster, where each network client can send his geo location, and appear in the database transiently
- it's also capable of offline tracking through webhooks: in this case you're getting updates that happen in one area, through webhooks

It's an in-memory database, but persistent data is persisted to a BoltDB file.

Geeo is designed to scale vertically only, there's no provision for sharding, but it supports a large number of concurrent real time connections. Geeo was tested with more than 10k clients on a single cheap 10$ instance. It was a design goal.

## What can I do with it

Completely new use cases are possible with Geeo. It was designed with Pokemon-Go in mind but it's even more capable. It's made for mobile, geo location, real time use cases. Imagine any app where people who are using the app can track where the other users are, and where things (POI) are.

Imagine a game where players must move physically to move in the game, and they see other players around them, and can build stuff in the game(POIs), and set trip wires (air beacons).

Imagine an app where you can find places to go to, but you also see which people are going there, and place-owners can see who's around their place to send offers in real time.

It's more than just storing geo location data (Google maps), it's more than just sending updates in real time (Uber), it's more than knowing who's around you (beacons in shops): it's all of that.

Use Geeo responsibly though. Don't encourage stalking.

## Concepts

Geo coordinates are longitude-latitude based (Lon/Lat ([-180, 180], [-90, 90]). There's no support for altitude and coordinates don't wrap around the earth (not supported).

A Point Of Interest, POI, is a single piece of data stored attached to a geo location. It's stored persistently.

An Agent is a websocket client which sends his coordinates in real time. It's stored transiently in Geeo, in a rooster fashion.

A View is a window open to a part of the coordinates space which receives notifications when somethings happens inside of it. If Agents enter/exit or move within the window, they will receive notifications. They will receive notifications of new/removed POIs too. A View is attached to a network client.

An AirBeacon is like a view, but it's stored persistently, and it will receive updates through a webhook. It can be used by another backend to react to Geeo events like people entering/leaving a zone.

Network protocol is Websocket and JSON based. See the other repositories for JS and C# clients.

## Authentication / Authorization

There's no direct support in Geeo for user creation etc... Instead it's using JWT tokens with a signature. These tokens must be created outside of Geeo (by the backend that can authenticate users) and be passed to Geeo to connect. They hold the identity of users, and their capabilities.

This simple system is enough for Geeo as it's very unlikely that Geeo will be used as the only backend of an app.

All websocket communication is encrypted with SSL. Certificates are issued automatically with Let's Encrypt.

## Building

`go build .` will create a `GeeoServer` binary. Or use Docker with the provided Dockerfile (linux/Intel).

## Running

`GeeoServer -ssl -sslhost demo.geeo.io -dev` starts a dev server with TLS support for https://demo.geeo.io and wss://demo.geeo.io

Env variables used for webhooks handling:

- `WEBHOOK_URL` is the URL that will receive the POST HTTP requests
- `WEBHOOK_HEADERS` is a JSON-encoded map of `{header:value}` sent to the webhook
- `WEBHOOK_BEARER` is a token sent as an `Authorization: Bearer Token` header

eg. `env WEBHOOK_URL=https://requestb.in/rgorydrg WEBHOOK_BEARER=delmenow WEBHOOK_HEADERS='{"apikey":"blah","apisecret":"bla"}' ./GeeoServer`

## Websocket

When connecting to the websocket endpoint, pass a `X-GEEO-TOKEN` header with a JWT token signed with your key.

This token must contain the following attributes

```
viewId: 'your view ID',
agentId: 'your agent ID',
publicProperties: {prop1: value, prop2: [array], prop3: {object: true}},
caps: {
	produce: false, 		// allow sending coordinates
	consume: true,  		// allow seeing others
	createPOI: true,		// allow creation of POIs
	createAirBeacon: true,	// allow creation of Air Beacons
	sendEvents: true,		// allow sending events
	receiveEvents: true,	// allow receiving events
	maxView: [15,15]	// max size of view
	maxAirBeacon: [15,15]	// max size of air beacon
}
```

## Messages sent

You'll send messages to the websocket server as simple JSON objects.

They follow the form
```
{
	command: params,
	...
}
```
to allow passing more than one command at a time.

### AgentPosition

Sending
```
{
	agentPosition: [0.5, 0.5]
}
```
will put your agent (whose id was passed in the JWT token) at (0.5,0.5). Views will be notified of this change.

### View

Sending
```
{
	viewPosition: [0,0,10,10]
}
```
will set or move your view to (0,0)x(10,10). You'll receive events with the View contents and notifications 
as soon as changes over time.

As seen above, you can combine different commands in a single message :
```
{
	agentPosition: [0.5, 0.5]
	viewPosition: [0,0,10,10]
}
```
will perform both a move of your agent, and of your view.

## Messages received

You will normally receive arrays of objects through the websocket.
Erros are an exception, and you'll receive a single object on the websocket.

### Points of Interest

```
{
	poi_id: 'a POI id', 
	pos: [ 9, 9 ], 
	publicData: {any:"thing"}
	entered: true,
	left: false,
	creator: 'a creator id'
}
```

If the object has a `poi_id` property, it's a Point of Interest.
`entered` will be true if the object has just appeared in your view.
`left` will be true if the object left your view.
If `left` is true, `pos`, `publicData` and `entered` are always missing as they are no longer necessary.

### Agents

```
{
	agent_id: 'an agent Id',
	pos: [ 0.5, 0.5 ],
	publicData: { any: "thing" },
	entered: true,
	left: false
}
```

If the object has an `agent_id` property, it's an Agent.
`entered` and `left` will tell you if the object entered or left your View.
If `left` is true, `pos`, `publicData` and `entered` are always missing as they are no longer necessary.

If the Agent object has no entered/left/publicData field, it's just a move event :
```
{
	agent_id: 'an agent Id',
	pos: [ 3.2772948326607665, 0.8537365413192854 ] 
}
```
Geeo only sends you information once to save bandwidth. In this case (agent_id == 'an agent Id'), it has already sent you
the `publicData` for this agent when it appeared in your View: there's no need for a resend.

### Errors

Errors are sent as an object with a propery named `error` and an optional `message` property.

### Webhook

You can specify a webhook to be called to handle AirBeacon notifications. A single webhook receives enter/leave messages for all airbeacons, batched, every second at most.

Simply use the `WEBHOOK_URL` param to specify the URL Geeo.io will `POST` to.
If you need additional HTTP headers, use `WEBHOOK_HEADERS` to transmit a JSON encoded map of header-value strings.
Finally, if you want the webhook to verify that POSTs are coming from Geeo, have it check the `Authorization: Bearer` header, it should contain the value specified in `WEBHOOK_BEARER`.

The webhook will receive enter/leave messages for Agents and POIs.

Messages are a JSON encoded array of message. Each message has the structure `{beacon_id, message}` where message is similar to messages received on websockets. Example: `[{"beacon_id":"airbeacon 1","message":{"agent_id":"chrisAgent67","left":true}}]`.

The array can contain any number of messages for many beacons, it's ordrered by event time, and sent at most once per second.

### HTTP

2 routes allow the creation/deletion of POIs and AirBeacons :

The `/api/v1/POI` and `/api/v1/airbeacon` endpoints accept POST and DELETE requests similar to the websocket requests (same message format).

They require the same JWT token header (or url parameter) as websockets. The JWT token must include the `http` grant to allow HTTP access. HTTP access doesn't check poi and airbeacon's creator, allowing to remove any poi or airbeacon.
