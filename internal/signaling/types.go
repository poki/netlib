package signaling

import (
	"encoding/json"

	"github.com/poki/netlib/internal/cloudflare"
	"github.com/poki/netlib/internal/metrics"
	"github.com/poki/netlib/internal/signaling/stores"
)

type PingPacket struct {
	Type string `json:"type"`
}

type HelloPacket struct {
	Type string `json:"type"`

	Game   string `json:"game"`
	ID     string `json:"id"`
	Secret string `json:"secret"`
	Lobby  string `json:"lobby"`
}

type WelcomePacket struct {
	Type string `json:"type"`

	ID     string `json:"id"`
	Secret string `json:"secret"`
}

type ListPacket struct {
	RequestID string `json:"rid"`
	Type      string `json:"type"`

	Filter string `json:"filter"`
}

type LobbiesPacket struct {
	RequestID string `json:"rid"`
	Type      string `json:"type"`

	Lobbies []stores.Lobby `json:"lobbies"`
}

type CreatePacket struct {
	RequestID string `json:"rid"`
	Type      string `json:"type"`

	CodeFormat string         `json:"codeFormat"`
	Public     bool           `json:"public"`
	Password   string         `json:"password"`
	MaxPlayers int            `json:"maxPlayers"`
	CustomData map[string]any `json:"customData"`
}

type JoinPacket struct {
	RequestID string `json:"rid"`
	Type      string `json:"type"`

	Lobby string `json:"lobby"`
}

type JoinedPacket struct {
	RequestID string `json:"rid"`
	Type      string `json:"type"`

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

type ClosePacket struct {
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
