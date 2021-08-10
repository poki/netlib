package signaling

type CreatePacket struct {
	Type string `json:"type"`

	Game string `json:"game"`
}

type JoinPacket struct {
	Type string `json:"type"`

	Game  string `json:"game"`
	Lobby string `json:"lobby"`
}

type JoinOrCreatePacket struct {
	Type string `json:"type"`

	Game   string `json:"game"`
	Prefix string `json:"prefix"`
}

type JoinedPacket struct {
	Type string `json:"type"`

	Lobby string `json:"lobby"`
	ID    string `json:"id"`
}

type ConnectPacket struct {
	Type string `json:"type"`

	ID     string `json:"id"`
	Polite bool   `json:"polite"`
}

type LeavePacket struct {
	Type string `json:"type"`

	ID     string `json:"id"`
	Reason string `json:"reason"`
}

type ForwardablePacket struct {
	Type string `json:"type"`

	Source    string `json:"source"`
	Recipient string `json:"recipient"`
}
