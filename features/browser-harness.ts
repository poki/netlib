import { Network } from '../lib'
import { LobbyListEntry, PeerConfiguration } from '../lib/types'

export const browserHarnessLoaded = true

declare global {
  interface Window {
    netlibTest: any
    netlibTestReady: boolean
    netlibTestLoadErrors?: string[]
  }
}

interface RecordedEvent {
  eventName: string
  eventPayload: any[]
}

interface PlayerOptions {
  gameID: string
  signalingURL: string
  testproxyURL?: string
}

const allEvents = ['close', 'ready', 'lobby', 'left', 'connected', 'disconnected', 'reconnecting', 'reconnected', 'message', 'signalingerror', 'signalingreconnected', 'leader', 'lobbyUpdated']

let network: Network | undefined
let events: RecordedEvent[] = []
let scanIndex = 0
let lastReceivedLobbies: LobbyListEntry[] = []
let lastError: string | null = null

function requireNetwork (): Network {
  if (network === undefined) {
    throw new Error('network has not been created')
  }
  return network
}

function serializeArg (arg: any): any {
  if (arg === undefined) {
    return null
  }
  if (arg === null) {
    return null
  }
  if (typeof arg === 'object') {
    if (!Array.isArray(arg)) {
      const text = String(arg)
      if (text !== '[object Object]') {
        return { __netlibString: text }
      }
    }
    try {
      return JSON.parse(JSON.stringify(arg))
    } catch (_) {
      return { __netlibString: String(arg) }
    }
  }
  return arg
}

function stringifyArg (arg: any): string {
  if (arg !== null && typeof arg === 'object' && typeof arg.__netlibString === 'string') {
    return arg.__netlibString
  }
  return String(arg)
}

function normalizeComparable (arg: any): any {
  if (arg === null || typeof arg !== 'object') {
    return arg
  }
  if (Array.isArray(arg)) {
    return arg.map(normalizeComparable)
  }
  const copy: Record<string, any> = {}
  Object.keys(arg)
    .filter(key => key !== 'createdAt' && key !== 'updatedAt')
    .sort()
    .forEach(key => {
      copy[key] = normalizeComparable(arg[key])
    })
  return copy
}

function debugArg (arg: any): string {
  if (arg !== null && typeof arg === 'object') {
    try {
      return JSON.stringify(normalizeComparable(arg))
    } catch (_) {
      return stringifyArg(arg)
    }
  }
  return stringifyArg(arg)
}

function matchEvent (event: RecordedEvent, eventName: string, matchArguments: any[] = []): boolean {
  if (event.eventName !== eventName) {
    return false
  }
  let argumentsMatch = true
  for (let i = 0; i < matchArguments.length; i++) {
    const expected = matchArguments[i]
    const actual = event.eventPayload[i]
    if (typeof expected === 'string' || expected instanceof String) {
      argumentsMatch = stringifyArg(actual) === expected
    } else if (actual !== null && typeof actual === 'object') {
      argumentsMatch = JSON.stringify(normalizeComparable(actual)) === JSON.stringify(normalizeComparable(expected))
    } else if (actual !== expected) {
      argumentsMatch = false
    } else {
      argumentsMatch = true
    }
  }
  return argumentsMatch
}

function findEvent (eventName: string, matchArguments: any[] = [], fromScanIndex: boolean = false): RecordedEvent | null {
  const offset = fromScanIndex ? scanIndex : 0
  const ix = events.slice(offset).findIndex(event => matchEvent(event, eventName, matchArguments))
  if (ix < 0) {
    return null
  }
  return events[offset + ix]
}

function consumeEvent (event: RecordedEvent): void {
  const ix = events.slice(scanIndex).indexOf(event)
  if (ix >= 0) {
    scanIndex += ix + 1
  }
}

async function delay (ms: number): Promise<void> {
  return await new Promise(resolve => setTimeout(resolve, ms))
}

function sdpCandidateLines (sdp?: string): string[] {
  return (sdp ?? '').split('\r\n').filter(line => line.startsWith('a=candidate'))
}

function peerDiagnostics (): any[] {
  if (network === undefined) {
    return []
  }
  return Array.from(network.peers.entries()).map(([id, peer]: [string, any]) => ({
    id,
    connectionState: peer.conn.connectionState,
    iceConnectionState: peer.conn.iceConnectionState,
    iceGatheringState: peer.conn.iceGatheringState,
    signalingState: peer.conn.signalingState,
    localCandidates: sdpCandidateLines(peer.conn.localDescription?.sdp),
    remoteCandidates: sdpCandidateLines(peer.conn.remoteDescription?.sdp)
  }))
}

async function waitForEvent (eventName: string, matchArguments: any[] = [], consume: boolean = true, timeoutMs: number = 20000): Promise<RecordedEvent> {
  if (!allEvents.includes(eventName)) {
    throw new Error(`Event type ${eventName} not tracked, add to allEvents in browser-harness.ts`)
  }

  const deadline = performance.now() + timeoutMs
  for (;;) {
    const event = findEvent(eventName, matchArguments, true)
    if (event !== null) {
      if (consume) {
        consumeEvent(event)
      }
      return event
    }
    if (performance.now() >= deadline) {
      const sameEvents = events.slice(scanIndex)
        .filter(event => event.eventName === eventName)
        .map(event => event.eventPayload.map(arg => debugArg(arg)).join(','))
        .join(' + ')
      throw new Error(`Event not found, timed out, got: ${sameEvents}; peers: ${JSON.stringify(peerDiagnostics())}`)
    }
    await delay(100)
  }
}

