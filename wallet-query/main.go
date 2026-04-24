package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strings"
	"time"

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
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", "wallet-query")
	slog.SetDefault(logger)

	// CLI flags
	var (
		replayFlag              = flag.Bool("replay", false, "rebuild the read model from the event store, then exit")
		aggregateFlag           = flag.String("aggregate", "", "when used with --replay, restrict the rebuild to a single aggregate id")
		reprocessDeadLetterFlag = flag.String("reprocess-dead-letter", "", "reprocess a single dead-letter row by dead_letter_key, then exit")
	)
	flag.Parse()

	if *replayFlag && *reprocessDeadLetterFlag != "" {
		slog.Error("flags --replay and --reprocess-dead-letter are mutually exclusive")
		os.Exit(1)
	}

	cfg := config.Load()

	// Connect to PostgreSQL via GORM
	db, err := gorm.Open(postgres.Open(cfg.PostgresDSN), &gorm.Config{})
	if err != nil {
		slog.Error("PostgreSQL connect error", "error", err)
		os.Exit(1)
	}
	slog.Info("Connected to PostgreSQL")

	// Auto-migrate the read model, the processed-events inbox, and the
	// operational dead-letter table used by the live Kafka consumer.
	if err := db.AutoMigrate(
		&domain.Wallet{},
		&domain.Transaction{},
		&domain.LedgerEntry{},
		&infrastructure.ProcessedEvent{},
		&infrastructure.DeadLetterEvent{},
		&infrastructure.ProjectionVersion{},
	); err != nil {
		slog.Error("AutoMigrate error", "error", err)
		os.Exit(1)
	}

	// Wire up dependencies shared by both modes.
	repo := infrastructure.NewWalletRepository(db)
	eventHandler := infrastructure.NewWalletEventHandler(repo)
	deadLetters := infrastructure.NewDeadLetterRepository(db)
	deadLetterReprocessor := infrastructure.NewDeadLetterReprocessor(deadLetters, eventHandler)
	projectionVersions := infrastructure.NewProjectionVersionRepository(db)
	projectionVersionManager := infrastructure.NewProjectionVersionManager(
		projectionVersions,
		infrastructure.DefinedProjectionVersions,
	)

	if err := projectionVersionManager.CheckStartup(); err != nil {
		slog.Error("Projection version startup check failed", "error", err)
		os.Exit(1)
	}

	if *replayFlag {
		runReplay(cfg, db, eventHandler, projectionVersionManager, *aggregateFlag)
		return
	}
	if *reprocessDeadLetterFlag != "" {
		runDeadLetterReprocess(deadLetterReprocessor, *reprocessDeadLetterFlag)
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
	queryDispatcher.RegisterHandler(reflect.TypeOf(queries.FindLedgerEntriesQuery{}), func(q corequeries.BaseQuery) ([]coredomain.BaseEntity, error) {
		return queryHandler.HandleFindLedgerEntries(q.(queries.FindLedgerEntriesQuery))
	})

	// Start Kafka consumers in background
	consumer := infrastructure.NewWalletEventConsumer(cfg.KafkaBootstrap, cfg.KafkaGroupID, eventHandler, deadLetters)
	consumer.Start(context.Background())

	// Set up HTTP routes
	r := gin.Default()
	r.Use(corsMiddleware())

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
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if err := sqlDB.PingContext(ctx); err != nil {
			c.JSON(503, gin.H{"status": "DOWN", "reason": "database unavailable"})
			return
		}
		c.JSON(200, gin.H{"status": "READY"})
	})

	r.GET("/metrics", func(c *gin.Context) {
		c.JSON(200, observability.DefaultMetrics.Snapshot())
	})

	v1 := r.Group("/api/v1")
	controllers.RegisterRoutes(v1, queryDispatcher, deadLetters, deadLetterReprocessor)

	slog.Info("Query service starting", "port", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		slog.Error("Server error", "error", err)
		os.Exit(1)
	}
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if isAllowedOrigin(origin) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func isAllowedOrigin(origin string) bool {
	if origin == "" {
		return false
	}

	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}

	host := parsed.Hostname()
	return host == "localhost" || host == "127.0.0.1" || strings.EqualFold(host, "host.docker.internal")
}

// runReplay performs a one-shot rebuild of the read model from the event
// store. It never touches Kafka and reuses the live WalletEventHandler so
// there is only one projection path in the codebase.
func runReplay(
	cfg config.Config,
	db *gorm.DB,
	handler *infrastructure.WalletEventHandler,
	projectionVersionManager *infrastructure.ProjectionVersionManager,
	aggregateID string,
) {
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

	replayer := infrastructure.NewReplayer(reader, handler, projectionVersionManager, db)
	if err := replayer.Run(ctx, infrastructure.ReplayOptions{AggregateID: aggregateID}); err != nil {
		slog.Error("replay failed", "error", err)
		os.Exit(1)
	}
}

func runDeadLetterReprocess(reprocessor *infrastructure.DeadLetterReprocessor, deadLetterKey string) {
	ctx := context.Background()
	if err := reprocessor.Reprocess(ctx, deadLetterKey); err != nil {
		slog.Error("dead-letter reprocess failed", "deadLetterKey", deadLetterKey, "error", err)
		os.Exit(1)
	}
}
