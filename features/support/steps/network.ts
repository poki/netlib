import { After, DataTable, Given, Then, When } from '@cucumber/cucumber'
import { World } from '../world'

After(async function (this: World) {
  this.players.forEach(p => {
    p.network.close('closing test suite')
  })
  this.players.clear()
})

Given('{string} is connected as {string} and ready for game {string}', async function (this: World, playerName: string, peerID: string, gameID: string) {
  const player = await this.createPlayer(playerName, gameID)
  const event = await player.waitForEvent('ready')
  if (event == null) {
    throw new Error(`unable to add player ${playerName} to network`)
  }
  if (player.network.id !== peerID) {
    throw new Error(`expected peer ID ${peerID} but got ${player.network.id}`)
  }
})

async function areJoinedInALobby (this: World, playerNamesRaw: string): Promise<void> {
  const playerNames = playerNamesRaw.split(',').map(s => s.trim())
  if (playerNames.length < 2) {
    throw new Error('need at least 2 players to join a lobby')
  }
  const first = this.players.get(playerNames[0])
  if (first === undefined) {
    throw new Error(`player ${playerNames[0]} not found`)
  }

  void first.network.create()
  const lobbyEvent = await first.waitForEvent('lobby')
  const lobbyCode = lobbyEvent.eventPayload[0] as string

  for (let i = 1; i < playerNames.length; i++) {
    const playerName = playerNames[i]
    const player = this.players.get(playerName)
    if (player == null) {
      throw new Error(`player ${playerName} not found`)
    }
    void player.network.join(lobbyCode)
    await player.waitForEvent('lobby')
  }

  for (let i = 0; i < playerNames.length; i++) {
    const playerName = playerNames[i]
    const player = this.players.get(playerName)
    for (let j = 0; j < playerNames.length - 1; j++) {
      await player?.waitForEvent('connected')
    }
    if (player?.network.peers.size !== playerNames.length - 1) {
      throw new Error('player not connected with enough others')
    }
  }
}

Given('{string} are joined in a lobby', areJoinedInALobby)

Given('{string} are joined in a lobby for game {string}', async function (this: World, playerNamesRaw: string, gameID: string) {
  const playerNames = playerNamesRaw.split(',').map(s => s.trim())
  if (playerNames.length < 2) {
    throw new Error('need at least 2 players to join a lobby')
  }

  for (let i = 0; i < playerNames.length; i++) {
    const playerName = playerNames[i]
    const player = await this.createPlayer(playerName, gameID)
    const event = await player.waitForEvent('ready')
    if (event == null) {
      throw new Error(`unable to add player ${playerName} to network`)
    }
  }

  await areJoinedInALobby.call(this, playerNamesRaw)
})

Given('these lobbies exist:', async function (this: World, lobbies: DataTable) {
  if (this.testproxyURL === undefined) {
    throw new Error('testproxy not active')
  }

  const columns: string[] = []
  const values: string[] = []

  lobbies.hashes().forEach(row => {
    const v: string[] = []

    Object.keys(row).forEach(key => {
      const value = row[key]
      if (key === 'playerCount') {
        if (!columns.includes('peers')) {
          columns.push('peers')
        }

        const n = parseInt(value, 10)
        const peers: string[] = []

        for (let i = 0; i < n; i++) {
          peers.push(`'peer${i}'`)
        }

        v.push(`ARRAY[${peers.join(', ')}]::VARCHAR(20)[]`)
      } else {
        if (!columns.includes(key)) {
          columns.push(key)
        }

        if (value === 'null') {
          v.push('NULL')
        } else {
          v.push(`'${value}'`)
        }
      }
    })

    values.push(`(${v.join(', ')})`)
  })

  await fetch(`${this.testproxyURL}/sql`, {
    method: 'POST',
    body: 'INSERT INTO lobbies (' + columns.join(', ') + ') VALUES ' + values.join(', ')
  })
})

When('{string} creates a network for game {string}', async function (this: World, playerName: string, gameID: string) {
  await this.createPlayer(playerName, gameID)
})

When('{string} creates a lobby', async function (this: World, playerName: string) {
  const player = this.players.get(playerName)
  if (player == null) {
    throw new Error('no such player')
  }
  await player.network.create()
})

When('{string} creates a lobby with these settings:', async function (this: World, playerName: string, settingsBlob: string) {
  const player = this.players.get(playerName)
  if (player == null) {
    throw new Error('no such player')
  }
  const settings = JSON.parse(settingsBlob)
  await player.network.create(settings)
})

