# Basic Usage Guide

## Getting Started

#### 1. Add dependency
First add [@poki/netlib](https://www.npmjs.com/package/@poki/netlib) as a dependency to your project:
```sh
yarn add @poki/netlib
# or using npm:
npm install @poki/netlib
```

#### 2. Network Setup
Import and create a _Network_ interface:
```js
import { Network } from '@poki/netlib'

const network = new Network('<your-game-id-here>')
```
The game-id can be either:
- Your Poki game-id (found in the URL on your game page in Poki for Developers)
- Any random UUID for development/testing

#### 3. Managing Lobbies

##### Creating a Lobby
When a player wants to create a new game:
```js
// Listen for lobby creation
network.on('lobby', code => {
  console.log(`Lobby created with code: ${code}`)
  // You can now share this code with other players
  // e.g., display it in your UI or generate a share URL
})

// Wait for network to be ready before creating
network.on('ready', () => {
  network.create({
    public: true,   // Make lobby visible in listings
    maxPlayers: 4,  // Optional: limit number of players
  })
})
```

##### Joining a Lobby
When a player wants to join an existing game:
```js
network.on('ready', () => {
  network.join(lobbyCode)
})
```

#### 4. Communication

##### Sending Messages
The library provides two types of channels:
- 'reliable': Guaranteed delivery, like TCP (good for chat, game state)
- 'unreliable': Fast but may drop packets, like UDP (good for position updates)

```js
// Send to specific peer
network.send('reliable', peerId, 'Hello!')

// Broadcast to all peers
network.broadcast('unreliable', { x: 100, y: 200 })

// Send binary data
const buffer = new Uint8Array([1, 2, 3])
network.send('reliable', peerId, buffer)
```

##### Receiving Messages
```js
network.on('message', (peer, channel, data) => {
  console.log(`Received on ${channel} from ${peer.id}:`, data)
  
  // Check message channel type
  if (channel === 'reliable') {
    handleReliableMessage(data)
  } else {
    handleUnreliableMessage(data)
  }
})
```

#### 5. Managing Peers

##### Peer Connection Events
```js
// New peer joins
network.on('connected', peer => {
  console.log(`${peer.id} connected`)
  // You might want to send them the current game state
})

// Peer leaves/disconnects
network.on('disconnected', peer => {
  console.log(`${peer.id} disconnected`)
  // Clean up any game state for this peer
})
```

#### 6. Listing Lobbies
You can list available lobbies using the `list` function. This function supports filtering using MongoDB-style filters:

```js
// List all public lobbies
network.list({}).then(lobbies => {
  console.log('Available lobbies:', lobbies)
})

// Filter lobbies with specific criteria
network.list({
  $and: [
    { playerCount: { $gte: 2, $lt: 10 } }, // Not empty or full
    { password: "" },                      // No password set
    { 
      $or: [                               // Match specific maps
        { map: { $regex: "aztec" } },
        { map: { $regex: "nuke" } }
      ]
    }
  ]
}).then(lobbies => {
  console.log('Filtered lobbies:', lobbies)
})
```

The filter supports the following operators:
- Basics: `$eq`, `$ne`, `$gt`, `$gte`, `$lt`, `$lte`, `$regex`, `$exists`
- Logical operators: `$and`, `$or`, `$not`, `$nor`
- Array operators: `$in`, `$nin`, `$elemMatch`
- Field comparison: `$field` (to compare fields with each other)

Each lobby in the result includes:
- `code`: The lobby code
- `playerCount`: Number of players in the lobby
- `public`: Whether the lobby is public
- `customData`: Any custom data set by the lobby creator
- `createdAt`: When the lobby was created
- `updatedAt`: When the lobby was last updated
- `leader`: The current leader of the lobby
- `canUpdateBy`: Who can update the lobby settings
- `creator`: The peer who created the lobby
- `hasPassword`: Whether the lobby has a password
- `maxPlayers`: Maximum number of players allowed in the lobby

## Best Practices

1. **Error Handling**
```js
network.on('error', error => {
  console.error('Network error:', error)
  // Handle or display error to user
})
```

2. **Wait for ready**
```js
network.on('ready', state => {
  // Only do network.list/create/join here after this
})
```

3. **Clean Up**
```js
// When your game stops
network.close()
```

4. **Latency Optimization**
- Use 'unreliable' channel for frequent updates (positions, rotations)
- Use 'reliable' channel for important game state
- Monitor peer.latency values to potentially warn players of high latencies
