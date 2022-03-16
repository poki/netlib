package signaling

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

var ErrMissingEnvironmentVariables = errors.New("missing environment variables")

func GetCredentials(ctx context.Context) (*CredentialsPacket, error) {
	cloudflareZone := os.Getenv("CLOUDFLARE_ZONE")
	cloudflareAppID := os.Getenv("CLOUDFLARE_APP_ID")
	cloudflareAuthUser := os.Getenv("CLOUDFLARE_AUTH_USER")
	cloudflareAuthKey := os.Getenv("CLOUDFLARE_AUTH_KEY")
	if cloudflareZone == "" || cloudflareAppID == "" || cloudflareAuthUser == "" || cloudflareAuthKey == "" {
		return nil, ErrMissingEnvironmentVariables
	}

	start := time.Now()
	defer func() {
		fmt.Println("took", time.Since(start))
	}()
	url := "https://api.cloudflare.com/client/v4/zones/" + cloudflareZone + "/webrtc-turn/credential/" + cloudflareAppID
	body := strings.NewReader(`{"lifetime": 3600}`)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Auth-Email", cloudflareAuthUser)
	req.Header.Set("X-Auth-Key", cloudflareAuthKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected error from Cloudflare: %s", resp.Status)
	}

	response := CloudflareResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode Cloudflare response: %w", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("cloudflare error: %v", response.Errors)
	}

	return &CredentialsPacket{
		Type: "credentials",

		URL:        response.URL(),
		Username:   response.Result.Userid,
		Credential: response.Result.Credential,
		Lifetime:   response.Result.Lifetime,
	}, nil
}

type CloudflareResponse struct {
	Result struct {
		Protocol string `json:"protocol"`
		DNS      struct {
			Name string `json:"name"`
		} `json:"dns"`
		Lifetime   int    `json:"lifetime"`
		Userid     string `json:"userid"`
		Credential string `json:"credential"`
	} `json:"result"`
	Success bool          `json:"success"`
	Errors  []interface{} `json:"errors"`
}

// URL returns in the following format:
// turn:webrtc-turn.example.com:50000?transport=udp
func (r CloudflareResponse) URL() string {
	protocol := r.Result.Protocol
	parts := strings.Split(protocol, "/")
	if len(parts) != 2 {
		parts = []string{"udp", "50000"}
	}
	return "turn:" + r.Result.DNS.Name + ":" + parts[1] + "?transport=" + parts[0]
}
