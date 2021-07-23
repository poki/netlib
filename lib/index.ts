export const DefaultSignalingURL = 'ws://localhost:8080/v0/signaling'

export const DefaultRTCConfiguration: RTCConfiguration = {
  iceServers: [
    {
      urls: [
        'stun:stun.l.google.com:19302',
        'stun:stun3.l.google.com:19302'
      ]
    },
    {
      urls: [
        'turn:localhost:8080'
      ],
      username: 'optional-username',
      credential: 'secret',
      credentialType: 'password'
    }
  ]
}

export const DefaultDataChannels: {[label: string]: RTCDataChannelInit} = {
  reliable: {
    ordered: true
  },
  unreliable: {
    ordered: false,
    maxRetransmits: 0
  }
}

export { default as Network } from './network'
