import Network from './network'
import Signaling from './signaling'
import { SignalingPacketTypes } from './types'

export default class Peer {
  public readonly conn: RTCPeerConnection

  // Signaling state:
  private makingOffer: boolean = false
  private ignoreOffer: boolean = false
  private isSettingRemoteAnswerPending: boolean = false

  // Connection state:
  private opened: boolean = false
  private closing: boolean = false

  // Connection stats:
  public latency: number = 0

  private readonly checkStateInterval: ReturnType<typeof setInterval>
  private readonly channels: {[name: string]: RTCDataChannel}

  constructor (private readonly network: Network, private readonly signaling: Signaling, public readonly id: string, private readonly polite: boolean, config: RTCConfiguration) {
    this.channels = {}
    this.network.log('creating peer')

    this.conn = new RTCPeerConnection(config)
    this.conn.addEventListener('icecandidate', e => {
      const candidate = e.candidate
      if (candidate !== null) {
        signaling.send({
          type: 'candidate',
          source: this.network.id,
          recipient: this.id,
          candidate
        })
      }
    })
    this.conn.addEventListener('negotiationneeded', () => {
      setTimeout(() => {
        (async () => {
          try {
            if (this.closing) {
              return
            }
            this.makingOffer = true
            if (process.env.NODE_ENV === 'test') {
              // Running tests with node and the wrtc package causes the
              // setLocalDescription to fail with undefined as argment.
              await this.conn.setLocalDescription(await this.conn.createOffer())
            } else {
              await this.conn.setLocalDescription()
            }
            const description = this.conn.localDescription
            if (description != null) {
              this.signaling.send({
                type: 'description',
                source: this.network.id,
                recipient: this.id,
                description
              })
            }
          } catch (e) {
            this.network.emit('signalingerror', e)
            if (this.network.listenerCount('signalingerror') === 0) {
              console.error('signallingerror not handled:', e)
            }
          } finally {
            this.makingOffer = false
          }
        })().catch(_ => {})
      }, this.polite ? 100 : 0)
    })

    this.checkStateInterval = setInterval(() => {
      this.checkState()
    }, 500)
    this.conn.addEventListener('signalingstatechange', () => this.checkState())
    this.conn.addEventListener('connectionstatechange', () => this.checkState())
    this.conn.addEventListener('iceconnectionstatechange', () => {
      if (this.conn.iceConnectionState === 'failed') {
        this.conn.restartIce()
      }
      this.checkState()
    })

    this.network.emit('peerconnecting', this)

    let i = 0
    for (const label in this.network.dataChannels) {
      const chan = this.conn.createDataChannel(label, {
        ...this.network.dataChannels[label],
        id: i++,
        negotiated: true
      })
      chan.addEventListener('error', e => this.onError(e))
      chan.addEventListener('close', () => this.checkState())
      chan.addEventListener('open', () => {
        if (!this.opened && !Object.values(this.channels).some(c => c.readyState !== 'open')) {
          this.signaling.send({
            type: 'connected',
            id: this.id
          })
          this.opened = true
          this.network.emit('peerconnected', this)
        }
      })
      chan.addEventListener('message', e => {
        this.network.emit('message', this, label, e.data)
      })
      this.channels[label] = chan
    }
  }

  public close (reason?: string): void {
    if (this.closing) {
      return
    }
    this.closing = true

    // Inform signaling server that the peer has been disconnected:
    this.signaling.send({
      type: 'disconnected',
      id: this.id,
      reason: reason ?? 'normal closure'
    })

    Object.values(this.channels).forEach(c => c.close())
    this.conn.close()
    this.network._removePeer(this)
    if (this.checkStateInterval != null) {
      clearInterval(this.checkStateInterval)
    }

    if (this.opened) {
      this.network.emit('peerdisconnected', this)
    }
  }

  private checkState (): void {
    const connectionState = this.conn.connectionState ?? this.conn.iceConnectionState
    if (this.closing) {
      return
    }
    if (!this.opened) {
      if (connectionState === 'failed') {
        this.close('connecting failed')
      }
      return
    }
    if (Object.values(this.channels).some(c => c.readyState !== 'open')) {
      this.close('data channel closed')
    }
    if (connectionState !== 'connected') {
      this.close(`invalid connection state ${connectionState}/${this.conn.signalingState}`)
    }

    this.conn.getStats().then(stats => {
      stats.forEach((report) => {
        if (report.type === 'transport') {
          this.latency = report.currentRoundTripTime ?? 0
        }
      })
    }).catch(_ => {})
  }

  private onError (e: RTCErrorEvent): void {
    this.network.emit('rtcerror', e)
    if (this.network.listenerCount('rtcerror') === 0) {
      console.error('rtcerror not handled:', e)
    }
    this.checkState()
  }

  async _onSignalingMessage (packet: SignalingPacketTypes): Promise<void> {
    switch (packet.type) {
      case 'candidate':
        if (packet.candidate != null) {
          try {
            await this.conn.addIceCandidate(packet.candidate)
          } catch (e) {
            if (!this.ignoreOffer) {
              throw e
            }
          }
        }
        break

      case 'description':
        {
          const { description } = packet
          const readyForOffer =
            !this.makingOffer &&
            (this.conn.signalingState === 'stable' || this.isSettingRemoteAnswerPending)
          const offerCollision = description.type === 'offer' && !readyForOffer

          this.ignoreOffer = !this.polite && offerCollision
          if (this.ignoreOffer) {
            return
          }
          this.isSettingRemoteAnswerPending = description.type === 'answer'
          await this.conn.setRemoteDescription(description)
          this.isSettingRemoteAnswerPending = false
          if (description.type === 'offer') {
            if (process.env.NODE_ENV === 'test') {
              await this.conn.setLocalDescription(await this.conn.createAnswer())
            } else {
              await this.conn.setLocalDescription()
            }
            const description = this.conn.localDescription
            if (description != null) {
              this.signaling.send({
                type: 'description',
                source: this.network.id,
                recipient: this.id,
                description
              })
            }
          }
        }
        break
    }
  }

  send (channel: string, data: string | Blob | ArrayBuffer | ArrayBufferView): void {
    if (!(channel in this.channels)) {
      throw new Error('unknown channel ' + channel)
    }
    const chan = this.channels[channel]
    if (chan.readyState === 'open') {
      chan.send(data as any)
    }
  }

  toString (): string {
    return `[Peer: ${this.id}]`
  }
}
