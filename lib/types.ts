
const JOIN = 'join'
const LEAVE = 'leave'
const CONNECT = 'connect'
const CANDIDATE = 'candidate'
const DESCRIPTION = 'description'
const CONNECTED = 'connected'
const DISCONNECTED = 'disconnected'

interface Base {
  type: string
}

interface Signaling extends Base {
}

export interface JoinPacket extends Signaling {
  type: typeof JOIN
  game: string
  lobby?: string
  id?: string
}

export interface LeavePacket extends Signaling {
  type: typeof LEAVE
  id: string
  reason: string
}

export interface ConnectPacket extends Signaling {
  type: typeof CONNECT
  id: string
  ref: string
  polite: boolean
}

export interface ConnectedPacket extends Signaling {
  type: typeof CONNECTED
  ref: string
}

export interface DisconnectedPacket extends Signaling {
  type: typeof DISCONNECTED
  ref: string
  reason: string
}

export interface CandidatePacket extends Signaling {
  type: typeof CANDIDATE
  ref: string
  candidate: RTCIceCandidate | null
}

export interface DescriptionPacket extends Signaling {
  type: typeof DESCRIPTION
  ref: string
  description: RTCSessionDescription
}

export type SignalingPacketTypes = JoinPacket | LeavePacket | ConnectPacket | CandidatePacket | DescriptionPacket | ConnectedPacket | DisconnectedPacket
