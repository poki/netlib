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

##### `quickJoin(settings?: QuickJoinSettings): Promise<LobbyListEntry | undefined>`
Joins the oldest public, password-free, nonempty lobby with available capacity for this game, or creates one atomically. Simultaneous quick-join requests share lobbies without exceeding their capacity. Lobbies with the same creation time are ordered by code.

```typescript
type QuickJoinSettings = Omit<LobbySettings, 'public' | 'password'>;

const lobby = await network.quickJoin({ maxPlayers: 4 });
```

`QuickJoinSettings` reuses the settings accepted by `create()`, except `public` and `password`. Settings apply only when creating a lobby; they do not filter or modify existing lobbies. This includes `codeFormat`, `maxPlayers`, `customData`, and `canUpdateBy`. New lobbies are always public and password-free. Defaults match `create()`, including four players and creator-only update permissions. Empty lobbies are excluded. The method returns `undefined` when the network is closing or not ready, matching `join()`.

Use this instead of separate `list()`, `join()`, and `create()` calls for automatic matchmaking. Existing lobby and connection events work as usual. Callers using `create()` can still create separate lobbies. Deploy signaling server support before using this SDK method; older servers return an error and there is no client-side fallback.

##### `leave(): Promise<void>`
Leaves the current lobby without closing the network.

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

#### Player Counts

##### `getPlayerCounts(filter?: object): Promise<PlayerCounts>`
Returns a server-side snapshot of connected Netlib sessions for this instance's game ID.
Call after the `ready` event.

```typescript
interface PlayerCounts {
  online: number;   // Connected sessions, including yourself
  inLobby: number;  // Connected sessions in public or private lobbies
}

const { online } = await network.getPlayerCounts();
label.textContent = `${online} players online`;
```

The optional filter uses the same MongoDB-style syntax as `list()`, including
lobby custom data fields. It applies only to `inLobby`; `online` always counts all
connected sessions for the game. Matching private lobbies are included, and each
connected session is counted at most once.

```typescript
const { inLobby } = await network.getPlayerCounts({ kind: 'ranked' });
```

Omitting the filter or passing `{}` counts membership in any lobby. Invalid filters
reject the promise.

`inLobby` is a subset of `online`. Counts represent sessions, not unique people;
multiple tabs can count separately. Disconnects are reflected once detected by the
server. Players outside a lobby are not necessarily searching for a game.
The promise rejects if the network is not ready, is closing, or the request fails.

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

#### Lobby Events
- `'lobby'`: Lobby created/joined
- `'left'`: Left lobby
- `'update'`: Lobby settings updated

#### Communication Events
- `'message'`: Received data from peer

## Peer Class

```typescript
interface Peer {
  id: string;          // Unique peer identifier
  maxMessageSize: number | null; // Max bytes per data channel message for this peer
  latency?: {
    last: number;      // Most recent latency measurement (in ms)
    average: number;   // Average latency over time
    jitter: number;    // Variation in latency
    max: number;       // Maximum observed latency
    min: number;       // Minimum observed latency
  };
}
```

##### `peer.maxMessageSize: number | null` (read-only)
Maximum message size, in bytes, that can be sent in a single WebRTC data channel message to this peer.

- `null`: the SCTP transport is not available yet or unsupported in the current environment.
- `0`: no practical message size limit is reported by the transport.
- `> 0`: maximum safe payload size in bytes.

Notes:
- This value is negotiated per peer and can differ across connections.
- Use this to decide when to chunk large payloads.
- If this is `null` or `0` you should use a conservative message size such as `16 * 1024` bytes for large payloads.
