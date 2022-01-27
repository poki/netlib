import { Given, When } from '@cucumber/cucumber'
import { World } from '../world'

Given('webrtc is intercepted by the testproxy', function (this: World) {
  this.useTestProxy = true
})

When('the connection between {string} and {string} is interrupted until the first {string} state', async function (this: World, player0Name: string, player1Name: string, state: string) {
  const player0 = this.players.get(player0Name)
  if (player0 == null) {
    throw new Error('no such player: ' + player0Name)
  }
  const player1 = this.players.get(player1Name)
  if (player1 == null) {
    throw new Error('no such player: ' + player1Name)
  }
  if (this.testproxyURL === undefined) {
    throw new Error('testproxy not active')
  }

  await fetch(`${this.testproxyURL}/interrupt?id=${player0.network.id + player1.network.id}`)
  await fetch(`${this.testproxyURL}/interrupt?id=${player1.network.id + player0.network.id}`)

  const connToPlayer1 = player0.network.peers.get(player1.network.id)?.conn
  if (connToPlayer1 !== undefined) {
    connToPlayer1.addEventListener('connectionstatechange', () => {
      if (connToPlayer1?.connectionState === state) {
        fetch(`${this.testproxyURL ?? ''}/uninterrupt?id=${player0.network.id + player1.network.id}`).then(() => {}).catch(console.error)
        fetch(`${this.testproxyURL ?? ''}/uninterrupt?id=${player1.network.id + player0.network.id}`).then(() => {}).catch(console.error)
      }
    })
  }
})
