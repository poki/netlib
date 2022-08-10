import { EventEmitter } from 'eventemitter3'

import { DefaultDataChannels, DefaultRTCConfiguration, DefaultSignalingURL } from '.'
import { PeerConfiguration } from './types'
import Signaling, { SignalingError } from './signaling'
import Peer from './peer'
import Credentials from './credentials'

interface NetworkListeners {
  ready: () => void | Promise<void>
  lobby: (code: string) => void | Promise<void>
  connecting: (peer: Peer) => void | Promise<void>
  connected: (peer: Peer) => void | Promise<void>
  reconnecting: (peer: Peer) => void | Promise<void>
  reconnected: (peer: Peer) => void | Promise<void>
  disconnected: (peer: Peer) => void | Promise<void>
  signalingreconnected: () => void | Promise<void>
  failed: () => void | Promise<void>
  message: (peer: Peer, channel: string, data: string | Blob | ArrayBuffer | ArrayBufferView) => void | Promise<void>
  close: (reason?: string) => void | Promise<void>
  rtcerror: (e: Event) => void | Promise<void> // TODO: Figure out how to make this e type be RTCErrorEvent
  signalingerror: (e: SignalingError) => void | Promise<void>
}

export default class Network extends EventEmitter<NetworkListeners> {
  private _closing: boolean = false
  public readonly peers: Map<string, Peer>
  private readonly signaling: Signaling
  private readonly credentials: Credentials
  public dataChannels: {[label: string]: RTCDataChannelInit} = DefaultDataChannels

  public log: (...data: any[]) => void = (...args: any[]) => {} // console.log

  private readonly unloadListener: () => void

  constructor (public readonly gameID: string, private readonly rtcConfig: PeerConfiguration = DefaultRTCConfiguration, signalingURL: string = DefaultSignalingURL) {
    super()
    this.peers = new Map<string, Peer>()
    this.signaling = new Signaling(this, this.peers, signalingURL)
    this.credentials = new Credentials(this.signaling)

    this.unloadListener = () => this.close()
    if (typeof window !== 'undefined') {
      window.addEventListener('unload', this.unloadListener)
    }
  }

  create (): void {
    if (this._closing) {
      return
    }
    this.signaling.send({
      type: 'create'
    })
  }

  join (lobby: string): void {
    if (this._closing || this.signaling.receivedID === undefined) {
      return
    }
    this.signaling.send({
      type: 'join',
      lobby
    })
  }

  close (reason?: string): void {
    if (this._closing || this.signaling.receivedID === undefined) {
      return
    }
    this._closing = true
    this.emit('close', reason)

    if (this.id !== '') {
      this.signaling.send({
        type: 'leave',
        id: this.id,
        reason: reason ?? 'normal closure'
      })
    }

    for (const peer of this.peers.values()) {
      peer.close(reason)
    }
    this.signaling.close()

    if (typeof window !== 'undefined') {
      window.removeEventListener('unload', this.unloadListener)
    }
  }

  send (channel: string, peerID: string, data: string | Blob | ArrayBuffer | ArrayBufferView): void {
    if (!(channel in this.dataChannels)) {
      throw new Error('unknown channel ' + channel)
    }
    if (this.peers.has(peerID)) {
      this.peers.get(peerID)?.send(channel, data)
    }
  }

  broadcast (channel: string, data: string | Blob | ArrayBuffer | ArrayBufferView): void {
    if (!(channel in this.dataChannels)) {
      throw new Error('unknown channel ' + channel)
    }
    for (const peer of this.peers.values()) {
      peer.send(channel, data)
    }
  }

  async _addPeer (id: string, polite: boolean): Promise<void> {
    const config = await this.credentials.fillCredentials(this.rtcConfig)

    config.iceServers = config.iceServers?.filter(server => !(server.urls.includes('turn:') && server.username === undefined))

    const peer = new Peer(this, this.signaling, id, config, polite)
    this.peers.set(id, peer)
  }

  _removePeer (peer: Peer): boolean {
    return this.peers.delete(peer.id)
  }

  _prefetchTURNCredentials (): void {
    this.credentials.fillCredentials(this.rtcConfig).catch(() => {})
  }

  get id (): string {
    return this.signaling.receivedID ?? ''
  }

  get closing (): boolean {
    return this._closing
  }

  get size (): number {
    return this.peers.size
  }
}
