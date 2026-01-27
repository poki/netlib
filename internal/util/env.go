package util

import "os"

func Getenv(key, def string) string {
	if val, found := os.LookupEnv(key); found {
		return val
	}
	return def
}

// TopologyMode returns the configured network topology mode.
// "mesh" (default): all peers connect to all peers
// "star": peers only connect to the leader, who relays messages
func TopologyMode() string {
	return Getenv("TOPOLOGY_MODE", "mesh")
}

// IsStarTopology returns true if the server is configured for star topology.
func IsStarTopology() bool {
	return TopologyMode() == "star"
}
