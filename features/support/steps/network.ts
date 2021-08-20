import { After, Given, Then, When } from '@cucumber/cucumber'
import { Network } from '../../../lib'
import { Player } from '../types'
import { World } from '../world'

After(function (this: World) {
  this.players.forEach(p => {
    p.network.close()
  })
})

Given('{string} is connected and ready for game {string}', async function (this: World, playerName: string, gameID: string) {
  const network = new Network(gameID, this.signalingURL)
  const player = new Player(playerName, network)
  this.players.set(playerName, player)
  const event = await player.waitForEvent('ready')
  if (event == null) {
    throw new Error(`unable to add player ${playerName} to network`)
  }
})

When('{string} creates a network for game {string}', function (this: World, playerName: string, gameID: string) {
  const network = new Network(gameID, this.signalingURL)
  const player = new Player(playerName, network)
  this.players.set(playerName, player)
})

When('{string} creates a lobby', function (this: World, playerName: string) {
  const player = this.players.get(playerName)
  if (player == null) {
    return 'no such player'
  }
  player.network.create()
})

Then('{string} receives the network event {string}', async function (this: World, playerName: string, eventName: string) {
  const player = this.players.get(playerName)
  if (player == null) {
    throw new Error('no such player')
  }
  const event = await player.waitForEvent(eventName)
  if (event == null) {
    throw new Error(`no event ${eventName} received`)
  }
})

Then('{string} receives the network event {string} with the argument {string}', async function (this: World, playerName: string, eventName: string, expectedArgument: string) {
  const player = this.players.get(playerName)
  if (player == null) {
    throw new Error('no such player')
  }
  const event = await player.waitForEvent(eventName)
  if (event == null) {
    throw new Error(`no event ${eventName} received`)
  }
  if (event.eventPayload[0] !== expectedArgument) {
    throw new Error(`event ${eventName} received with wrong argument ${event.eventPayload[0] as string}`)
  }
})

Then('{string} has recieved the peer ID {string}', async function (this: World, playerName: string, exepctedID: string) {
  const player = this.players.get(playerName)
  if (player == null) {
    throw new Error('no such player')
  }
  if (player.network.id !== exepctedID) {
    throw new Error(`expected peer ID ${exepctedID} but got ${player.network.id}`)
  }
})
