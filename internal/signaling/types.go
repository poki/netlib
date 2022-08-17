package signaling

import (
	"encoding/json"

	"github.com/poki/netlib/internal/cloudflare"
	"github.com/poki/netlib/internal/metrics"
)

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

type EventPacket struct {
	metrics.EventParams
	Type string `json:"type"`
}

type MissingRecipientError struct {
	Recipient string `json:"recipient"`
	Cause     error  `json:"cause"`
}

func (e *MissingRecipientError) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"recipient": e.Recipient,
		"cause":     e.Cause.Error(),
	})
}

func (e *MissingRecipientError) Error() string {
	return "missing recipient: " + e.Recipient
}

func (e *MissingRecipientError) ErrorCode() string {
	return "missing-recipient"
}

func (e *MissingRecipientError) Unwrap() error {
	return e.Cause
}