When('{string} connects to the lobby {string}', async function (this: World, playerName: string, lobbyCode: string) {
  const player = this.players.get(playerName)
  if (player == null) {
    throw new Error('no such player')
  }
  await player.network.join(lobbyCode)
})

When('{string} connects to the lobby {string} with the password {string}', async function (this: World, playerName: string, lobbyCode: string, password: string) {
  const player = this.players.get(playerName)
  if (player == null) {
    throw new Error('no such player')
  }
  await player.network.join(lobbyCode, password)
})

When('{string} tries to connect to the lobby {string} with the password {string}', async function (this: World, playerName: string, lobbyCode: string, password: string) {
  const player = this.players.get(playerName)
  if (player == null) {
    throw new Error('no such player')
  }
  try {
    this.lastError.delete(playerName)
    await player.network.join(lobbyCode, password)
  } catch (e) {
    this.lastError.set(playerName, e as Error)
  }
})

When('{string} tries to connect to the lobby {string} without a password', async function (this: World, playerName: string, lobbyCode: string) {
  const player = this.players.get(playerName)
  if (player == null) {
    throw new Error('no such player')
  }
  try {
    this.lastError.delete(playerName)
    await player.network.join(lobbyCode)
  } catch (e) {
    this.lastError.set(playerName, e as Error)
  }
})

When('{string} boardcasts {string} over the reliable channel', function (this: World, playerName: string, message: string) {
  const player = this.players.get(playerName)
  if (player == null) {
    throw new Error('no such player')
  }
  player.network.broadcast('reliable', message)
})

When('{string} disconnects', async function (this: World, playerName: string) {
  const player = this.players.get(playerName)
  if (player == null) {
    throw new Error('no such player')
  }
  player.network.close()
})

When('{string} leaves the lobby', async function (this: World, playerName: string) {
  const player = this.players.get(playerName)
  if (player == null) {
    throw new Error('no such player')
  }
  await player.network.leave()
})

When('{string} requests all lobbies', async function (this: World, playerName: string) {
  const player = this.players.get(playerName)
  if (player == null) {
    throw new Error('no such player')
  }
  const lobbies = await player.network.list()
  player.lastReceivedLobbies = lobbies
})

When('{string} requests lobbies with this filter:', async function (this: World, playerName: string, filter: string) {
  const player = this.players.get(playerName)
  if (player == null) {
    throw new Error('no such player')
  }
  const lobbies = await player.network.list(JSON.parse(filter))
  player.lastReceivedLobbies = lobbies
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
  const event = await player.waitForEvent(eventName, [expectedArgument])
  if (event == null) {
    throw new Error(`no event ${eventName}(${expectedArgument}) received`)
  }
})

Then('{string} receives the network event {string} with the arguments {string}, {string} and {string}', async function (this: World, playerName: string, eventName: string, expectedArgument0: string, expectedArgument1: string, expectedArgument2: string) {
  const player = this.players.get(playerName)
  if (player == null) {
    throw new Error('no such player')
  }
  const event = await player.waitForEvent(eventName, [expectedArgument0, expectedArgument1, expectedArgument2])
  if (event == null) {
    throw new Error(`no event ${eventName}(${expectedArgument0}, ${expectedArgument1}, ${expectedArgument2}) received`)
  }
})

Then('{string} receives the network event {string} with the arguments:', async function (this: World, playerName: string, eventName: string, args: string) {
  const player = this.players.get(playerName)
  if (player == null) {
    throw new Error('no such player')
  }
  const event = await player.waitForEvent(eventName, JSON.parse(args))
  if (event == null) {
    throw new Error(`no event ${eventName}(...) received`)
  }
})

Then('{string} has recieved the peer ID {string}', async function (this: World, playerName: string, exepctedID: string) {
  const player = this.players.get(playerName)
  if (player == null) {
    throw new Error('no such player')
  }
  if (player.network.id === '') {
    await player.waitForEvent('ready', [], false)
  }
  if (player.network.id !== exepctedID) {
    throw new Error(`expected peer ID ${exepctedID} but got ${player.network.id}`)
  }
})

