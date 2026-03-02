package config

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port           string
	MaxFileSize    int64
	MaxMessageSize int64

	//BenchID is the unique id of each bench in the park.
	//if not set via env var, a random buid (bench unique id) is generated at startup and it is being used to avoid re-broadcasting messages they first received from another bench and prevent infinite message loops
	BenchID string

	//NATSPeers is the bench friendlist of servers the bench I'm in, connects to.
	//Empty means standalone - no federation, all rooms stay local.
	//Set by NATS_PEERS env var as a comma-separated list
	NATSPeers []string
}

func Load() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	maxFileSize := parseEnvInt64("MAX_FILE_SIZE", 10485760)
	maxMessageSize := parseEnvInt64("MAX_MESSAGE_SIZE", 10485760)
	//I'm going to make the default maximum for both 10MB for consistency. It's not a photo album, remember that later.
	return Config{
		Port:           port,
		MaxFileSize:    maxFileSize,
		MaxMessageSize: maxMessageSize,
		BenchID:        loadBenchID(),
		NATSPeers:      loadNATSPeers(),
	}
}

// the loadBenchID reads the buid from env. If none is set, a random one is generated as an 8-byte hex string of 16 characters.
// this means every fresh deployment without explicit BENCH_ID gets a new identity.
// in this case it's fine, because buid is used for loop prevention within a single running session.
func loadBenchID() string {
	id := os.Getenv("BENCH_ID")
	if id != "" {
		return id
	}

	//random 8 bytes generated -> 16 hex characters
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		//not going to happen a lot but it's best to fall back to a fixed string rather than panic (DEFENSIVE FRICCIN PROGRAMMING B*TCH)
		return "bench-fallback"
	}
	return hex.EncodeToString(b)
}

// So, here "loadNATSPeers" reads NATS_PEERS from env and splits on each comma.
// it gives back an empty slice if the var is not set, that's for standalone.
// whitespace around each address is trimmed so operators don't have to be precise with spacing in their env config.
func loadNATSPeers() []string {
	raw := os.Getenv("NATS_PEERS")
	if raw == "" {
		return []string{}
	}

	parts := strings.Split(raw, ",")
	peers := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			peers = append(peers, p)
		}
	}
	return peers
}

// handy: reads a env variable as int64, if it's faulty or if it's not there, it falls back to default values
func parseEnvInt64(key string, defaultVal int64) int64 {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}

	parsed, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return defaultVal
	}

	return parsed
}
