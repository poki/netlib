import { Network } from '../../lib'
import { LobbyListEntry } from '../../lib/types'

interface RecordedEvent {
  eventName: string
  eventPayload: IArguments
}

const allEvents = ['close', 'ready', 'lobby', 'connected', 'disconnected', 'reconnecting', 'reconnected', 'message', 'signalingerror', 'signalingreconnected']

export class Player {
  public lastReceivedLobbies: LobbyListEntry[] = []
  public events: RecordedEvent[] = []
  public scanIndex = 0

  constructor (public name: string, public network: Network) {
    allEvents.forEach(eventName => {
      const events = this.events
      this.network.on(eventName as any, function () {
        // console.log(name, `(${network.id})`, 'received event', eventName, `${arguments[0] as string}`, arguments[1], arguments[2])
        events.push({
          eventName: eventName,
          eventPayload: arguments
        })
      })
    })

    network.on('signalingerror', _ => {})
    network.on('rtcerror', _ => {})
  }

  findEvent (eventName: string, matchArguments: any[] = []): RecordedEvent | undefined {
    return this.events.find(e => matchEvent(e, eventName, matchArguments))
  }

  async waitForEvent (eventName: string, matchArguments: any[] = [], consume: boolean = true): Promise<RecordedEvent> {
    if (!allEvents.includes(eventName)) {
      throw new Error(`Event type ${eventName} not tracked, add to allEvents in types.ts`)
    }

    const find = (): RecordedEvent | null => {
      const ix = this.events.slice(this.scanIndex).findIndex(e => matchEvent(e, eventName, matchArguments))
      if (ix >= 0) {
        const event = this.events[this.scanIndex + ix]
        if (consume) {
          this.scanIndex += ix + 1
        }
        return event
      }
      return null
    }

    return await new Promise((resolve, reject) => {
      const event = find()
      if (event !== null) {
        resolve(event)
      } else {
        let interval: NodeJS.Timeout | null = null
        const timeout = setTimeout(() => {
          const sameEvents = this.events.slice(this.scanIndex).filter(e => e.eventName === eventName)
          const others = sameEvents.map(e => Array.from(e.eventPayload).map(a => `${a as string}`).join(',')).join(' + ')
          if (interval !== null) {
            clearInterval(interval)
          }
          reject(new Error(`Event not found, timed out, got: ${others}`))
        }, 20000)
        interval = setInterval(() => {
          const event = find()
          if (event !== null) {
            if (interval !== null) {
              clearInterval(interval)
            }
            clearTimeout(timeout)
            resolve(event)
          }
        }, 100)
      }
    })
  }
}

function matchEvent (e: RecordedEvent, eventName: string, matchArguments: any[] = []): boolean {
  if (e.eventName !== eventName) {
    return false
  }
  let argumentsMatch = true
  matchArguments.forEach((arg, i) => {
    if (typeof arg === 'string' || arg instanceof String) {
      // Fool typescript into calling toString() on the event argument:
      argumentsMatch = `${e.eventPayload[i] as string}` === arg
    } else {
      argumentsMatch = e.eventPayload[i] === arg
    }
    return argumentsMatch
  })
  return argumentsMatch
}
