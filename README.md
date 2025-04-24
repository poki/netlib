# The Poki Networking Library &nbsp;&nbsp;[<img src="https://img.shields.io/npm/v/@poki/netlib?color=lightgray">](https://www.npmjs.com/package/@poki/netlib)

<img align="right" src="https://raw.githubusercontent.com/poki/netlib/main/.github/logo.png" width=140>
The Poki Networking Library is a peer-to-peer networking library for web games, leveraging WebRTC datachannels to enable direct UDP connections between players. Think of it as the Steam Networking Library for the web, designed to make WebRTC as simple to use as WebSockets for game development.

<p></p>

> [!WARNING]
> This library is still under development and considered a beta. While it's being actively used in production by some games, the API can change. Make sure to get in touch if you want to go live with this so we can keep you up-to-date about changes.

## Features

- **True Peer-to-Peer (P2P) Networking**
  - Direct client-to-client connections without a central game server
  - Lower latency for geographically close players
  - Reduced server costs and infrastructure complexity
  - No need to duplicate game logic between client and server
  - Three main advantages:
    1. No server costs - there is no server running the game
    2. No double implementation - you don't need to write your game logic twice (for the client and the server)
    3. Lower latency - when players are living close by, the latency is often much lower than when connected via a server

- **UDP Performance**
  - Choice between reliable (TCP) and unreliable (UDP) channels
  - Optimized for real-time gaming with minimal latency
  - Perfect for fast-paced multiplayer games
  - Unlike WebSockets or HTTP (which use TCP), UDP doesn't pause new packets when one packet is slow or dropped
  - Includes reliable data channels for critical events like chat messages or NPC spawns

- **Easy to Use**
  - Simple WebSocket-like API
  - Built-in lobby system with filtering
  - Automatic connection management
  - Comprehensive TypeScript support

- **Production Ready**
  - Fallback to TURN servers when direct P2P fails
  - Built-in connection quality monitoring
  - Automatic reconnection handling
  - Secure by default

## Quick Start

1. Install the package:
```sh
yarn add @poki/netlib
# or
npm install @poki/netlib
```

2. Create a network instance:
```js
import { Network } from '@poki/netlib'
const network = new Network('<your-game-id>')
```

3. Create or join a lobby:
```js
// Create a new lobby
network.on('ready', () => {
  network.create()
})

// Or join an existing one
network.on('ready', () => {
  network.join('ed84')
})
```

4. Start communicating:
```js
// Send messages
network.broadcast('unreliable', { x: 100, y: 200 })

// Receive messages
network.on('message', (peer, channel, data) => {
  console.log(`Received from ${peer.id}:`, data)
})
```

For more detailed examples and API documentation:
- [Basic Usage Guide](./docs/basic-usage.md)
- [Example Usage](./example/)

## Roadmap

- [x] Basic P2P connectivity
- [x] Lobby system
- [x] Lobby discovery and filtering
- [ ] WebRTC data compression
- [ ] Connection statistics and debugging tools
- [ ] More extensive documentation
- [ ] API stability

## Architecture

### Network Stack
```
Your Game
    ↓
Netlib API
    ↓
WebRTC DataChannels
    ↓
(STUN/TURN if needed)
    ↓
UDP Transport
```

### Infrastructure Components

#### 1. Signaling Server
- Handles initial peer discovery
- Manages lobby creation and joining
- Facilitates WebRTC connection establishment

#### 2. STUN/TURN Servers
- STUN: Helps peers discover their public IP (by default Google STUN servers)
- TURN: Provides fallback relay when direct P2P fails (when using the Poki hosted version, Cloudflare TURN servers are used)

## Self-Hosting

While Poki provides hosted STUN/TURN and signaling services for free, you can also self-host these components:

1. Set up your own signaling server using the provided Docker image
2. Configure your own STUN/TURN servers
3. Initialize the network with custom endpoints:
```js
const network = new Network('<game-id>', {
  signalingServer: 'wss://your-server.com',
  stunServer: 'stun:your-stun.com:3478',
  turnServer: 'turn:your-turn.com:3478'
})
```

## Contributing

We welcome contributions! Please see our [Contributing Guide](./.github/CONTRIBUTING.md) for details. This project adheres to the [Poki Vulnerability Disclosure Policy](https://poki.com/en/c/vulnerability-disclosure-policy).

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Main Contributors

- [Koen Bollen](https://github.com/koenbollen)
- [Erik Dubbelboer](https://github.com/erikdubbelboer)

