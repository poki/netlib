import { When } from '@cucumber/cucumber'
import { World } from '../world'

When('I sleep for {int} second', async function (this: World, seconds: number) {
  await new Promise(resolve => setTimeout(resolve, seconds * 1000))
})

When('timeout disconnect threshold passes', async function (this: World) {
  await new Promise(resolve => setTimeout(resolve, 1000))
})

let time: number = 0
When(/I (start|stop) measuring time/, function (this: World, act: string) {
  if (act === 'start') {
    time = performance.now()
  } else if (act === 'stop') {
    const now = performance.now()
    console.log('took', now - time)
  }
})
