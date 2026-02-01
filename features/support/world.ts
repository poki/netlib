import { After, AfterAll, Before, BeforeAll, setWorldConstructor, World as CucumberWorld, setDefaultTimeout } from '@cucumber/cucumber'
import { spawn } from 'child_process'
import { unlinkSync } from 'fs'

import fetch from 'node-fetch'
import ws from 'ws'
import wrtc from '@roamhq/wrtc'

import { Player } from './types'
import { PeerConfiguration } from '../../lib/types'

import { Network } from '../../lib'

;(global as any).fetch = fetch
;(global as any).WebSocket = ws
;(global as any).RTCPeerConnection = wrtc.RTCPeerConnection

process.env.NODE_ENV = 'test'

let anyScenarioFailed = false

interface backend {
  process: ReturnType<typeof spawn>
  port: number
  wait: Promise<void>
}

setDefaultTimeout(120 * 1000)

export class World extends CucumberWorld {
  public scenarioRunning: boolean = false

  public backends: Map<string, backend> = new Map<string, backend>()

  public signalingURL?: string
  public testproxyURL?: string
  public useTestProxy: boolean = false
  public databaseURL?: string

  public players: Map<string, Player> = new Map<string, Player>()
  public lastError: Map<string, Error> = new Map<string, Error>()

  public print (message: string): void {
    if (this.scenarioRunning) {
      void this.attach(message)
    } else {
      // void this.attach(message)
    }
  }

  public async createPlayer (playerName: string, gameID: string, country?: string, region?: string): Promise<Player> {
    return await new Promise((resolve) => {
      const config: PeerConfiguration = {}
      if (this.useTestProxy) {
        config.testproxyURL = this.testproxyURL
      }
      let signalingURL = this.signalingURL
      if (signalingURL !== undefined && (country !== undefined || region !== undefined)) {
        const url = new URL(signalingURL)
        if (country !== undefined) {
          url.searchParams.set('country', country)
        }
        if (region !== undefined) {
          url.searchParams.set('region', region)
        }
        signalingURL = url.toString()
      }

      const network = new Network(gameID, config, signalingURL)
      const player = new Player(playerName, network)
      this.players.set(playerName, player)

      // Give the Network some time to connect to the signaling server.
      // Giving this some time makes our test less flaky.
      setTimeout(() => {
        resolve(player)
      }, 50)
    })
  }
}
setWorldConstructor(World)

BeforeAll(async () => await new Promise(resolve => {
  let c = 2;
  ['signaling', 'testproxy'].forEach(backend => {
    const proc = spawn('go', ['build', '-o', `/tmp/netlib-cucumber-${backend}`, `cmd/${backend}/main.go`], {
      windowsHide: true,
      stdio: 'inherit'
    })
    proc.on('close', () => {
      if (proc.exitCode !== 0) {
        console.log('failed to compile', backend)
        process.exit(1)
      }
      if (--c === 0) {
        resolve(undefined)
      }
    })
  })
}))

AfterAll(function () {
  unlinkSync('/tmp/netlib-cucumber-signaling')
  unlinkSync('/tmp/netlib-cucumber-testproxy')

  // node-webrtc seem to always SEGFAULT when the process is killed, this is
  // a quick workaround to make sure the process is killed neatly.
  // source: https://github.com/node-webrtc/node-webrtc/issues/636#issuecomment-774171409
  process.on('beforeExit', (code) => process.exit(code))

  setTimeout(() => {
    console.log('cucumber did not exit cleanly, forcing exit')
    process.exit(anyScenarioFailed ? 1 : 0)
  }, 2000).unref()
})

Before(function (this: World) {
  this.scenarioRunning = true
})
After(function (this: World, { result }) {
  this.scenarioRunning = false

  if (result?.status === 'FAILED') {
    anyScenarioFailed = true
  }
})
