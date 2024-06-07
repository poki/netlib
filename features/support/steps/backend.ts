import { spawn } from 'child_process'
import { After, Given } from '@cucumber/cucumber'
import { World } from '../world'

Given('the {string} backend is running', async function (this: World, backend: string) {
  return await new Promise((resolve, reject) => {
    const port = 10000 + Math.ceil(Math.random() * 1000)
    const env: NodeJS.ProcessEnv = {
      ...process.env,
      ADDR: `127.0.0.1:${port}`,
      ENV: 'test'
    }

    if (this.databaseURL !== undefined) {
      env.DATABASE_URL = this.databaseURL
    }

    const prc = spawn(`/tmp/netlib-cucumber-${backend}`, [], {
      windowsHide: true,
      env
    })
    prc.stderr.setEncoding('utf8')

    let resolved = false
    prc.stderr.on('data', (data: string) => {
      const lines = data.split('\n')
      lines.forEach(line => {
        if (line === '') {
          return
        }
        try {
          const entry = JSON.parse(line)
          const severity = entry.severity.toLowerCase()
          if (!resolved && (severity === 'error' || severity === 'emergency')) {
            return reject(new Error(`error before backend was started: ${entry.message as string}`))
          }
          if (entry.message === 'using database') {
            this.databaseURL = entry.url
          } else if (entry.message === 'listening') {
            resolved = true
            resolve(undefined)
          }
        } catch (_) {
        }
        this.print(line)
      })
    })
    prc.addListener('exit', () => {
      this.print(`${backend} exited`)

      if (!resolved) {
        reject(new Error(`${backend} exited before it was ready`))
      }
    })

    // Create a promise that resolves when the backend is closed so
    // we can block for it in the After step that kills the backend:
    const waiter = new Promise<void>(resolve => {
      prc.addListener('close', () => {
        this.print(`${backend} closed (exitcode: ${prc.exitCode ?? 0})`)
        prc.unref()
        prc.removeAllListeners()
        resolve()
      })
    })

    this.backends.set(backend, { process: prc, port, wait: waiter })
    switch (backend) {
      case 'signaling':
        this.signalingURL = `ws://127.0.0.1:${port}/v0/signaling`
        break
      case 'testproxy':
        this.testproxyURL = `http://127.0.0.1:${port}`
        break
    }
  })
})

After(async function (this: World) {
  for (const [key, backend] of this.backends) {
    this.print('killing ' + key)
    backend.process.kill()
    await backend.wait // wait for the backend to close
  }
})
