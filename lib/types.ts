
export interface PeerConfiguration extends RTCConfiguration {
  /**
   * @internal
   */
  testproxyURL?: string
}

export interface LobbySettings {
  codeFormat?: 'default' | 'short'
  codeLength?: number
  maxPlayers?: number
  password?: string
  public?: boolean
  customData?: {[key: string]: any}
}

export interface LobbyListEntry {
  code: string
  public: boolean
  playerCount: number
  customData?: {[key: string]: any}
  leader: string | null
  term: number
  createdAt: string
  updatedAt: string
}

interface Base {
  type: string
  rid?: string
}

export type SignalingPacketTypes =
| CandidatePacket
| ClosePacket
| ConnectedPacket
| ConnectPacket
| CreatePacket
| CredentialsPacket
| DescriptionPacket
| DisconnectedPacket
| DisconnectPacket
| ErrorPacket
| EventPacket
| HelloPacket
| JoinedPacket
| JoinPacket
| LeaderPacket
| ListPacket
| LobbiesPacket
| PingPacket
| WelcomePacket

export interface PingPacket extends Base {
  type: 'ping'
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
  secret?: string
}

export interface WelcomePacket extends Base {
  type: 'welcome'
  id: string
  secret: string
}

export interface ListPacket extends Base {
  type: 'list'
  filter?: string
}

export interface LobbiesPacket extends Base {
  type: 'lobbies'
  lobbies: LobbyListEntry[]
}

export interface CreatePacket extends Base {
  type: 'create'
  settings?: LobbySettings
}

export interface JoinPacket extends Base {
  type: 'join'
  lobby: string
}

export interface JoinedPacket extends Base {
  type: 'joined'
  lobbyInfo: LobbyListEntry
}

export interface LeaderPacket extends Base {
  type: 'leader'
  leader: string
  term: number
}

export interface ClosePacket extends Base {
  type: 'close'
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
