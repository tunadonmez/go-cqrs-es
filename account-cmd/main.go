package main

import (
	"context"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/techbank/account-cmd/api/controllers"
	"github.com/techbank/account-cmd/application"
	"github.com/techbank/account-cmd/config"
	"github.com/techbank/account-cmd/infrastructure"

	// Trigger event registration via init()
	_ "github.com/techbank/account-common/events"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func main() {
	cfg := config.Load()

	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		log.Fatalf("MongoDB connect error: %v", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatalf("MongoDB ping error: %v", err)
	}
	log.Println("Connected to MongoDB")

	db := client.Database("bankAccount")

	// Wire up dependencies
	repo := infrastructure.NewEventStoreRepository(db)
	producer := infrastructure.NewAccountEventProducer(cfg.KafkaBootstrap)
	eventStore := infrastructure.NewAccountEventStore(producer, repo)
	eventSourcingHandler := infrastructure.NewAccountEventSourcingHandler(eventStore)
	cmdHandler := application.NewCommandHandler(eventSourcingHandler)

	// Set up HTTP routes
	r := gin.Default()
	v1 := r.Group("/api/v1")
	{
		v1.POST("/openBankAccount", controllers.OpenAccountHandler(cmdHandler))
		v1.PUT("/depositFunds/:id", controllers.DepositFundsHandler(cmdHandler))
		v1.PUT("/withdrawFunds/:id", controllers.WithdrawFundsHandler(cmdHandler))
		v1.DELETE("/closeBankAccount/:id", controllers.CloseAccountHandler(cmdHandler))
	}

	log.Printf("Command service starting on port %s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
