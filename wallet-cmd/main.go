package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"reflect"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tunadonmez/go-cqrs-es/wallet-cmd/api/commands"
	"github.com/tunadonmez/go-cqrs-es/wallet-cmd/api/controllers"
	"github.com/tunadonmez/go-cqrs-es/wallet-cmd/application"
	"github.com/tunadonmez/go-cqrs-es/wallet-cmd/config"
	"github.com/tunadonmez/go-cqrs-es/wallet-cmd/infrastructure"
	_ "github.com/tunadonmez/go-cqrs-es/wallet-common/events"
	"github.com/tunadonmez/go-cqrs-es/wallet-common/observability"

	corecommands "github.com/tunadonmez/go-cqrs-es/cqrs-core/commands"
	coreinfra "github.com/tunadonmez/go-cqrs-es/cqrs-core/infrastructure"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func main() {
	// Initialize slog
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", "wallet-cmd")
	slog.SetDefault(logger)

	cfg := config.Load()

	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		slog.Error("MongoDB connect error", "error", err)
		os.Exit(1)
	}
	if err := client.Ping(ctx, nil); err != nil {
		slog.Error("MongoDB ping error", "error", err)
		os.Exit(1)
	}
	slog.Info("Connected to MongoDB")

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
	// Logging & Metrics middleware
	cmdDispatcher.Use(func(cmd interface{}) error {
		commandType := reflect.TypeOf(cmd)
		if commandType.Kind() == reflect.Ptr {
			commandType = commandType.Elem()
		}
		attrs := []any{"commandType", commandType.Name()}
		if identified, ok := cmd.(corecommands.IdentifiedCommand); ok {
			attrs = append(attrs, "commandId", identified.GetID())
		}
		observability.DefaultMetrics.CommandsReceived.Add(1)
		slog.Info("Command received", attrs...)
		return nil
	})
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

	// Health check endpoints
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "UP"})
	})
	r.GET("/ready", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if err := client.Ping(ctx, nil); err != nil {
			c.JSON(503, gin.H{"status": "DOWN", "reason": "mongodb unavailable"})
			return
		}
		c.JSON(200, gin.H{"status": "READY"})
	})

	r.GET("/metrics", func(c *gin.Context) {
		c.JSON(200, observability.DefaultMetrics.Snapshot())
	})

	v1 := r.Group("/api/v1")
	{
		v1.POST("/wallets", controllers.CreateWalletHandler(cmdDispatcher))
		v1.PUT("/wallets/:id/credit", controllers.CreditWalletHandler(cmdDispatcher))
		v1.PUT("/wallets/:id/debit", controllers.DebitWalletHandler(cmdDispatcher))
		v1.POST("/wallets/:id/transfer", controllers.TransferFundsHandler(cmdDispatcher))
	}

	slog.Info("Command service starting", "port", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		slog.Error("Server error", "error", err)
		os.Exit(1)
	}
}
