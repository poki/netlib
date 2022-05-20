package signaling

import "github.com/poki/netlib/internal/cloudflare"

type HelloPacket struct {
	Type string `json:"type"`

	Game  string `json:"game"`
	ID    string `json:"id"`
	Lobby string `json:"lobby"`
}

type WelcomePacket struct {
	Type string `json:"type"`

	ID string `json:"id"`
}

type CreatePacket struct {
	Type string `json:"type"`
}

type JoinPacket struct {
	Type string `json:"type"`

	Lobby string `json:"lobby"`
}

type JoinedPacket struct {
	Type string `json:"type"`

	Lobby string `json:"lobby"`
}

type ConnectPacket struct {
	Type string `json:"type"`

	ID     string `json:"id"`
	Polite bool   `json:"polite"`
}

type DisconnectPacket struct {
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
	cloudflare.Credentials
	Type string `json:"type"`
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
