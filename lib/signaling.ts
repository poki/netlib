import Network from './network'
import Peer from './peer'
import { SignalingPacketTypes } from './types'

export default class Signaling {
  private readonly ws: WebSocket
  receivedID?: string

  private readonly connections: Map<string, Peer>

  constructor (private readonly network: Network, peers: Map<string, Peer>, url: string) {
    this.connections = peers

    this.ws = new WebSocket(url)
    this.ws.addEventListener('open', () => {
      this.network.emit('ready')
    })
    this.ws.addEventListener('error', e => {
      this.network.emit('signalingerror', e)
    })
    this.ws.addEventListener('close', () => {
      // TODO: ...
    })
    this.ws.addEventListener('message', ev => {
      this.handleSignalingMessage(ev.data).catch(_ => {})
    })
  }

  close (reason?: string): void {
    if (this.receivedID != null) {
      this.send({
        type: 'leave',
        id: this.receivedID,
        reason: reason ?? 'normal closure'
      })
    }
    this.ws.close()
    this.connections.clear()
  }

  send (packet: SignalingPacketTypes): void {
    if (this.ws.readyState === WebSocket.CONNECTING || this.ws.readyState === WebSocket.OPEN) {
      console.log('sending signaling packet:', packet)
      const data = JSON.stringify(packet)
      this.ws.send(data)
    }
  }

  private async handleSignalingMessage (data: string): Promise<void> {
    try {
      const packet = JSON.parse(data) as SignalingPacketTypes
      console.log('signaling packet received:', packet)
      switch (packet.type) {
        case 'joined':
          if (packet.id === '') {
            throw new Error('missing id on received connect packet')
          }
          if (packet.lobby === '') {
            throw new Error('missing lobby on received connect packet')
          }
          this.receivedID = packet.id
          this.network.emit('lobby', packet.lobby)
          break

        case 'connect':
          if (this.receivedID === packet.id) {
            return
          }
          this.network._addPeer(packet.id, packet.polite)
          break
        case 'disconnected':
          if (this.connections.has(packet.id)) {
            this.connections.get(packet.id)?.close()
          }
          break

        case 'candidate':
        case 'description':
          if (this.connections.has(packet.source)) {
            await this.connections.get(packet.source)?._onSignalingMessage(packet)
          } else {
            console.error('recieved packet for unknown connection (id):', packet.source)
          }
          break
      }
    } catch (e) {
      this.network.emit('signalingerror', e)
    }
  }
}
