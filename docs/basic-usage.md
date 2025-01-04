# Basis usage


#### 1. Add dependency
First add [@poki/netlib](https://www.npmjs.com/package/@poki/netlib) as a dependency to your project:
```sh
yarn add @poki/netlib
# or using npm:
npm install @poki/netlib
```

#### 2. Import and create a network
Then you can import and create a _Network_ interface:
```js
import { Network } from '@poki/netlib'

const network = new Network('<your-game-id-here>')
```
As game-id you can use any random UUID or use your Poki game-id (which you can find in the url on the gamepage in Poki for Developers).

#### 3. Create/join lobby:

Next one of the clients needs to create a lobby (e.g the player clicks on 'create game'):
```js
network.on('lobby', code => {
  console.log(`lobby was created: ${code}`)
})

network.on('ready', () => {
  network.create()
})
```
Have the player share the code with a friend (using a share url maybe).

And the other client can join the lobby:
```js
network.on('ready', () => {
  network.join(lobby)
})
```
(note: make sure to call create() or join() only when the Network is 'ready')

#### 4. Sending data
```js
network.on('message', (peer, channel, data) => {
  console.log(`received ${data} on channel ${channel} from ${peer.id}`)
})
network.on('connected', peer => {
  console.log(`new peer connected: ${peer.id}`)
  network.send('reliable', peer.id, 'welcome')
})
network.broadcast('unreliable', 'Hello, world!!')
```

#### 5. Handle a peer being disconnected
```js
network.on('disconnected', peer => {
  console.log(`${peer.id} disconnected, their latency was ${peer.latency.average}`)
})
```
