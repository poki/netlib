# API Reference

## Network Class

### Constructor

```typescript
new Network(gameId: string, options?: NetworkOptions)
```

#### Parameters
- `gameId`: A unique identifier for your game. Can be a UUID or your Poki game ID.
- `options`: (Optional) Configuration options
  ```typescript
  interface NetworkOptions {
    signalingServer?: string;    // Custom signaling server URL
    stunServer?: string;         // Custom STUN server URL
    turnServer?: string;         // Custom TURN server URL
  }
  ```

### Methods

#### Lobby Management

##### `create(options?: LobbyOptions): Promise<void>`
Creates a new lobby.
```typescript
interface LobbyOptions {
  public?: boolean;              // Make lobby visible in listings
  maxPlayers?: number;           // Maximum number of players allowed
  password?: string;             // Optional password protection
  customData?: any;              // Custom lobby data
  canUpdateBy?: 'anyone' | 'leader' | 'creator'; // Who can update lobby settings
}
```

##### `join(code: string, password?: string): Promise<void>`
Joins an existing lobby.

##### `list(filter?: object): Promise<Lobby[]>`
Lists available lobbies with optional MongoDB-style filtering.
```typescript
interface Lobby {
  code: string;          // Lobby identifier
  playerCount: number;   // Current number of players
  public: boolean;       // Whether lobby is listed
  customData: any;       // Custom lobby data
  createdAt: Date;       // Creation timestamp
  updatedAt: Date;       // Last update timestamp
  leader: string;        // Current leader's peer ID
  canUpdateBy: string;   // Who can update settings
  creator: string;       // Creator's peer ID
  hasPassword: boolean;  // Password protection status
  maxPlayers: number;    // Player limit
}
```

#### Communication

##### `send(channel: string, peerId: string, data: any): void`
Sends data to a specific peer.
- `channel`: Either 'reliable' or 'unreliable'
- `peerId`: Target peer's ID
- `data`: Data to send (string, object, or ArrayBuffer)

##### `broadcast(channel: string, data: any): void`
Sends data to all connected peers.

##### `close(): void`
Disconnects everything and cleans up resources.

### Events

Subscribe to events using `network.on(eventName, callback)`:

#### Connection Events
- `'ready'`: Network is ready to create/join lobbies
- `'error'`: Network error occurred
- `'connected'`: New peer connected
- `'disconnected'`: Peer disconnected
- `'rtcerror'`: WebRTC error occurred (callback receives `RTCErrorEvent`)

#### Lobby Events
- `'lobby'`: Lobby created/joined
- `'leave'`: Left lobby
- `'update'`: Lobby settings updated

#### Communication Events
- `'message'`: Received data from peer

## Peer Class

```typescript
interface Peer {
  id: string;          // Unique peer identifier
  latency?: {
    last: number;      // Most recent latency measurement (in ms)
    average: number;   // Average latency over time
    jitter: number;    // Variation in latency
    max: number;       // Maximum observed latency
    min: number;       // Minimum observed latency
  };
}
```
