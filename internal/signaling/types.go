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

type DisconnectedPacket struct {
	Type string `json:"type"`

	ID     string `json:"id"`
	Reason string `json:"reason"`
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

type CredentialsPacket struct {
	Type string `json:"type"`

	URL        string `json:"url"`
	Username   string `json:"username"`
	Credential string `json:"credential"`
	Lifetime   int    `json:"lifetime"`
}

type MissingRecipientError struct {
	Recipient string
	Cause     error
}

func (e *MissingRecipientError) Error() string {
	return "missing recipient: " + e.Recipient
}

func (e *MissingRecipientError) Unwrap() error {
	return e.Cause
}
