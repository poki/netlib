package metrics

type Event struct {
	Time    int64  `json:"time"`
	Client  string `json:"client"`
	Game    string `json:"game"`
	Version string `json:"version"`

	Category string `json:"category"`
	Action   string `json:"action"`

	Peer  string  `json:"peer"`
	Lobby string  `json:"lobby,omitempty"`
	Lat   float64 `json:"lat,omitempty"`
	Lon   float64 `json:"lon,omitempty"`

	Data map[string]string `json:"data,omitempty"`
}
