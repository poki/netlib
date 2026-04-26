import Network from './network'
import Signaling, { SignalingError } from './signaling'
import Latency from './latency'
import { PeerConfiguration, SignalingPacketTypes } from './types'

const LatencyRestartIceThreshold = 1000 // ms
const ReconnectionWindow = 8000 // ms

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
  private abortReconnectionAt: number = 0
  private allowNextManualRestartIceAt: number = 0

  public latency: Latency = new Latency(this)
  private lastMessageReceivedAt: number = 0

  private politenessTimeout?: ReturnType<typeof setTimeout>
  private reportLatencyEventTimeout?: ReturnType<typeof setTimeout>
  private readonly checkStateInterval: ReturnType<typeof setInterval>
  private readonly channels: { [name: string]: RTCDataChannel }

  private readonly testSessionWrapper?: (desc: RTCSessionDescription, config: PeerConfiguration, selfID: string, otherID: string) => Promise<void>

  /**
   * @internal
   */
  constructor (private readonly network: Network, private readonly signaling: Signaling, public readonly id: string, public readonly config: PeerConfiguration, private readonly polite: boolean) {
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
      this.politenessTimeout = setTimeout(() => {
        (async () => {
          try {
            if (this.closing) {
              return
            }
            this.makingOffer = true
            await this.conn.setLocalDescription()
            await this.waitForTestProxyCandidates()
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
            const error = new SignalingError('unknown-error', e as string)
            this.network._onSignalingError(error)
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
      chan.binaryType = 'arraybuffer'
      chan.addEventListener('error', e => this.onError(e))
      chan.addEventListener('closing', () => this.checkState())
      chan.addEventListener('close', () => this.checkState())
      chan.addEventListener('open', () => {
        if (!this.opened && !Object.values(this.channels).some(c => c.readyState !== 'open')) {
          if ('control' in this.channels) {
            this.latency = new Latency(this, this.channels.control)
          }

          if (this.politenessTimeout !== undefined) {
            clearTimeout(this.politenessTimeout)
          }

          this.signaling.send({
            type: 'connected',
            id: this.id
          })
          this.opened = true
          this.network.emit('connected', this)
          void this.signaling.event('rtc', 'connected', { target: this.id })
          this.reportLatencyEventTimeout = setTimeout(() => {
            void this.signaling.event('rtc', 'avg-latency-at-10s', { target: this.id, latency: `${this.latency.average}` })
          }, 10000)
        }
      })
      chan.addEventListener('message', e => {
        this.lastMessageReceivedAt = performance.now()
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
    if (this.reportLatencyEventTimeout != null) {
      clearTimeout(this.reportLatencyEventTimeout)
    }

    if (this.opened) {
      this.network.emit('disconnected', this)
      void this.signaling.event('rtc', 'disconnected', {
        target: this.id,
        reason: reason ?? '',
        reconnecting: this.reconnecting ? 'true' : 'false'
      })
    }
  }

  private checkState (): void {
    const now = performance.now()
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
      this.abortReconnectionAt = now + ReconnectionWindow
      this.network.emit('reconnecting', this)
      void this.signaling.event('rtc', 'attempt-reconnect', { target: this.id })
    } else if (this.reconnecting && connectionState === 'connected') {
      this.reconnecting = false
      this.network.emit('reconnected', this)
      void this.signaling.event('rtc', 'attempt-reconnected', { target: this.id })
    } else if (this.reconnecting && now > this.abortReconnectionAt) {
      this.close('reconnection timed out')
    }
    if (!this.reconnecting && 'control' in this.channels) {
      const lastPing = this.lastMessageReceivedAt
      if (lastPing !== 0) {
        const delta = now - lastPing
        if (delta > LatencyRestartIceThreshold && now > this.allowNextManualRestartIceAt) {
          this.allowNextManualRestartIceAt = now + 10000
          this.conn.restartIce()
        }
      }
    }
  }

  private onError (e: Event): void {
    this.network.emit('rtcerror', e)
    if (this.network.listenerCount('rtcerror') === 0) {
      console.error('rtcerror not handled:', e)
    }
    this.checkState()
    void this.signaling.event('rtc', 'error', { target: this.id, error: JSON.stringify(e) })
  }

  private async waitForTestProxyCandidates (): Promise<void> {
    if (this.testSessionWrapper === undefined || this.conn.iceGatheringState === 'complete') {
      return
    }

    await new Promise<void>(resolve => {
      const done = (): void => {
        clearTimeout(timeout)
        this.conn.removeEventListener('icegatheringstatechange', onStateChange)
        resolve()
      }
      const onStateChange = (): void => {
        if (this.conn.iceGatheringState === 'complete') {
          done()
        }
      }
      const timeout = setTimeout(done, 5000)
      this.conn.addEventListener('icegatheringstatechange', onStateChange)
      onStateChange()
    })
  }

  /**
   * @internal
   */
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
            await this.conn.setLocalDescription()
            await this.waitForTestProxyCandidates()
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

  get maxMessageSize (): number | null {
    return this.conn.sctp?.maxMessageSize ?? null
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
    return !l.startsWith('a=candidate') || parseProxyableCandidate(l) !== undefined
  })

  for (let i = 0; i < lines.length; i++) {
    const candidate = parseProxyableCandidate(lines[i])
    if (candidate !== undefined) {
      const params = new URLSearchParams({
        id: selfID + otherID,
        host: candidate.host,
        port: candidate.port
      })
      const resp = await fetch(`${config.testproxyURL}/create?${params.toString()}`)
      const substitudePort = await resp.text()
      candidate.parts[4] = '127.0.0.1'
      candidate.parts[5] = substitudePort
      lines[i] = candidate.parts.join(' ')
    }
  }

  ;(desc as any).sdp = lines.join('\r\n')
}

function parseProxyableCandidate (line: string): { parts: string[], host: string, port: string } | undefined {
  if (!line.startsWith('a=candidate')) {
    return undefined
  }
  const parts = line.split(' ')
  const protocol = parts[2]?.toLowerCase()
  const host = parts[4]
  const port = parts[5]
  const typ = parts[7]
  if (protocol !== 'udp' || typ !== 'host' || host === undefined || port === undefined || host.includes(':')) {
    return undefined
  }
  return { parts, host, port }
}
