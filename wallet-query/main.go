package main

import (
	"context"
	"log"
	"reflect"

	"github.com/gin-gonic/gin"
	"github.com/tunadonmez/go-cqrs-es/wallet-query/api/controllers"
	"github.com/tunadonmez/go-cqrs-es/wallet-query/api/queries"
	"github.com/tunadonmez/go-cqrs-es/wallet-query/config"
	"github.com/tunadonmez/go-cqrs-es/wallet-query/domain"
	"github.com/tunadonmez/go-cqrs-es/wallet-query/infrastructure"

	// Trigger event registration via init()
	coredomain "github.com/tunadonmez/go-cqrs-es/cqrs-core/domain"
	coreinfra "github.com/tunadonmez/go-cqrs-es/cqrs-core/infrastructure"
	corequeries "github.com/tunadonmez/go-cqrs-es/cqrs-core/queries"
	_ "github.com/tunadonmez/go-cqrs-es/wallet-common/events"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	cfg := config.Load()

	// Connect to PostgreSQL via GORM
	db, err := gorm.Open(postgres.Open(cfg.PostgresDSN), &gorm.Config{})
	if err != nil {
		log.Fatalf("PostgreSQL connect error: %v", err)
	}
	log.Println("Connected to PostgreSQL")

	// Auto-migrate the read model
	if err := db.AutoMigrate(&domain.Wallet{}, &domain.Transaction{}); err != nil {
		log.Fatalf("AutoMigrate error: %v", err)
	}

	// Wire up dependencies
	repo := infrastructure.NewWalletRepository(db)
	eventHandler := infrastructure.NewWalletEventHandler(repo)
	queryHandler := infrastructure.NewWalletQueryHandler(repo)
	queryDispatcher := coreinfra.NewQueryDispatcher()
	queryDispatcher.RegisterHandler(reflect.TypeOf(queries.FindAllWalletsQuery{}), func(q corequeries.BaseQuery) ([]coredomain.BaseEntity, error) {
		return queryHandler.HandleFindAll(q.(queries.FindAllWalletsQuery))
	})
	queryDispatcher.RegisterHandler(reflect.TypeOf(queries.FindWalletByIDQuery{}), func(q corequeries.BaseQuery) ([]coredomain.BaseEntity, error) {
		return queryHandler.HandleFindByID(q.(queries.FindWalletByIDQuery))
	})
	queryDispatcher.RegisterHandler(reflect.TypeOf(queries.FindWalletTransactionsQuery{}), func(q corequeries.BaseQuery) ([]coredomain.BaseEntity, error) {
		return queryHandler.HandleFindTransactions(q.(queries.FindWalletTransactionsQuery))
	})

	// Start Kafka consumers in background
	consumer := infrastructure.NewWalletEventConsumer(cfg.KafkaBootstrap, cfg.KafkaGroupID, eventHandler)
	consumer.Start(context.Background())

	// Set up HTTP routes
	r := gin.Default()
	v1 := r.Group("/api/v1")
	controllers.RegisterRoutes(v1, queryDispatcher)

	log.Printf("Query service starting on port %s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
