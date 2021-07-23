import { Network } from '../lib'

const n = new Network('d0fe1ca1-7fa0-47ed-9469-6c792f68bae0')
;(window as any).n = n

n.on('ready', () => {
  const code = prompt('Lobby code? (empty to create a new one)')
  if (code == null || code === '') {
    n.create()
  } else {
    n.join(code)
  }
})

n.on('lobby', code => {
  console.log('%cLobby: %s', 'font-weight:bold', code)
})

n.on('signalingerror', console.error.bind(console.error))
n.on('rtcerror', console.error.bind(console.error))

n.on('peerconnected', peer => {
  console.log('peer connected!', peer.id, n.size)
  n.on('message', (peer, channel, data) => {
    console.log(`${peer.id} said "${data as string}" via ${channel}`)
  })
  n.broadcast('reliable', `got new peer! ${peer.id}`)
})
