export const DefaultSignalingURL = 'ws://localhost:8080/v0/signaling'

export const DefaultRTCConfiguration: RTCConfiguration = {}

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
