package config

import "os"

type Config struct {
	Port     string
	MongoURI string
	MongoDB  string
}

func Load() *Config {
	return &Config{
		Port:     getEnv("SERVER_PORT", "8080"),
		MongoURI: getEnv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDB:  getEnv("MONGO_DB", "kfc_training"),
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
