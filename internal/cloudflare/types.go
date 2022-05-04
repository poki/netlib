package cloudflare

import "strings"

type Credentials struct {
	URL        string `json:"url"`
	Username   string `json:"username"`
	Credential string `json:"credential"`
	Lifetime   int    `json:"lifetime"`
}

type response struct {
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
func (r response) URL() string {
	protocol := r.Result.Protocol
	parts := strings.Split(protocol, "/")
	if len(parts) != 2 {
		parts = []string{"udp", "50000"}
	}
	return "turn:" + r.Result.DNS.Name + ":" + parts[1] + "?transport=" + parts[0]
}
