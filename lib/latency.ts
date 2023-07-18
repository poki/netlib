import Peer from './peer'

const PingInterval = 500
const WindowSampleSize = 50

const PING = 'ping'
const PONG = 'pong'

export default class Latency {
  private readonly window: number[] = []
  private lastPingSentAt: number = 0

  public last: number = 0
  public average: number = 0
  public jitter: number = 0
  public max: number = 0
  public min: number = 0

  /**
   * @internal
   */
  constructor (private readonly peer: Peer, private readonly control?: RTCDataChannel) {
    if (control !== undefined) {
      this.ping()

      control.addEventListener('message', e => this.onMessage(e.data))
    }
  }

  private ping (): void {
    this.lastPingSentAt = performance.now()
    if (this.control?.readyState === 'open') {
      this.control?.send(PING)
    }
  }

  private onMessage (data: string): void {
    if (data === PING) {
      if (this.control?.readyState === 'open') {
        this.control?.send(PONG)
      }
      return
    }
    if (data !== PONG) {
      return
    }

    const now = performance.now()
    const delta = now - this.lastPingSentAt

    this.window.unshift(delta)
    if (this.window.length > WindowSampleSize) {
      this.window.pop()
    }

    this.last = delta
    this.max = Math.max(...this.window)
    this.min = Math.min(...this.window)

    this.average = this.window.reduce((a, b) => a + b, 0) / this.window.length

    if (this.window.length > 1) {
      this.jitter = this.window.slice(1).map((x, i) => Math.abs(x - this.window[i])).reduce((a, b) => a + b, 0) / (this.window.length - 1)
    }

    setTimeout(() => this.ping(), PingInterval - delta)
  }
}
