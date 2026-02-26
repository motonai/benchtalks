package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port           string
	MaxFileSize    int64
	MaxMessageSize int64
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
	}
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
