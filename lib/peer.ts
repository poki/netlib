import Network from './network'
import Signaling from './signaling'
import Latency from './latency'
import { PeerConfiguration, SignalingPacketTypes } from './types'

export default class Peer {
  public readonly conn: RTCPeerConnection

  // Signaling state:
  private makingOffer: boolean = false
  private ignoreOffer: boolean = false
  private isSettingRemoteAnswerPending: boolean = false

  // Connection state:
  private opened: boolean = false
  private closing: boolean = false
  private reconnecting: boolean = false

  public latency: Latency = new Latency(this)

  private readonly checkStateInterval: ReturnType<typeof setInterval>
  private readonly channels: {[name: string]: RTCDataChannel}

  private readonly testSessionWrapper?: (desc: RTCSessionDescription, config: PeerConfiguration, selfID: string, otherID: string) => Promise<void>

  constructor (private readonly network: Network, private readonly signaling: Signaling, public readonly id: string, private readonly polite: boolean, private readonly config: PeerConfiguration) {
    this.channels = {}
    this.network.log('creating peer')

    this.testSessionWrapper = undefined

    this.conn = new RTCPeerConnection(config)
    if (config.testproxyURL === undefined) { // we dont push candidates in a test setup
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
    } else {
      this.testSessionWrapper = wrapSessionDescription
    }
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
              await this.testSessionWrapper?.(description, this.config, this.network.id, this.id)
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
    this.conn.addEventListener('iceconnectionstatechange', () => this.checkState())

    this.network.emit('connecting', this)

    let i = 0
    for (const label in this.network.dataChannels) {
      const chan = this.conn.createDataChannel(label, {
        ...this.network.dataChannels[label],
        id: i++,
        negotiated: true
      })
      chan.addEventListener('error', e => this.onError(e))
      chan.addEventListener('closing', () => this.checkState())
      chan.addEventListener('close', () => this.checkState())
      chan.addEventListener('open', () => {
        if (!this.opened && !Object.values(this.channels).some(c => c.readyState !== 'open')) {
          if ('control' in this.channels) {
            this.latency = new Latency(this, this.channels.control)
          }

          this.signaling.send({
            type: 'connected',
            id: this.id
          })
          this.opened = true
          this.network.emit('connected', this)
        }
      })
      chan.addEventListener('message', e => {
        if (label !== 'control') {
          this.network.emit('message', this, label, e.data)
        }
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
      this.network.emit('disconnected', this)
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
    // console.log('state', this.id, this.conn.connectionState, this.conn.iceConnectionState, Object.values(this.channels).map(c => c.readyState))
    if (!this.reconnecting && (connectionState === 'disconnected' || connectionState === 'failed')) {
      this.reconnecting = true
      this.network.emit('reconnecting', this)
    } else if (this.reconnecting && connectionState === 'connected') {
      this.reconnecting = false
      this.network.emit('reconnected', this)
    }
    if (connectionState === 'failed' || connectionState === 'closed') {
      this.conn.restartIce()
    }
    // TODO: Actually close at some point. 😅
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
              await this.testSessionWrapper?.(description, this.config, this.network.id, this.id)
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

async function wrapSessionDescription (desc: RTCSessionDescription, config: PeerConfiguration, selfID: string, otherID: string): Promise<void> {
  if (config.testproxyURL === undefined) {
    return
  }

  let lines = desc.sdp.split('\r\n')
  lines = lines.filter(l => {
    return !l.startsWith('a=candidate') || (l.includes('127.0.0.1') && l.includes('udp'))
  })

  for (let i = 0; i < lines.length; i++) {
    const l = lines[i]
    if (l.startsWith('a=candidate') && l.includes('127.0.0.1')) {
      const orignalPort = l.split('127.0.0.1 ').pop()?.split(' ')[0] // find port
      if (orignalPort != null) {
        const resp = await fetch(`${config.testproxyURL}/create?id=${selfID + otherID}&port=${orignalPort}`)
        const substitudePort = await resp.text()
        lines[i] = l.replaceAll(` ${orignalPort} `, ` ${substitudePort} `)
      }
    }
  }

  ;(desc as any).sdp = lines.join('\r\n')
}
