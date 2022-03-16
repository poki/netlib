import Signaling from './signaling'
import { CredentialsPacket, PeerConfiguration } from './types'

const FetchTimeout = 5000
export const PokiTurnMatch = 'turn:turn.rtc.poki.com'

export default class Credentials {
  private cachedCredentials?: CredentialsPacket
  private cachedCredentialsExpireAt: number = 0

  private runningPromise?: Promise<CredentialsPacket>

  constructor (public signaling: Signaling) {
  }

  async fillCredentials (config: PeerConfiguration): Promise<RTCConfiguration> {
    const cloned = JSON.parse(JSON.stringify(config)) as PeerConfiguration

    if (config.testproxyURL !== undefined) {
      return cloned
    }

    const hasPokiTurn = cloned.iceServers?.some(s => s.urls === PokiTurnMatch || s.urls.includes(PokiTurnMatch)) ?? false
    if (!hasPokiTurn || cloned.iceServers === undefined) {
      console.log('no match')
      return cloned
    }
    if (this.runningPromise === undefined) {
      console.log('starting promise')
      this.runningPromise = new Promise<CredentialsPacket>((resolve) => {
        if (this.cachedCredentials != null && this.cachedCredentialsExpireAt > performance.now()) {
          resolve(this.cachedCredentials)
          return
        }
        this.signaling.once('credentials', credentials => {
          if (credentials.type === 'credentials') {
            this.cachedCredentials = credentials
            this.cachedCredentialsExpireAt = performance.now() + (((credentials.lifetime ?? 0) - 60) * 1000)
            resolve(credentials)
          }
        })
        this.signaling.send({
          type: 'credentials'
        })
        setTimeout(() => {
          resolve({ type: 'credentials' })
        }, FetchTimeout)
      })
    }
    const credentials = await this.runningPromise
    this.runningPromise = undefined
    console.log('got credentials', credentials)

    if (credentials.url === undefined) {
      return cloned
    }

    cloned.iceServers.forEach(s => {
      if (s.urls === PokiTurnMatch || s.urls.includes(PokiTurnMatch)) {
        s.urls = credentials.url ?? ''
        s.username = credentials.username
        s.credential = credentials.credential
      }
    })
    return cloned
  }
}
