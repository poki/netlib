import { Network } from '../lib'

const n = new Network('d0fe1ca1-7fa0-47ed-9469-6c792f68bae0')
;(window as any).n = n

const inp = document.getElementById('input') as HTMLInputElement
const out = document.getElementById('output') as HTMLTextAreaElement
const log = (text: string): void => {
  console.log(text)
  if (out?.innerText != null) {
    const time = (new Date()).toLocaleTimeString()
    out.value += `[${time}] ${text.trim()}\n`
    out.scrollTop = out.scrollHeight
  }
}

n.on('ready', () => {
  log('network ready')

  n.on('message', (peer, channel, data) => {
    if (channel === 'reliable') {
      log(`${peer.id} said "${data as string}" via ${channel}`)
    }
  })
  inp.addEventListener('keyup', e => {
    if (e.key === 'Enter') {
      log(`sending ${inp.value}`)
      n.broadcast('reliable', inp.value)
      inp.value = ''
    }
  })

  document.querySelector('a[data-action="create"]')?.addEventListener('click', () => {
    if (n.currentLobby === undefined) {
      void n.create({ codeFormat: 'short' })
    }
  })

  document.querySelector('a[data-action="join"]')?.addEventListener('click', () => {
    if (n.currentLobby === undefined) {
      const code = prompt('Lobby code? (empty to create a new one)')
      if (code != null && code !== '') {
        void n.join(code)
      }
    }
  })

  const queryLobbies = (): void => {
    console.log('querying lobbies...')
    void n.list().then(lobbies => {
      console.log('queried lobbies:', lobbies)
      const el = document.getElementById('lobbies')
      if (el !== null) {
        el.innerHTML = ''
        if (lobbies === null || lobbies.length === 0) {
          const li = document.createElement('li')
          li.innerHTML = '<i>no lobbies</i>'
          el.appendChild(li)
        } else {
          lobbies.forEach(lobby => {
            const li = document.createElement('li')
            li.id = lobby.code
            li.innerHTML = `<a href="javascript:void(0)" class="code">${lobby.code}</a> - <span class="players">${lobby.playerCount}</span> players`
            el.appendChild(li)
            if (n.currentLobby === undefined) {
              li.querySelector('a.code')?.addEventListener('click', () => {
                void n.join(lobby.code)
              })
            }
          })
        }
      }
    })
  }
  queryLobbies()
  setInterval(queryLobbies, 5000)
})

n.on('lobby', code => {
  log(`lobby code ready: ${code} (and you are ${n.id})`)
})

n.on('signalingerror', console.error.bind(console.error))
n.on('rtcerror', console.error.bind(console.error))

n.on('connecting', peer => { log(`peer connecting ${peer.id}`) })
n.on('disconnected', peer => {
  log(`peer disconnected ${peer.id} (${n.size} peers now)`)
  document.getElementById(peer.id)?.remove()
})

n.on('connected', peer => {
  (window as any).peer = peer

  log(`peer connected: ${peer.id} (${n.size} peers now)`)
  n.broadcast('reliable', `got new peer! ${peer.id}`)

  const li = document.createElement('li')
  li.id = peer.id
  li.innerHTML = `${peer.id} (ping: <span>0</span>)`
  document.getElementById('peers')?.appendChild(li)

  setInterval(() => {
    (li.querySelector('span') as any).innerHTML = `${peer.latency.average.toFixed(1)}ms ${peer.latency.jitter.toFixed(1)}ms`
    n.send('unreliable', peer.id, 'bogus data')
  }, 16)
})
