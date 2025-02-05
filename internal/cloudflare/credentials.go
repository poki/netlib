package cloudflare

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/koenbollen/logging"
	"go.uber.org/zap"
)

type CredentialsClient struct {
	appID   string
	authKey string

	lifetime time.Duration

	mutex  sync.RWMutex
	cached *Credentials
}

func NewCredentialsClient(appID, key string, lifetime time.Duration) *CredentialsClient {
	c := &CredentialsClient{
		appID:   appID,
		authKey: key,

		lifetime: lifetime,
	}
	return c
}

func (c *CredentialsClient) Run(ctx context.Context) {
	if os.Getenv("ENV") != "production" && c.appID == "" {
		return
	}

	logger := logging.GetLogger(ctx)

	for ctx.Err() == nil {
		start := time.Now()
		logger.Info("refetching credentials")
		fetchctx, fetchcancel := context.WithTimeout(ctx, 2*time.Minute)
		creds, err := c.fetchCredentials(fetchctx)
		fetchcancel()
		if err != nil {
			logger.Error("failed to fetch credentials", zap.Error(err),
				zap.Duration("duration", time.Since(start)))
			time.Sleep(1 * time.Minute)
			continue
		}
		logger.Info("fetched credentials", zap.Duration("duration", time.Since(start)))

		c.mutex.Lock()
		c.cached = creds
		c.mutex.Unlock()

		logger.Info("credentials cache set")

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
	lifetime := c.lifetime / time.Second

	url := "https://rtc.live.cloudflare.com/v1/turn/keys/" + c.appID + "/credentials/generate"
	body := strings.NewReader(fmt.Sprintf(`{"ttl":%d}`, lifetime))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.authKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("unexpected error from Cloudflare: %s", resp.Status)
	}

	response := response{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode Cloudflare response: %w", err)
	}

	return &Credentials{
		URL:        response.URL(),
		Username:   response.ICEServers.Userid,
		Credential: response.ICEServers.Credential,
		Lifetime:   int(lifetime),
	}, nil
}
