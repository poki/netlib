import { After, Given, Then, When } from '@cucumber/cucumber'
import { Network } from '../../../lib'
import { World } from '../world'

After(async function (this: World) {
  this.players.forEach(p => {
    p.network.close('closing test suite')
  })
  this.players.clear()
})

Given('{string} is connected and ready for game {string}', async function (this: World, playerName: string, gameID: string) {
  const player = this.createPlayer(playerName, gameID)
  const event = await player.waitForEvent('ready')
  if (event == null) {
    throw new Error(`unable to add player ${playerName} to network`)
  }
})

Given('{string} are joined in a lobby', async function (this: World, playerNamesRaw: string) {
  const playerNames = playerNamesRaw.split(',').map(s => s.trim())
  if (playerNames.length < 2) {
    throw new Error('need at least 2 players to join a lobby')
  }
  const first = this.players.get(playerNames[0])
  if (first === undefined) {
    throw new Error(`player ${playerNames[0]} not found`)
  }

  first.network.create()
  const lobbyEvent = await first.waitForEvent('lobby')
  const lobbyCode = lobbyEvent.eventPayload[0] as string

  for (let i = 1; i < playerNames.length; i++) {
    const playerName = playerNames[i]
    const player = this.players.get(playerName)
    if (player == null) {
      return new Error(`player ${playerName} not found`)
    }
    player.network.join(lobbyCode)
    await player.waitForEvent('lobby')
  }

  for (let i = 0; i < playerNames.length; i++) {
    const playerName = playerNames[i]
    const player = this.players.get(playerName)
    if (player?.network.peers.size !== playerNames.length - 1) {
      return new Error('player not connected with enough others')
    }
    for (const [, peer] of player?.network.peers) {
      await player?.waitForEvent('connected', peer)
    }
  }
})

When('{string} creates a network for game {string}', function (this: World, playerName: string, gameID: string) {
  this.createPlayer(playerName, gameID)
})

When('{string} creates a lobby', function (this: World, playerName: string) {
  const player = this.players.get(playerName)
  if (player == null) {
    return 'no such player'
  }
  player.network.create()
})

When('{string} connects to the lobby {string}', function (this: World, playerName: string, lobbyCode: string) {
  const player = this.players.get(playerName)
  if (player == null) {
    return 'no such player'
  }
  player.network.join(lobbyCode)
})

When('{string} boardcasts {string} over the reliable channel', function (this: World, playerName: string, message: string) {
  const player = this.players.get(playerName)
  if (player == null) {
    return 'no such player'
  }
  player.network.broadcast(Network.CHANNEL_RELIABLE, message)
})

When('{string} disconnects', async function (this: World, playerName: string) {
  const player = this.players.get(playerName)
  if (player == null) {
    return 'no such player'
  }
  player.network.close()
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
  const event = await player.waitForEvent(eventName, expectedArgument)
  if (event == null) {
    throw new Error(`no event ${eventName}(${expectedArgument}) received`)
  }
})

Then('{string} receives the network event {string} with the arguments {string}, {string} and {string}', async function (this: World, playerName: string, eventName: string, expectedArgument0: string, expectedArgument1: string, expectedArgument2: string) {
  const player = this.players.get(playerName)
  if (player == null) {
    throw new Error('no such player')
  }
  const event = await player.waitForEvent(eventName, expectedArgument0, expectedArgument1, expectedArgument2)
  if (event == null) {
    throw new Error(`no event ${eventName}(${expectedArgument0}, ${expectedArgument1}, ${expectedArgument2}) received`)
  }
})

Then('{string} has recieved the peer ID {string}', async function (this: World, playerName: string, exepctedID: string) {
  const player = this.players.get(playerName)
  if (player == null) {
    throw new Error('no such player')
  }
  if (player.network.id === '') {
    await player.waitForEvent('lobby')
  }
  if (player.network.id !== exepctedID) {
    throw new Error(`expected peer ID ${exepctedID} but got ${player.network.id}`)
  }
})
