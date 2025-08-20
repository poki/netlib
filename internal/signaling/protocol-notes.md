# Signaling Protocol


## Client connects to websocket server and sends:
=> `{"type": "hello", "game": "GameUUID", "id?": "previousPeerID", "lobby?": "previousLobby"}`

## Server responds with:
<= `{"type": "welcome", "id": "newPeerID", "secret": "peerSecret", "protocolVersion": "1.0.0", "warnings?": ["optional warning messages"]}`

## Then the connection idles until client calls create() or join('lobby')

## On create():
=> `{"type": "create"}`
  ### Server responds with:
  <= `{"type": "joined", "lobby": "newLobbyCode"}`

## On join('lobbyCode')
=> `{"type": "join", "lobby": "lobbyCode"}`
  ### Server responds with:
  <= `{"type": "joined", "lobby": "lobbyCode"}`
  ### Server sends connect messages to all peers with the new peer


## Server connecting peerA and peerB:
  ### Sends to peerA
  <= `{"type": "connect", "id": "peerB", "polite": true}`
  ### Sends to peerB
  <= `{"type": "connect", "id": "peerA", "polite": false}`

  ### On connection the clients send back:
  => `{"type": "connected", "id": "otherPeerID"}`

  ### When a client really is disconnected from another peer (e.g webrtc failed multiple times):
  => `{"type": "disconnected", "id": "otherPeerID"}`


## A client closes the network and leaves the lobby:
=> `{"type": "leave", "lobby": "lobbyCode"}`  
** Closes connection

  ### Server sends disconnect messages to all peers with the new peer:
  <= `{"type": "disconnect", "id": "peerA"}`
