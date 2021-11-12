import { Network } from '../../lib'

interface RecordedEvent {
  eventName: string
  eventPayload: IArguments
}

const allEvents = ['ready', 'lobby', 'peerconnected']

export class Player {
  public events: RecordedEvent[] = []

  constructor (public name: string, public network: Network) {
    allEvents.forEach(eventName => {
      const events = this.events
      this.network.on(eventName as any, function () {
        events.push({
          eventName: eventName,
          eventPayload: arguments
        })
      })
    })

    network.on('signalingerror', err => {
      console.error(err)
    })
    network.on('rtcerror', err => {
      console.error(err)
    })
  }

  async waitForEvent (eventName: string): Promise<RecordedEvent> {
    if (!allEvents.includes(eventName)) {
      throw new Error(`Event type ${eventName} not tracked, add to allEvents in types.ts`)
    }
    return await new Promise(resolve => {
      const events = this.events.filter(e => e.eventName === eventName)
      if (events[0]?.eventName === eventName) {
        resolve(events[0])
      } else {
        this.network.on(eventName as any, function () {
          resolve({
            eventName: eventName,
            eventPayload: arguments
          })
        })
      }
    })
  }
}
