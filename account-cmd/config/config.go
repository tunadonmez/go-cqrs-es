package config

import "os"

type Config struct {
	Port           string
	MongoURI       string
	KafkaBootstrap string
}

func Load() Config {
	return Config{
		Port:           getEnv("PORT", "5000"),
		MongoURI:       getEnv("SPRING_DATA_MONGODB_URI", "mongodb://root:root@localhost:27017/bankAccount?authSource=admin"),
		KafkaBootstrap: getEnv("SPRING_KAFKA_BOOTSTRAP_SERVERS", "localhost:9092"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
