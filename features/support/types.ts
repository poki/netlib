import { Network } from '../../lib'

interface RecordedEvent {
  eventName: string
  eventPayload: IArguments
}

export class Player {
  public events: RecordedEvent[] = []

  constructor (public name: string, public network: Network) {
    const allEvents = ['ready', 'lobby']
    allEvents.forEach(eventName => {
      const events = this.events
      this.network.on(eventName as any, function () {
        events.push({
          eventName: eventName,
          eventPayload: arguments
        })
      })
    })
  }

  async waitForEvent (eventName: string): Promise<RecordedEvent> {
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
