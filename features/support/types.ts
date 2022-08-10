import { Network } from '../../lib'

interface RecordedEvent {
  eventName: string
  eventPayload: IArguments
}

const allEvents = ['close', 'ready', 'lobby', 'connected', 'disconnected', 'reconnecting', 'reconnected', 'message', 'signalingerror', 'signalingreconnected']

export class Player {
  public events: RecordedEvent[] = []

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

  hasSeenEvent (eventName: string, ...matchArguments: any[]): boolean {
    return this.events.some(e => matchEvent(e, eventName, ...matchArguments))
  }

  async waitForEvent (eventName: string, ...matchArguments: any[]): Promise<RecordedEvent> {
    if (!allEvents.includes(eventName)) {
      throw new Error(`Event type ${eventName} not tracked, add to allEvents in types.ts`)
    }

    return await new Promise(resolve => {
      const events = this.events.filter(e => matchEvent(e, eventName, ...matchArguments))
      if (events.length > 0) {
        resolve(events[0])
      } else {
        this.network.on(eventName as any, function () {
          const e = {
            eventName: eventName,
            eventPayload: arguments
          }
          if (matchEvent(e, eventName, ...matchArguments)) {
            resolve(e)
          }
        })
      }
    })
  }
}

function matchEvent (e: RecordedEvent, eventName: string, ...matchArguments: any[]): boolean {
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
