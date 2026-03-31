package main

import (
	"context"
	"log"
	"reflect"

	"github.com/gin-gonic/gin"
	"github.com/tunadonmez/go-cqrs-es/account-query/api/controllers"
	"github.com/tunadonmez/go-cqrs-es/account-query/api/queries"
	"github.com/tunadonmez/go-cqrs-es/account-query/config"
	"github.com/tunadonmez/go-cqrs-es/account-query/domain"
	"github.com/tunadonmez/go-cqrs-es/account-query/infrastructure"

	// Trigger event registration via init()
	_ "github.com/tunadonmez/go-cqrs-es/account-common/events"
	coredomain "github.com/tunadonmez/go-cqrs-es/cqrs-core/domain"
	coreinfra "github.com/tunadonmez/go-cqrs-es/cqrs-core/infrastructure"
	corequeries "github.com/tunadonmez/go-cqrs-es/cqrs-core/queries"

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
	queryDispatcher := coreinfra.NewQueryDispatcher()
	queryDispatcher.RegisterHandler(reflect.TypeOf(queries.FindAllAccountsQuery{}), func(q corequeries.BaseQuery) ([]coredomain.BaseEntity, error) {
		return queryHandler.HandleFindAll(q.(queries.FindAllAccountsQuery))
	})
	queryDispatcher.RegisterHandler(reflect.TypeOf(queries.FindAccountByIdQuery{}), func(q corequeries.BaseQuery) ([]coredomain.BaseEntity, error) {
		return queryHandler.HandleFindByID(q.(queries.FindAccountByIdQuery))
	})

	// Start Kafka consumers in background
	consumer := infrastructure.NewAccountEventConsumer(cfg.KafkaBootstrap, cfg.KafkaGroupID, eventHandler)
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
