package cloudflare

import "strings"

type Credentials struct {
	URL        string `json:"url"`
	Username   string `json:"username"`
	Credential string `json:"credential"`
	Lifetime   int    `json:"lifetime"`
}

type response struct {
	ICEServers struct {
		URLs       []string `json:"urls"`
		Userid     string   `json:"username"`
		Credential string   `json:"credential"`
	} `json:"iceServers"`
}

// URL returns in the following format:
// turn:webrtc-turn.example.com:50000?transport=udp
func (r response) URL() string {
	for _, url := range r.ICEServers.URLs {
		if strings.HasPrefix(url, "turn:") && strings.Contains(url, "?transport=udp") {
			return url
		}
	}

	return ""
}
