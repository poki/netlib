
export interface PeerConfiguration extends RTCConfiguration {
  testproxyURL?: string
}

interface Base {
  type: string
}

interface Signaling extends Base {
}

export interface CreatePacket extends Signaling {
  type: 'create'
  game: string
}

export interface JoinPacket extends Signaling {
  type: 'join'
  game: string
  lobby: string
  id?: string
}

export interface JoinedPacket extends Signaling {
  type: 'joined'
  lobby: string
  id: string
}

export interface LeavePacket extends Signaling {
  type: 'leave'
  id: string
  reason: string
}

export interface ConnectPacket extends Signaling {
  type: 'connect'
  id: string
  polite: boolean
}

export interface ConnectedPacket extends Signaling {
  type: 'connected'
  id: string
}

export interface DisconnectedPacket extends Signaling {
  type: 'disconnected'
  id: string
  reason: string
}

export interface CandidatePacket extends Signaling {
  type: 'candidate'
  source: string
  recipient: string
  candidate: RTCIceCandidate | null
}

export interface DescriptionPacket extends Signaling {
  type: 'description'
  source: string
  recipient: string
  description: RTCSessionDescription
}

export interface CredentialsPacket extends Signaling {
  type: 'credentials'

  url?: string
  username?: string
  credential?: string
  lifetime?: number
}

export type SignalingPacketTypes = CreatePacket | JoinPacket | JoinedPacket | LeavePacket | ConnectPacket | CandidatePacket | DescriptionPacket | ConnectedPacket | DisconnectedPacket | CredentialsPacket
