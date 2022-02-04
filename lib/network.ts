import { EventEmitter } from 'eventemitter3'

import { DefaultDataChannels, DefaultRTCConfiguration, DefaultSignalingURL } from '.'
import { PeerConfiguration } from './types'
import Signaling from './signaling'
import Peer from './peer'

interface NetworkListeners {
  ready: () => void | Promise<void>
  lobby: (code: string) => void | Promise<void>
  connecting: (peer: Peer) => void | Promise<void>
  connected: (peer: Peer) => void | Promise<void>
  reconnecting: (peer: Peer) => void | Promise<void>
  reconnected: (peer: Peer) => void | Promise<void>
  disconnected: (peer: Peer) => void | Promise<void>
  message: (peer: Peer, channel: string, data: string | Blob | ArrayBuffer | ArrayBufferView) => void | Promise<void>
  close: (reason?: string) => void | Promise<void>
  rtcerror: (e: RTCErrorEvent) => void | Promise<void>
  signalingerror: (e: any) => void | Promise<void>
}

export default class Network extends EventEmitter<NetworkListeners> {
  private _closing: boolean = false
  public readonly peers: Map<string, Peer>
  private readonly signaling: Signaling
  public dataChannels: {[label: string]: RTCDataChannelInit} = DefaultDataChannels

  public log: (...data: any[]) => void = (...args: any[]) => {} // console.log

  constructor (public readonly gameID: string, private readonly signalingURL: string = DefaultSignalingURL, private readonly rtcConfig: PeerConfiguration = DefaultRTCConfiguration) {
    super()
    this.peers = new Map<string, Peer>()
    this.signaling = new Signaling(this, this.peers, signalingURL)
  }

  create (): void {
    this.signaling.send({
      type: 'create',
      game: this.gameID
    })
  }

  join (lobby: string): void {
    this.signaling.send({
      type: 'join',
      game: this.gameID,
      lobby
    })
  }

  close (reason?: string): void {
    if (this._closing) {
      return
    }
    this._closing = true
    this.emit('close', reason)
    for (const peer of this.peers.values()) {
      peer.close(reason)
    }
    this.signaling.close(reason)
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

  _addPeer (id: string, polite: boolean): Peer {
    const peer = new Peer(this, this.signaling, id, polite, this.rtcConfig)
    this.peers.set(id, peer)
    return peer
  }

  _removePeer (peer: Peer): boolean {
    return this.peers.delete(peer.id)
  }

  public static CHANNEL_RELIABLE = 'reliable'
  public static CHANNEL_UNRELIABLE = 'unreliable'

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
