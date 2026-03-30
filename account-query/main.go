package main

import (
	"context"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/techbank/account-query/api/controllers"
	"github.com/techbank/account-query/config"
	"github.com/techbank/account-query/domain"
	"github.com/techbank/account-query/infrastructure"

	// Trigger event registration via init()
	_ "github.com/techbank/account-common/events"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	cfg := config.Load()

	// Connect to MySQL via GORM
	db, err := gorm.Open(mysql.Open(cfg.MySQLDSN), &gorm.Config{})
	if err != nil {
		log.Fatalf("MySQL connect error: %v", err)
	}
	log.Println("Connected to MySQL")

	// Auto-migrate the read model
	if err := db.AutoMigrate(&domain.Account{}); err != nil {
		log.Fatalf("AutoMigrate error: %v", err)
	}

	// Wire up dependencies
	repo := infrastructure.NewAccountRepository(db)
	eventHandler := infrastructure.NewAccountEventHandler(repo)
	queryHandler := infrastructure.NewAccountQueryHandler(repo)

	// Start Kafka consumers in background
	consumer := infrastructure.NewAccountEventConsumer(cfg.KafkaBootstrap, cfg.KafkaGroupID, eventHandler)
	consumer.Start(context.Background())

	// Set up HTTP routes
	r := gin.Default()
	v1 := r.Group("/api/v1")
	controllers.RegisterRoutes(v1, queryHandler)

	log.Printf("Query service starting on port %s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
