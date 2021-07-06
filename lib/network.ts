import { EventEmitter } from 'eventemitter3'

import { DefaultDataChannels, DefaultRTCConfiguration, DefaultSignalingURL } from '.'
import Signaling from './signaling'
import Peer from './peer'

interface NetworkListeners {
  ready: () => void | Promise<void>
  lobby: (code: string) => void | Promise<void>
  peerconnecting: (peer: Peer) => void | Promise<void>
  peerconnected: (peer: Peer) => void | Promise<void>
  peerdisconnected: (peer: Peer) => void | Promise<void>
  message: (peer: Peer, channel: string, data: any) => void | Promise<void>
  close: (reason?: string) => void | Promise<void>
  rtcerror: (e: RTCErrorEvent) => void | Promise<void>
  signalingerror: (e: Event) => void | Promise<void>
}

export default class Network extends EventEmitter<NetworkListeners> {
  private closing: boolean = false
  private readonly peers: Map<string, Peer>
  private readonly signaling: Signaling
  public dataChannels: {[label: string]: RTCDataChannelInit} = DefaultDataChannels

  constructor (public readonly gameID: string, private readonly signalingURL: string = DefaultSignalingURL) {
    super()
    this.peers = new Map<string, Peer>()
    this.signaling = new Signaling(this, signalingURL)
  }

  join (lobby?: string): void {
    this.signaling.send({
      type: 'join',
      game: this.gameID,
      lobby
    })
  }

  close (reason?: string): void {
    if (this.closing) {
      return
    }
    this.closing = true
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

  _addPeer (id: string, ref: string, polite: boolean): Peer {
    const peer = new Peer(this, this.signaling, id, ref, polite, DefaultRTCConfiguration)
    this.peers.set(id, peer)
    return peer
  }

  get id (): string {
    return this.signaling.receivedID ?? ''
  }

  get size (): number {
    return this.peers.size
  }
}
