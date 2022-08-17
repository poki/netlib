
export interface PeerConfiguration extends RTCConfiguration {
  testproxyURL?: string
}

interface Base {
  type: string
}

export interface ErrorPacket extends Base {
  type: 'error'
  message: string
  error?: any
  code?: string
}

export interface HelloPacket extends Base {
  type: 'hello'
  game: string
  id?: string
  lobby?: string
}

export interface WelcomePacket extends Base {
  type: 'welcome'
  id: string
}

export interface CreatePacket extends Base {
  type: 'create'
}

export interface JoinPacket extends Base {
  type: 'join'
  lobby: string
}

export interface JoinedPacket extends Base {
  type: 'joined'
  lobby: string
  id: string
}

export interface LeavePacket extends Base {
  type: 'leave'
  id: string
  reason: string
}

export interface ConnectPacket extends Base {
  type: 'connect'
  id: string
  polite: boolean
}

export interface DisconnectPacket extends Base {
  type: 'disconnect'
  id: string
}

export interface ConnectedPacket extends Base {
  type: 'connected'
  id: string
}

export interface DisconnectedPacket extends Base {
  type: 'disconnected'
  id: string
  reason: string
}

export interface CandidatePacket extends Base {
  type: 'candidate'
  source: string
  recipient: string
  candidate: RTCIceCandidate | null
}

export interface DescriptionPacket extends Base {
  type: 'description'
  source: string
  recipient: string
  description: RTCSessionDescription
}

export interface CredentialsPacket extends Base {
  type: 'credentials'

  url?: string
  username?: string
  credential?: string
  lifetime?: number
}

export interface EventPacket extends Base {
  type: 'event'

  game: string
  category: string
  action: string
  peer: string
  lobby?: string

  data?: {[key: string]: string}
}

export type SignalingPacketTypes = ErrorPacket | HelloPacket | WelcomePacket | CreatePacket | JoinPacket | JoinedPacket | LeavePacket | ConnectPacket | CandidatePacket | DescriptionPacket | ConnectedPacket | DisconnectPacket | DisconnectedPacket | CredentialsPacket | EventPacket
