import { PokiTurnMatch } from './credentials'
import { PeerConfiguration } from './types'

export const DefaultSignalingURL = process.env.NODE_ENV === 'production' ? 'wss://netlib.poki.io/v0/signaling' : 'ws://localhost:8080/v0/signaling'

export const DefaultRTCConfiguration: PeerConfiguration = {
  iceServers: [
    {
      urls: [
        'stun:stun.l.google.com:19302'
      ]
    },
    {
      urls: PokiTurnMatch
    }
  ]
}

export const DefaultDataChannels: { [label: string]: RTCDataChannelInit } = {
  reliable: {
    ordered: true
  },
  unreliable: {
    ordered: true,
    maxRetransmits: 0
  },
  control: {
    ordered: false
  }
}

export { default as Network } from './network'
