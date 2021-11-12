declare module 'wrtc' {
  declare const wrtc: {
    MediaStream: MediaStream
    MediaStreamTrack: MediaStreamTrack
    RTCDataChannel: RTCDataChannel
    RTCDataChannelEvent: RTCDataChannelEvent
    RTCDtlsTransport: RTCDtlsTransport
    RTCIceCandidate: RTCIceCandidate
    RTCIceTransport: RTCIceTransport
    RTCPeerConnection: RTCPeerConnection
    RTCPeerConnectionIceEvent: RTCPeerConnectionIceEvent
    RTCRtpReceiver: RTCRtpReceiver
    RTCRtpSender: RTCRtpSender
    RTCRtpTransceiver: RTCRtpTransceiver
    RTCSctpTransport: RTCSctpTransport
    RTCSessionDescription: RTCSessionDescription
    getUserMedia: typeof navigator.mediaDevices['getUserMedia']
    mediaDevices: typeof navigator.mediaDevices
  }
  export type WRTC = typeof wrtc
  export default wrtc
}
