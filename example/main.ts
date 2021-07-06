import { Network } from '../lib'

const n = new Network('d0fe1ca1-7fa0-47ed-9469-6c792f68bae0')

n.on('lobby', code => {
  console.log(code)
})

n.on('signalingerror', console.error, console.error)
n.on('rtcerror', console.error, console.error)

n.on('peerconnected', peer => {
  console.log('peer connected!', peer.id, n.size)
})

n.on('ready', () => {
  n.join('testlobby')
})