Then('{string} should receive {int} lobbies', function (this: World, playerName: string, expectedLobbyCount: number) {
  const player = this.players.get(playerName)
  if (player == null) {
    throw new Error('no such player')
  }
  if (player.lastReceivedLobbies?.length !== expectedLobbyCount) {
    throw new Error(`expected ${expectedLobbyCount} lobbies but got ${player.lastReceivedLobbies?.length}`)
  }
})

Then('{string} should have received only these lobbies:', function (this: World, playerName: string, expectedLobbies: DataTable) {
  const player = this.players.get(playerName)
  if (player == null) {
    throw new Error('no such player')
  }
  expectedLobbies.hashes().forEach(row => {
    const correctCodeLobby = player.lastReceivedLobbies.filter(lobby => lobby.code === row.code)
    if (correctCodeLobby.length !== 1) {
      throw new Error(`expected to find one lobby with code ${row.code} but found ${correctCodeLobby.length}`)
    }
    const lobby = correctCodeLobby[0] as any
    Object.keys(lobby).forEach(key => {
      if (!Object.prototype.hasOwnProperty.call(row, key)) {
        // eslint-disable-next-line @typescript-eslint/no-dynamic-delete
        delete lobby[key]
      }
    })
    const want = row as any
    Object.keys(row).forEach(key => {
      if (`${lobby[key] as string}` !== `${want[key] as string}`) {
        throw new Error(`expected ${key} to be ${want[key] as string} but got ${lobby[key] as string}`)
      }
    })
  })
  if (player.lastReceivedLobbies.length !== expectedLobbies.hashes().length) {
    throw new Error(`expected ${expectedLobbies.hashes().length} lobbies but got ${player.lastReceivedLobbies.length}`)
  }
})

When('{string} has not seen the {string} event', function (this: World, playerName: string, eventName: string) {
  const player = this.players.get(playerName)
  if (player == null) {
    throw new Error('no such player')
  }
  if (player.findEvent(eventName) !== undefined) {
    throw new Error(`${playerName} has recieved a ${eventName} event`)
  }
})

When('{string} disconnected from the signaling server', function (this: World, playerName: string) {
  const player = this.players.get(playerName)
  if (player == null) {
    throw new Error('no such player')
  }
  ;(player.network as any).signaling.ws.close()
})

When('the websocket of {string} is reconnected', function (this: World, playerName: string) {
  const player = this.players.get(playerName)
  if (player == null) {
    throw new Error('no such player')
  }
  player.network._forceReconnectSignaling()
})

Given('{string} is the leader of the lobby', async function (this: World, playerName: string) {
  const player = this.players.get(playerName)
  if (player == null) {
    throw new Error('no such player')
  }
  if (player.network.currentLeader !== player.network.id) {
    throw new Error('player is not the leader')
  }
})

Given('{string} becomes the leader of the lobby', async function (this: World, playerName: string) {
  const player = this.players.get(playerName)
  if (player == null) {
    throw new Error('no such player')
  }
  const event = await player.waitForEvent('leader', [player.network.id], false)
  if (event == null) {
    throw new Error(`no event leader(${player.network.id}) received`)
  }
  if (player.network.currentLeader !== player.network.id) {
    throw new Error('player is not the leader')
  }
})

When('{string} updates the lobby with these settings:', async function (this: World, playerName: string, settings: string) {
  const player = this.players.get(playerName)
  if (player == null) {
    throw new Error('no such player')
  }
  const r = await player.network.setLobbySettings(JSON.parse(settings))
  if (r !== true) {
    throw new Error(`failed to update lobby: ${r.message}`)
  }
})

When('{string} fails to update the lobby with these settings:', async function (this: World, playerName: string, settings: string) {
  const player = this.players.get(playerName)
  if (player == null) {
    throw new Error('no such player')
  }
  try {
    await player.network.setLobbySettings(JSON.parse(settings))
  } catch (e) {
    return // we expect this to fail
  }
  throw new Error('no error thrown')
})

Then('the latest error for {string} is {string}', function (playerName: string, message: string) {
  const error = this.lastError.get(playerName)
  if (error === undefined) {
    throw new Error('no error thrown')
  } else if (error.message !== message) {
    throw new Error(`expected error to be '${message}' but got '${error.message as string}'`)
  }
})

Then('{string} failed to join the lobby', function (playerName: string) {
  const player = this.players.get(playerName)
  if (player === null) {
    throw new Error('no such player')
  }

  if (player.network.currentLobby !== undefined) {
    throw new Error(`player is in lobby ${player.network.currentLobby as string}`)
  }
})
