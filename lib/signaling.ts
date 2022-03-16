import { EventEmitter } from 'eventemitter3'
import Network from './network'
import Peer from './peer'
import { SignalingPacketTypes } from './types'

interface SignalingListeners {
  credentials: (data: SignalingPacketTypes) => void | Promise<void>
}

export default class Signaling extends EventEmitter<SignalingListeners> {
  private readonly ws: WebSocket
  receivedID?: string

  private readonly connections: Map<string, Peer>

  private readonly replayQueue: Map<string, SignalingPacketTypes[]>

  constructor (private readonly network: Network, peers: Map<string, Peer>, url: string) {
    super()

    this.connections = peers
    this.replayQueue = new Map()

    this.ws = new WebSocket(url)
    this.ws.addEventListener('open', () => {
      this.network.emit('ready')
      this.network._prefetchTURNCredentials()
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
    if (this.receivedID != null) {
      this.send({
        type: 'leave',
        id: this.receivedID,
        reason: reason ?? 'normal closure'
      })
    }
    this.ws.close()
  }

  send (packet: SignalingPacketTypes): void {
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
          await this.network._addPeer(packet.id, packet.polite)
          for (const p of this.replayQueue.get(packet.id) ?? []) {
            await this.connections.get(packet.id)?._onSignalingMessage(p)
          }
          this.replayQueue.delete(packet.id)
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
            const queue = this.replayQueue.get(packet.source) ?? []
            queue.push(packet)
            this.replayQueue.set(packet.source, queue)
          }
          break
        case 'credentials':
          this.emit('credentials', packet)
          break
      }
    } catch (e) {
      this.network.emit('signalingerror', e)
    }
  }
}
