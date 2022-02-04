import Network from './network'
import Peer from './peer'
import { SignalingPacketTypes } from './types'

export default class Signaling {
  private ws: WebSocket | null = null
  private readonly url: string
  receivedID?: string

  private readonly connections: Map<string, Peer>

  constructor (private readonly network: Network, peers: Map<string, Peer>, url: string) {
    this.url = url
    this.connections = peers
  }

  connect (): void {
    this.ws = new WebSocket(this.url)
    this.ws.addEventListener('open', () => {
      this.network.emit('ready')
    })
    this.ws.addEventListener('error', e => {
      this.network.emit('signalingerror', e)
      if (this.network.listenerCount('signalingerror') === 0) {
        console.error('signallingerror not handled:', e)
      }
    })
    this.ws.addEventListener('close', () => {
      this.network.close('signaling websocket closed')
    })
    this.ws.addEventListener('message', ev => {
      this.handleSignalingMessage(ev.data).catch(_ => {})
    })
  }

  close (reason?: string): void {
    if (this.ws === null) return
    if (this.receivedID != null) {
      this.send({
        type: 'leave',
        id: this.receivedID,
        reason: reason ?? 'normal closure'
      })
    }
    this.ws.close()
    this.ws = null
  }

  send (packet: SignalingPacketTypes): void {
    if (this.ws === null) return
    if (this.ws.readyState === WebSocket.CONNECTING || this.ws.readyState === WebSocket.OPEN) {
      this.network.log('sending signaling packet:', packet.type)
      const data = JSON.stringify(packet)
      this.ws.send(data)
    }
  }

  private async handleSignalingMessage (data: string): Promise<void> {
    try {
      const packet = JSON.parse(data) as SignalingPacketTypes
      this.network.log('signaling packet received:', packet.type)
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
            return // Skip self
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
            if (!this.network.closing) {
              console.error(this.network.id, 'recieved packet for unknown connection (id):', packet.source)
            }
          }
          break
      }
    } catch (e) {
      this.network.emit('signalingerror', e)
    }
  }
}
