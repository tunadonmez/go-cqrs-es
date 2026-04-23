package config

import "os"

type Config struct {
	Port           string
	PostgresDSN    string
	KafkaBootstrap string
	KafkaGroupID   string

	// MongoURI is the event store used by the replay CLI.
	// The live query service does not need it — only the read-model
	// rebuild path connects directly to the source of truth.
	MongoURI string
	MongoDB  string
}

func Load() Config {
	return Config{
		Port:           getEnv("PORT", "5001"),
		PostgresDSN:    getEnv("POSTGRES_DSN", "host=localhost user=postgres password=postgres dbname=walletLedger port=5432 sslmode=disable TimeZone=UTC"),
		KafkaBootstrap: getEnv("KAFKA_BOOTSTRAP_SERVERS", "localhost:9092"),
		KafkaGroupID:   getEnv("KAFKA_GROUP_ID", "walletConsumer"),
		MongoURI:       getEnv("MONGODB_URI", "mongodb://root:root@localhost:27017/walletLedger?authSource=admin"),
		MongoDB:        getEnv("MONGODB_DATABASE", "walletLedger"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
