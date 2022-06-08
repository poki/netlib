package metrics

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/koenbollen/logging"
	"github.com/poki/netlib/internal/util"
	"go.uber.org/zap"
)

const timeout = 5 * time.Second
const maxIdleConnsPerHost = 32
const maxConnsPerHost = 32

type EventParams struct {
	Game     string `json:"game"`
	Category string `json:"category"`
	Action   string `json:"action"`
	PeerID   string `json:"peer"`
	LobbyID  string `json:"lobby,omitempty"`

	Data map[string]string `json:"data,omitempty"`
}

type Client struct {
	url    string
	client http.Client
}

func NewClient(url string) *Client {
	c := &Client{
		url: url,
		client: http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxConnsPerHost:     maxConnsPerHost,
				MaxIdleConnsPerHost: maxIdleConnsPerHost,
				Dial: (&net.Dialer{
					Timeout: timeout,
				}).Dial,
				TLSHandshakeTimeout: timeout,
			},
		},
	}
	return c
}

func (c *Client) Record(ctx context.Context, category, action, game, peerID, lobbyID string, data ...string) {
	if len(data)%2 != 0 {
		panic("data must be pairs")
	}
	dataMap := make(map[string]string)
	for i := 0; i < len(data); i += 2 {
		dataMap[data[i]] = data[i+1]
	}
	c.RecordEvent(ctx, EventParams{
		Category: category,
		Action:   action,
		Game:     game,
		PeerID:   peerID,
		LobbyID:  lobbyID,
		Data:     dataMap,
	})
}

func (c *Client) RecordEvent(ctx context.Context, params EventParams) {
	logger := logging.GetLogger(ctx)
	now := util.Now(ctx)
	remoteAddr, _ := ctx.Value(remoteAddrKey).(string)
	userAgent, _ := ctx.Value(userAgentKey).(string)

	event := &Event{
		Time:    now.UnixMilli(),
		Client:  remoteAddr,
		Version: os.Getenv("VERSION"),
		Game:    params.Game,

		Category: params.Category,
		Action:   params.Action,
		Peer:     params.PeerID,
		Lobby:    params.LobbyID,

		Data: params.Data,
	}

	payload, err := json.Marshal(event)
	if err != nil {
		logger.Error("failed to marshal event", zap.Error(err))
		return
	}

	// Use a new context, we want to record events of users that are already disconnected.
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(payload))
	if err != nil {
		logger.Error("failed to create metrics request", zap.Error(err))
		return
	}

	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		logger.Error("failed execute metrics request", zap.Error(err))
		return
	}
	io.Copy(io.Discard, resp.Body) //nolint:errcheck
	resp.Body.Close()              //nolint:errcheck

	if resp.StatusCode != http.StatusNoContent {
		logger.Error("unexpected status code from metrics endpoint", zap.Int("status", resp.StatusCode))
		return
	}
}
