package cloudflare

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/koenbollen/logging"
	"go.uber.org/zap"
)

type CredentialsClient struct {
	zone     string
	appID    string
	authUser string
	authKey  string

	lifetime time.Duration

	mutex  sync.RWMutex
	cached *Credentials

	HasFetchedFirstCredentials bool
}

func NewCredentialsClient(zone, appID, user, key string, lifetime time.Duration) *CredentialsClient {
	c := &CredentialsClient{
		zone:     zone,
		appID:    appID,
		authUser: user,
		authKey:  key,

		lifetime: lifetime,
	}
	return c
}

func (c *CredentialsClient) Run(ctx context.Context) {
	logger := logging.GetLogger(ctx)

	if c.zone == "" {
		logger.Warn("no Cloudflare zone configured, not fetching credentials")
		return
	}

	for ctx.Err() == nil {
		logger.Info("refetching credentials")
		creds, err := c.fetchCredentials(context.Background())
		if err != nil {
			logger.Error("failed to fetch credentials", zap.Error(err))
			time.Sleep(1 * time.Minute)
			continue
		}
		c.mutex.Lock()
		c.cached = creds
		c.mutex.Unlock()

		select {
		case <-time.After(c.lifetime / 2):
			continue
		case <-ctx.Done():
			return
		}
	}
}

func (c *CredentialsClient) GetCredentials(ctx context.Context) (*Credentials, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	if c.cached == nil {
		return nil, errors.New("no credentials available")
	}
	return c.cached, nil
}

func (c *CredentialsClient) fetchCredentials(ctx context.Context) (*Credentials, error) {
	url := "https://api.cloudflare.com/client/v4/zones/" + c.zone + "/webrtc-turn/credential/" + c.appID
	body := strings.NewReader(fmt.Sprintf(`{"lifetime":%d}`, c.lifetime/time.Second))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Auth-Email", c.authUser)
	req.Header.Set("X-Auth-Key", c.authKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected error from Cloudflare: %s", resp.Status)
	}

	response := response{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode Cloudflare response: %w", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("cloudflare error: %v", response.Errors)
	}

	return &Credentials{
		URL:        response.URL(),
		Username:   response.Result.Userid,
		Credential: response.Result.Credential,
		Lifetime:   response.Result.Lifetime,
	}, nil
}
