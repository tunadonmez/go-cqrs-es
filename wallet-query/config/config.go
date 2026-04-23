package config

import "os"

type Config struct {
	Port           string
	PostgresDSN    string
	KafkaBootstrap string
	KafkaGroupID   string
}

func Load() Config {
	return Config{
		Port:           getEnv("PORT", "5001"),
		PostgresDSN:    getEnv("POSTGRES_DSN", "host=localhost user=postgres password=postgres dbname=walletLedger port=5432 sslmode=disable TimeZone=UTC"),
		KafkaBootstrap: getEnv("KAFKA_BOOTSTRAP_SERVERS", "localhost:9092"),
		KafkaGroupID:   getEnv("KAFKA_GROUP_ID", "walletConsumer"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
