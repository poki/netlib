import { After, AfterAll, Before, BeforeAll, setWorldConstructor, World as CucumberWorld, setDefaultTimeout } from '@cucumber/cucumber'
import { spawn } from 'child_process'
import { unlinkSync } from 'fs'

import fetch from 'node-fetch'
import ws from 'ws'
import wrtc from '@roamhq/wrtc'

import { Player } from './types'

import { Network } from '../../lib'

;(global as any).fetch = fetch
;(global as any).WebSocket = ws
;(global as any).RTCPeerConnection = wrtc.RTCPeerConnection

process.env.NODE_ENV = 'test'

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

  public print (message: string): void {
    if (this.scenarioRunning) {
      void this.attach(message)
    } else {
      // this.attach(message)
    }
  }

  public createPlayer (playerName: string, gameID: string): Player {
    const config = this.useTestProxy ? { testproxyURL: this.testproxyURL } : undefined
    const network = new Network(gameID, config, this.signalingURL)
    const player = new Player(playerName, network)
    this.players.set(playerName, player)
    return player
  }
}
setWorldConstructor(World)

BeforeAll((cb: Function) => {
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
        cb()
      }
    })
  })
})

AfterAll(function (this: World) {
  unlinkSync('/tmp/netlib-cucumber-signaling')
  unlinkSync('/tmp/netlib-cucumber-testproxy')

  // node-webrtc seem to always SEGFAULT when the process is killed, this is
  // a quick workaround to make sure the process is killed neatly.
  // source: https://github.com/node-webrtc/node-webrtc/issues/636#issuecomment-774171409
  process.on('beforeExit', (code) => process.exit(code))
})

Before(function (this: World) {
  this.scenarioRunning = true
})
After(function (this: World) {
  this.scenarioRunning = false
})
