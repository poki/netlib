# The Poki Networking Library &nbsp;&nbsp;[<img src="https://img.shields.io/npm/v/@poki/netlib?color=lightgray">](https://www.npmjs.com/package/@poki/netlib)

<img align="right" src="https://raw.githubusercontent.com/poki/netlib/main/.github/logo.png" width=140>
The Poki Networking Library is a peer-to-peer library for web games, using WebRTC datachannels to provide UDP connections between players directly. Like the Steam Networking Library, but for web.  
Netlib tries to make WebRTC as simple to use as the WebSocket interface (for game development).


## Project Status: alpha

This library is still under heavy development and considered in alpha. The library API can and will change without warning and without bumping the major version. Make sure to get in touch if you want to go live with this netlib.

One missing feature that is next on the roadmap is lobby listing and discovery. Currently you can only connect peers by having players share a lobby code externally.

## Library main advantages:

- **peer-to-peer (p2p)**  
  Clients connected to each other using this netlib will be connected directly without a central server in between (unless using the fallback TURN server). This has three main advantages:
    1. No server costs, there is no server running the game.
    1. No double implementation of the game. You don't need to write your game logic twice (for the client and the server).
    1. When players are living close by the latency is often a lot lower than when connected via a server
- **UDP**  
  Most web games rely on WebSockets or HTTP for communication which is always a TCP connection, but for realtime multiplayer games the UDP protocol is preferred. The main reason is:  
  When one packet is slow or dropped UDP doesn't pause new, already received, packets, this is great for things like position updates. The Poki netlib also supplies a reliable data channel useful for chat or npc spawn events.


## Setup

First add [`@poki/netlib`](https://www.npmjs.com/package/@poki/netlib) as a dependency to your project:
```sh
# either using yarn:
yarn add @poki/netlib
# or using npm:
npm i @poki/netlib
```
Then you can import and create a _Network_ interface:
```js
import { Network } from '@poki/netlib'
const network = new Network('<your-game-id-here>')
```
(any random UUID is a valid game-id, but if your game is hosted at Poki you should use Poki's game-id)

Next up: read the [**basic usage**](./docs/basic-usage.md) guide and make sure to checkout the [**example code**](./example/).


## STUN, TURN & Signaling Backend

The netlib is a peer-to-peer networking library which means players are connected directly to each other and data send between them is never send to a server.  
That said, to setup these connections we need a signaling service. This backend and STUN/TURN servers are hosted by Poki for free. You can however always decide to host the backend yourself.


## Main Contributors

- [Koen Bollen](https://github.com/koenbollen)
- [Erik Dubbelboer](https://github.com/erikdubbelboer)

