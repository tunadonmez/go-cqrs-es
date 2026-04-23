package main

import (
	"context"
	"errors"
	"log"
	"reflect"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tunadonmez/go-cqrs-es/account-cmd/api/commands"
	"github.com/tunadonmez/go-cqrs-es/account-cmd/api/controllers"
	"github.com/tunadonmez/go-cqrs-es/account-cmd/application"
	"github.com/tunadonmez/go-cqrs-es/account-cmd/config"
	"github.com/tunadonmez/go-cqrs-es/account-cmd/infrastructure"

	// Trigger event registration via init()
	corecommands "github.com/tunadonmez/go-cqrs-es/cqrs-core/commands"
	coreinfra "github.com/tunadonmez/go-cqrs-es/cqrs-core/infrastructure"
	coreproducers "github.com/tunadonmez/go-cqrs-es/cqrs-core/producers"

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
	var producer coreproducers.EventProducer = infrastructure.NewAccountEventProducer(cfg.KafkaBootstrap)
	var eventStore coreinfra.EventStore = infrastructure.NewAccountEventStore(producer, repo)
	eventSourcingHandler := infrastructure.NewAccountEventSourcingHandler(eventStore)
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
	cmdDispatcher.RegisterHandler(reflect.TypeOf(&commands.OpenAccountCommand{}), func(cmd interface{}) error {
		return cmdHandler.HandleOpenAccount(cmd.(*commands.OpenAccountCommand))
	})
	cmdDispatcher.RegisterHandler(reflect.TypeOf(&commands.DepositFundsCommand{}), func(cmd interface{}) error {
		return cmdHandler.HandleDepositFunds(cmd.(*commands.DepositFundsCommand))
	})
	cmdDispatcher.RegisterHandler(reflect.TypeOf(&commands.WithdrawFundsCommand{}), func(cmd interface{}) error {
		return cmdHandler.HandleWithdrawFunds(cmd.(*commands.WithdrawFundsCommand))
	})
	cmdDispatcher.RegisterHandler(reflect.TypeOf(&commands.CloseAccountCommand{}), func(cmd interface{}) error {
		return cmdHandler.HandleCloseAccount(cmd.(*commands.CloseAccountCommand))
	})

	// Set up HTTP routes
	r := gin.Default()
	v1 := r.Group("/api/v1")
	{
		v1.POST("/openBankAccount", controllers.OpenAccountHandler(cmdDispatcher))
		v1.PUT("/depositFunds/:id", controllers.DepositFundsHandler(cmdDispatcher))
		v1.PUT("/withdrawFunds/:id", controllers.WithdrawFundsHandler(cmdDispatcher))
		v1.DELETE("/closeBankAccount/:id", controllers.CloseAccountHandler(cmdDispatcher))
	}

	log.Printf("Command service starting on port %s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
