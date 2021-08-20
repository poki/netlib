import { spawn } from 'child_process'
import { After, Given } from '@cucumber/cucumber'
import { World } from '../world'

Given('the {string} backend is running', async function (this: World, backend: string) {
  return await new Promise(resolve => {
    const port = 10000 + Math.ceil(Math.random() * 1000)
    const prc = spawn(`/tmp/netlib-cucumber-${backend}`, [], {
      windowsHide: true,
      env: {
        ...process.env,
        ADDR: `127.0.0.1:${port}`,
        ENV: 'test'
      }
    })
    prc.stderr.setEncoding('utf8')
    prc.stderr.on('data', (data: string) => {
      const lines = data.split('\n')
      lines.forEach(line => {
        try {
          const entry = JSON.parse(line)
          if (entry.message === 'listening') {
            resolve(undefined)
          }
        } catch (_) {
        }
        this.print(line)
      })
    })
    prc.addListener('exit', () => {
      this.print(`${backend} exited`)
    })
    prc.addListener('close', () => {
      this.print(`${backend} closed`)
    })
    this.signalingURL = `ws://127.0.0.1:${port}/v0/signaling`
    this.backend = prc
  })
})

After(function (this: World) {
  this.print('killin')
  this.backend?.kill()
})
