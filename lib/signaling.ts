import { EventEmitter } from 'eventemitter3'
import Network from './network'
import Peer from './peer'
import { SignalingPacketTypes } from './types'

interface SignalingListeners {
  credentials: (data: SignalingPacketTypes) => void | Promise<void>
}

export default class Signaling extends EventEmitter<SignalingListeners> {
  private readonly url: string
  private ws: WebSocket
  private reconnectAttempt: number = 0
  private reconnecting: boolean = false
  receivedID?: string
  receivedSecret?: string
  currentLobby?: string

  private readonly connections: Map<string, Peer>

  private readonly replayQueue: Map<string, SignalingPacketTypes[]>

  private readonly requests: Map<string, RequestHandler> = new Map()

  constructor (private readonly network: Network, peers: Map<string, Peer>, url: string) {
    super()

    this.url = url
    this.connections = peers
    this.replayQueue = new Map()

    this.ws = this.connect()
  }

  private connect (): WebSocket {
    const ws = new WebSocket(this.url)
    const onOpen = (): void => {
      this.reconnectAttempt = 0
      this.reconnecting = false
      this.send({
        type: 'hello',
        game: this.network.gameID,
        id: this.receivedID,
        secret: this.receivedSecret,
        lobby: this.currentLobby
      })
    }
    const onError = (e: Event): void => {
      const error = new SignalingError('socket-error', 'unexpected websocket error', e)
      this.network._onSignalingError(error)
      if (ws.readyState === WebSocket.CLOSED) {
        this.reconnecting = false
        ws.removeEventListener('open', onOpen)
        ws.removeEventListener('error', onError)
        ws.removeEventListener('message', onMessage)
        ws.removeEventListener('close', onClose)
        this.reconnect()
      }
    }
    const onMessage = (ev: MessageEvent): void => {
      this.handleSignalingMessage(ev.data).catch(_ => {})
    }
    const onClose = (): void => {
      if (!this.network.closing) {
        const error = new SignalingError('socket-error', 'signaling socket closed')
        this.network._onSignalingError(error)
      }
      ws.removeEventListener('open', onOpen)
      ws.removeEventListener('error', onError)
      ws.removeEventListener('message', onMessage)
      ws.removeEventListener('close', onClose)
      this.reconnect()
    }
    ws.addEventListener('open', onOpen)
    ws.addEventListener('error', onError)
    ws.addEventListener('message', onMessage)
    ws.addEventListener('close', onClose)
    return ws
  }

  private reconnect (): void {
    if (this.reconnecting || this.network.closing) {
      return
    }

    this.close()

    this.requests.forEach((r) => r.reject(new SignalingError('socket-error', 'signaling socket closed')))
    this.requests.clear()

    if (this.reconnectAttempt > 42) {
      this.network.emit('failed')
      this.network._onSignalingError(new SignalingError('socket-error', 'giving up on reconnecting to signaling server'))
      return
    }
    void this.event('signaling', 'attempt-reconnect')
    this.reconnecting = true
    setTimeout(() => {
      this.ws = this.connect()
    }, Math.random() * 100 * this.reconnectAttempt)
    this.reconnectAttempt += 1
  }

  close (): void {
    this.ws.close()
  }

  async request (packet: SignalingPacketTypes): Promise<SignalingPacketTypes> {
    return await new Promise<SignalingPacketTypes>((resolve, reject) => {
      if (this.ws.readyState !== WebSocket.OPEN) {
        reject(new SignalingError('socket-error', 'signaling socket not open'))
      }
      const rid = Math.random().toString(36).slice(2)
      packet.rid = rid
      this.network.log('requesting signaling packet:', packet.type)
      const data = JSON.stringify(packet)
      this.ws.send(data)
      this.requests.set(rid, { resolve, reject })
    })
  }

  send (packet: SignalingPacketTypes): void {
    if (this.ws.readyState === WebSocket.OPEN) {
      this.network.log('sending signaling packet:', packet.type)
      const data = JSON.stringify(packet)
      this.ws.send(data)
    }
  }

  private async handleSignalingMessage (data: string): Promise<void> {
    try {
      const packet = JSON.parse(data) as SignalingPacketTypes
      this.network.log('signaling packet received:', packet.type)
      if (packet.rid !== undefined) {
        const request = this.requests.get(packet.rid)
        if (request != null) {
          this.requests.delete(packet.rid)
          if (packet.type === 'error') {
            request.reject(new SignalingError('server-error', packet.message))
          } else {
            request.resolve(packet)
          }
        }
      }
      switch (packet.type) {
        case 'error':
          {
            const error = new SignalingError('server-error', packet.message)
            this.network._onSignalingError(error)
            if (packet.code === 'missing-recipient' && packet.error?.recipient !== undefined) {
              const id = packet.error?.recipient
              if (this.connections.has(id)) {
                this.network.log('cleaning up missing recipient', id)
                this.connections.get(id)?.close('missing-recipient')
              }
            }
          }
          break

        case 'welcome':
          if (this.receivedID !== undefined) {
            this.network.log('signaling reconnected')
            this.network.emit('signalingreconnected')
            return
          }
          if (packet.id === '') {
            throw new Error('missing id on received welcome packet')
          }
          this.receivedID = packet.id
          this.receivedSecret = packet.secret
          this.network.emit('ready')
          this.network._prefetchTURNCredentials()
          break

        case 'joined':
          if (packet.lobby === '') {
            throw new Error('missing lobby on received connect packet')
          }
          this.currentLobby = packet.lobby
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
        case 'disconnect':
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
        case 'ping':
          break
      }
    } catch (e) {
      const error = new SignalingError('unknown-error', e as string)
      this.network._onSignalingError(error)
    }
  }

  async event (category: string, action: string, data?: {[key: string]: string}): Promise<void> {
    return await new Promise(resolve => {
      setTimeout(() => {
        this.send({
          type: 'event',
          game: this.network.gameID,
          lobby: this.currentLobby,
          peer: this.network.id,
          category,
          action,
          data
        })
        resolve()
      }, 0)
    })
  }
}

interface RequestHandler {
  resolve: (data: SignalingPacketTypes) => void
  reject: (reason?: any) => void
}

export class SignalingError {
  /**
   * @internal
   */
  constructor (public type: 'unknown-error' | 'socket-error' | 'server-error', public message: string, public event?: Event) {
  }

  public toString (): string {
    return `[${this.type}: ${this.message}]`
  }
}
