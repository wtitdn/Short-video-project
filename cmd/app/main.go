package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/wtitdn/renew_video/internal/config"
	"github.com/wtitdn/renew_video/internal/controller/http"
	"github.com/wtitdn/renew_video/internal/db"
	"github.com/wtitdn/renew_video/internal/middleware/auth"
	llm2 "github.com/wtitdn/renew_video/internal/middleware/llm"
	smtp "github.com/wtitdn/renew_video/pkg/SMTP"
	storage "github.com/wtitdn/renew_video/pkg/minio"
	"github.com/wtitdn/renew_video/pkg/observability"
	"github.com/wtitdn/renew_video/pkg/rabbitmq"
	rediscache "github.com/wtitdn/renew_video/pkg/redis"
)

func main() {
	// Load .env if present; continue without it in deployed environments.
	if err := godotenv.Load(); err != nil {
		log.Println(".env not found; continuing")
	}

	// Load application config.
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "../config/config.yaml"
	}
	log.Printf("Loading config from %s", configPath)
	cfg, usedDefault, err := config.LoadLocalDev(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	if usedDefault {
		log.Printf("Config File %s not found, using default local config", configPath)
	} else {
		log.Printf("Config loaded from file: %s", configPath)
	}
	auth.SetJWTSecret(cfg.JWT.Secret)

	// Initialize database and run migrations.
	sqlDB, err := db.NewDB(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect database: %v", err)
	}
	if err := db.AutoMigrate(sqlDB, cfg.Admin); err != nil {
		log.Fatalf("Failed to auto migrate database: %v", err)
	}
	defer db.CloseDB(sqlDB)

	// Initialize Redis; disable cache if it is unavailable.
	cache, err := rediscache.NewFromEnv(&cfg.Redis)
	if err != nil {
		log.Printf("Redis config error (cache disabled): %v", err)
		cache = nil
	} else {
		pingCtx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		defer cancel()
		if err := cache.Ping(pingCtx); err != nil {
			log.Printf("Redis not available (cache disabled): %v", err)
			_ = cache.Close()
			cache = nil
		} else {
			defer cache.Close()
			log.Printf("Redis connected (cache enabled)")
		}
	}

	// Initialize Minio; disable object storage if it is unavailable.
	mio, err := storage.NewMinio(&cfg.Minio)
	if err != nil {
		log.Printf("minio config error (disabled): %v", err)
		mio = nil
	} else {
		defer mio.Close()
		log.Printf("Minio connected")
	}
	// Initialize LLMClient
	llm, err := llm2.NewClient(&cfg.ApiKeyConfig)
	if err != nil {
		log.Fatalf("Failed to create llm client: %v", err)
	} else {
		defer llm.Close()
		log.Printf("llm initialized")
	}
	// Initialize RabbitMQ; disable message queue if it is unavailable.
	rmq, err := rabbitmq.NewRabbitMQ(&cfg.RabbitMQ)
	if err != nil {
		log.Printf("RabbitMQ config error (disabled): %v", err)
		rmq = nil
	} else {
		defer rmq.Close()
		log.Printf("RabbitMQ connected")
	}

	// Initialize SMTP mail client.
	smt := smtp.NewSMTPClient(cfg.SMTP)
	if smt != nil {
		defer smt.Close()
		log.Printf("SMTP client initialized")
	}

	// Start pprof server when enabled.
	pprofServer, err := observability.NewPprofServer(
		"API",
		cfg.ObservabilityConfig.Pprof.Enabled,
		cfg.ObservabilityConfig.Pprof.ApiAddr,
	)
	if err != nil {
		log.Printf("Failed to start API pprof server: %v", err)
	}
	if pprofServer != nil {
		defer pprofServer.Close()
	}

	// Register routes and start HTTP server.
	r := http.SetRouter(sqlDB, cache, rmq, mio, smt, llm)
	log.Printf("Server is running on port %d", cfg.Server.Port)
	if err := r.Run(":" + strconv.Itoa(cfg.Server.Port)); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
