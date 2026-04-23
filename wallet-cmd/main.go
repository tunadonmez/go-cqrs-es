package main

import (
	"context"
	"errors"
	"log"
	"reflect"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tunadonmez/go-cqrs-es/wallet-cmd/api/commands"
	"github.com/tunadonmez/go-cqrs-es/wallet-cmd/api/controllers"
	"github.com/tunadonmez/go-cqrs-es/wallet-cmd/application"
	"github.com/tunadonmez/go-cqrs-es/wallet-cmd/config"
	"github.com/tunadonmez/go-cqrs-es/wallet-cmd/infrastructure"
	_ "github.com/tunadonmez/go-cqrs-es/wallet-common/events"

	corecommands "github.com/tunadonmez/go-cqrs-es/cqrs-core/commands"
	coreinfra "github.com/tunadonmez/go-cqrs-es/cqrs-core/infrastructure"

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

	db := client.Database("walletLedger")

	// Wire up write-side components.
	repo := infrastructure.NewEventStoreRepository(db)
	producer := infrastructure.NewWalletEventProducer(cfg.KafkaBootstrap)
	defer producer.Close()

	// Start the transactional outbox publisher. SaveEvents only writes events
	// (with a PENDING outbox marker) to MongoDB; this background worker owns
	// the Kafka delivery step.
	outbox := infrastructure.NewOutboxPublisher(repo, producer, infrastructure.EventsTopic)
	outboxCtx, cancelOutbox := context.WithCancel(context.Background())
	defer cancelOutbox()
	outbox.Start(outboxCtx)

	var eventStore coreinfra.EventStore = infrastructure.NewWalletEventStore(repo)
	eventSourcingHandler := infrastructure.NewWalletEventSourcingHandler(eventStore)
	cmdHandler := application.NewCommandHandler(eventSourcingHandler)
	cmdDispatcher := coreinfra.NewCommandDispatcher()
	cmdDispatcher.Use(func(cmd interface{}) error {
		identified, ok := cmd.(corecommands.IdentifiedCommand)
		if !ok {
			return nil
		}
		if identified.GetID() == "" {
			return errors.New("command ID is required")
		}
		return nil
	})
	cmdDispatcher.RegisterHandler(reflect.TypeOf(&commands.CreateWalletCommand{}), func(cmd interface{}) error {
		return cmdHandler.HandleCreateWallet(cmd.(*commands.CreateWalletCommand))
	})
	cmdDispatcher.RegisterHandler(reflect.TypeOf(&commands.CreditWalletCommand{}), func(cmd interface{}) error {
		return cmdHandler.HandleCreditWallet(cmd.(*commands.CreditWalletCommand))
	})
	cmdDispatcher.RegisterHandler(reflect.TypeOf(&commands.DebitWalletCommand{}), func(cmd interface{}) error {
		return cmdHandler.HandleDebitWallet(cmd.(*commands.DebitWalletCommand))
	})
	cmdDispatcher.RegisterHandler(reflect.TypeOf(&commands.TransferFundsCommand{}), func(cmd interface{}) error {
		return cmdHandler.HandleTransferFunds(cmd.(*commands.TransferFundsCommand))
	})

	// Set up HTTP routes
	r := gin.Default()
	v1 := r.Group("/api/v1")
	{
		v1.POST("/wallets", controllers.CreateWalletHandler(cmdDispatcher))
		v1.PUT("/wallets/:id/credit", controllers.CreditWalletHandler(cmdDispatcher))
		v1.PUT("/wallets/:id/debit", controllers.DebitWalletHandler(cmdDispatcher))
		v1.POST("/wallets/:id/transfer", controllers.TransferFundsHandler(cmdDispatcher))
	}

	log.Printf("Command service starting on port %s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
