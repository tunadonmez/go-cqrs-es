package config

import "os"

type Config struct {
	Port           string
	MySQLDSN       string
	KafkaBootstrap string
	KafkaGroupID   string
}

func Load() Config {
	return Config{
		Port:           getEnv("PORT", "5001"),
		MySQLDSN:       getEnv("MYSQL_DSN", "root:techbankRootPsw@tcp(localhost:3306)/bankAccount?charset=utf8mb4&parseTime=True&loc=Local"),
		KafkaBootstrap: getEnv("KAFKA_BOOTSTRAP_SERVERS", "localhost:9092"),
		KafkaGroupID:   getEnv("KAFKA_GROUP_ID", "bankaccConsumer"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
