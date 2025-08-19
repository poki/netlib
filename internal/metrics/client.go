package metrics

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/koenbollen/logging"
	"github.com/poki/netlib/internal/util"
	"github.com/rs/xid"
	"go.uber.org/zap"
)

const timeout = 10 * time.Second
const maxIdleConnsPerHost = 32
const maxConnsPerHost = 32
const maxRetries = 5
const backoffRange = 1000 // milliseconds, picked randomly from a range times the number of retries

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
	now := util.NowUTC(ctx)
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

	idempotency := xid.New().String()
	logger = logger.With(zap.String("idempotency", idempotency))

	for i := range maxRetries {
		if i > 0 {
			time.Sleep(time.Duration(rand.Int63n(backoffRange)*int64(i)) * time.Millisecond)
		}

		// Use a new context, we want to record events of users that are already disconnected.
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(payload))
		if err != nil {
			logger.Error("failed to create metrics request", zap.Error(err))
			return
		}
		req.Header.Set("X-Idempotency-ID", idempotency)
		if userAgent != "" {
			req.Header.Set("User-Agent", userAgent)
		}

		resp, err := c.client.Do(req)
		if err != nil {
			if i < maxRetries-1 {
				logger.Warn("failed execute metrics request, retrying", zap.Int("attempt", i), zap.Error(err))
			} else {
				logger.Error("failed execute metrics request", zap.Error(err))
			}
			continue
		}
		io.Copy(io.Discard, resp.Body) //nolint:errcheck
		resp.Body.Close()              //nolint:errcheck

		if resp.StatusCode != http.StatusNoContent {
			logger.Error("unexpected status code from metrics endpoint", zap.Int("status", resp.StatusCode))
			if resp.StatusCode/100 == 5 {
				continue
			}
			return
		}

		return
	}
}
