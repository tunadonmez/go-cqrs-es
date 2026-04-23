package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"reflect"

	"github.com/gin-gonic/gin"
	"github.com/tunadonmez/go-cqrs-es/wallet-common/observability"
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
	// Initialize slog
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// CLI flags
	var (
		replayFlag    = flag.Bool("replay", false, "rebuild the read model from the event store, then exit")
		aggregateFlag = flag.String("aggregate", "", "when used with --replay, restrict the rebuild to a single aggregate id")
	)
	flag.Parse()

	cfg := config.Load()

	// Connect to PostgreSQL via GORM
	db, err := gorm.Open(postgres.Open(cfg.PostgresDSN), &gorm.Config{})
	if err != nil {
		slog.Error("PostgreSQL connect error", "error", err)
		os.Exit(1)
	}
	slog.Info("Connected to PostgreSQL")

	// Auto-migrate the read model and the processed-events inbox.
	if err := db.AutoMigrate(&domain.Wallet{}, &domain.Transaction{}, &infrastructure.ProcessedEvent{}); err != nil {
		slog.Error("AutoMigrate error", "error", err)
		os.Exit(1)
	}

	// Wire up dependencies shared by both modes.
	repo := infrastructure.NewWalletRepository(db)
	eventHandler := infrastructure.NewWalletEventHandler(repo)

	if *replayFlag {
		runReplay(cfg, db, eventHandler, *aggregateFlag)
		return
	}

	// --- Normal service mode ---
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

	// Health check endpoints
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "UP"})
	})
	r.GET("/ready", func(c *gin.Context) {
		// Basic check: can we ping Postgres?
		sqlDB, err := db.DB()
		if err != nil {
			c.JSON(503, gin.H{"status": "DOWN", "reason": "database error"})
			return
		}
		if err := sqlDB.Ping(); err != nil {
			c.JSON(503, gin.H{"status": "DOWN", "reason": "database unavailable"})
			return
		}
		c.JSON(200, gin.H{"status": "READY"})
	})

	r.GET("/metrics", func(c *gin.Context) {
		c.JSON(200, observability.DefaultMetrics.Snapshot())
	})

	v1 := r.Group("/api/v1")
	controllers.RegisterRoutes(v1, queryDispatcher)

	slog.Info("Query service starting", "port", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		slog.Error("Server error", "error", err)
		os.Exit(1)
	}
}

// runReplay performs a one-shot rebuild of the read model from the event
// store. It never touches Kafka and reuses the live WalletEventHandler so
// there is only one projection path in the codebase.
func runReplay(cfg config.Config, db *gorm.DB, handler *infrastructure.WalletEventHandler, aggregateID string) {
	ctx := context.Background()

	reader, err := infrastructure.NewEventSourceReader(ctx, cfg.MongoURI, cfg.MongoDB)
	if err != nil {
		slog.Error("replay: could not connect to event store", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := reader.Close(context.Background()); err != nil {
			slog.Error("replay: mongo disconnect error", "error", err)
		}
	}()

	replayer := infrastructure.NewReplayer(reader, handler, db)
	if err := replayer.Run(ctx, infrastructure.ReplayOptions{AggregateID: aggregateID}); err != nil {
		slog.Error("replay failed", "error", err)
		os.Exit(1)
	}
}