function registerEvents (n: Network): void {
  allEvents.forEach(eventName => {
    n.on(eventName as any, (...args: any[]) => {
      events.push({
        eventName,
        eventPayload: args.map(serializeArg)
      })
    })
  })

  n.on('signalingerror', _ => {})
  n.on('rtcerror', _ => {})
}

async function createPlayer (options: PlayerOptions): Promise<void> {
  events = []
  scanIndex = 0
  lastReceivedLobbies = []
  lastError = null

  const config: PeerConfiguration = {
    iceServers: []
  }
  if (options.testproxyURL !== undefined && options.testproxyURL !== '') {
    config.testproxyURL = options.testproxyURL
  }

  network = new Network(options.gameID, config, options.signalingURL)
  registerEvents(network)

  // Keep parity with the old Cucumber world, which gave the signaling socket a
  // short head start before the caller waited for concrete events.
  await delay(50)
}

async function createLobby (settings?: any): Promise<string> {
  return await requireNetwork().create(settings)
}

async function joinLobby (lobby: string, password?: string): Promise<LobbyListEntry | undefined> {
  lastError = null
  return await requireNetwork().join(lobby, password)
}

async function tryJoinLobby (lobby: string, password?: string): Promise<{ ok: boolean, message?: string }> {
  try {
    await joinLobby(lobby, password)
    return { ok: true }
  } catch (e) {
    lastError = e !== null && typeof e === 'object' && typeof (e as any).message === 'string' ? (e as any).message : String(e)
    return { ok: false, message: lastError }
  }
}

async function listLobbies (filter?: object, sort?: object, limit?: number): Promise<LobbyListEntry[]> {
  filter = filter ?? undefined
  sort = sort ?? undefined
  limit = limit ?? undefined
  lastReceivedLobbies = await requireNetwork().list(filter, sort, limit)
  return lastReceivedLobbies
}

async function setLobbySettings (settings: any): Promise<void> {
  const result = await requireNetwork().setLobbySettings(settings)
  if (result !== true) {
    throw result
  }
}

function broadcast (channel: string, data: string): void {
  requireNetwork().broadcast(channel, data)
}

async function closeNetwork (reason?: string): Promise<void> {
  const n = requireNetwork() as any
  const ws = n.signaling?.ws
  const closed = new Promise<void>(resolve => {
    if (ws === undefined || ws.readyState === WebSocket.CLOSED) {
      resolve()
      return
    }
    ws.addEventListener('close', () => resolve(), { once: true })
  })
  requireNetwork().close(reason)
  await Promise.race([closed, delay(100)])
}

async function leaveLobby (): Promise<void> {
  await requireNetwork().leave()
}

function closeSignalingSocket (): void {
  const n = requireNetwork() as any
  n.signaling.ws.close()
}

function forceReconnectSignaling (): void {
  requireNetwork()._forceReconnectSignaling()
}

function disableTestProxy (): void {
  requireNetwork().peers.forEach(peer => {
    peer.config.testproxyURL = undefined
  })
}

function uninterruptOnConnectionState (otherPeerID: string, state: string, testproxyURL: string): void {
  const n = requireNetwork()
  const peer = n.peers.get(otherPeerID)
  if (peer === undefined) {
    throw new Error(`peer ${otherPeerID} not found`)
  }
  const uninterrupt = (): void => {
    fetch(`${testproxyURL}/uninterrupt?id=${n.id + otherPeerID}`).then(() => {}).catch(console.error)
    fetch(`${testproxyURL}/uninterrupt?id=${otherPeerID + n.id}`).then(() => {}).catch(console.error)
  }
  const onConnectionStateChange = (): void => {
    if (peer.conn.connectionState === state) {
      peer.conn.removeEventListener('connectionstatechange', onConnectionStateChange)
      uninterrupt()
    }
  }
  peer.conn.addEventListener('connectionstatechange', onConnectionStateChange)
  onConnectionStateChange()
}

function getID (): string {
  return requireNetwork().id
}

function getPeerCount (): number {
  return requireNetwork().peers.size
}

function getCurrentLobby (): string {
  return requireNetwork().currentLobby ?? ''
}

function getCurrentLeader (): string {
  return requireNetwork().currentLeader ?? ''
}

function getLastReceivedLobbies (): LobbyListEntry[] {
  return lastReceivedLobbies
}

function getLastError (): string {
  return lastError ?? ''
}

window.netlibTest = {
  createPlayer,
  createLobby,
  joinLobby,
  tryJoinLobby,
  listLobbies,
  setLobbySettings,
  broadcast,
  closeNetwork,
  leaveLobby,
  closeSignalingSocket,
  forceReconnectSignaling,
  disableTestProxy,
  uninterruptOnConnectionState,
  waitForEvent,
  findEvent,
  findNewEvent: (eventName: string, matchArguments: any[] = []) => findEvent(eventName, matchArguments, true),
  getID,
  getPeerCount,
  getCurrentLobby,
  getCurrentLeader,
  getLastReceivedLobbies,
  getLastError
}
window.netlibTestReady = true
