package main

import (
	"fmt"
	"kucukaslan/clickhouse/api"
	"kucukaslan/clickhouse/services"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"kucukaslan/clickhouse/buildinfo"
	"kucukaslan/clickhouse/config"
	"kucukaslan/clickhouse/database"

	"github.com/gofiber/fiber/v2/middleware/recover"

	_ "kucukaslan/clickhouse/docs" // Import generated docs

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"
)

// @title ClickHouse Event Tracking API
// @version 1.0
// @description Event tracking and analytics service using ClickHouse and Redis
// @BasePath /
// @schemes http

const idleTimeout = 5 * time.Second

func main() {
	// Set application start time for accurate uptime tracking
	buildinfo.SetStartTime(time.Now())

	// Log build information
	info := buildinfo.GetInfo()
	log.Printf("Starting application\nVersion: %s, Commit: %s, BuildDate: %s, GoVersion: %s, Hostname: %s",
		info.Version, info.Commit, info.BuildDate, info.GoVersion, info.Hostname)
	// Load configuration
	cfg := config.Load()

	// Initialize ClickHouse connection
	if err := database.InitClickHouse(&cfg.ClickHouse); err != nil {
		log.Fatalf("Failed to initialize ClickHouse: %v", err)
	}

	// Initialize Redis connection
	if err := database.InitRedis(&cfg.Redis); err != nil {
		// TODO: we are (will be) using redis for idempotency/deduplication,
		// so it is not necessary to fail the application if redis is not available.
		// app can be slowed down by this, but it is not critical.
		// obviously we should have a fallback mechanism for the places where we use redis.
		// Possible in memory cache fallback.
		log.Fatalf("Failed to initialize Redis: %v", err)
	}

	eventService, err := services.NewEventService(database.GetClickHouseDB(), &cfg.ClickHouse, database.GetRedisClient(cfg.ClickHouse.RedisCacheDurationMS))
	if err != nil {
		log.Fatalf("Failed to initialize EventService: %v", err)
	}

	httpHandler := api.NewEventHandler(eventService)

	app := fiber.New(fiber.Config{
		IdleTimeout: idleTimeout,
	})

	app.Use(recover.New())

	// redirect to swagger docs
	app.Get("/", func(c *fiber.Ctx) error {
		return c.Redirect("/swagger/", fiber.StatusMovedPermanently)
	})

	// Health check endpoint
	app.Get("/health", api.HealthCheck)

	// Swagger documentation
	app.Get("/swagger/*", swagger.HandlerDefault)

	// Event endpoints
	app.Post("/events", httpHandler.PostEvent)
	app.Post("/events/bulk", httpHandler.PostEventsBulk)
	app.Get("/metrics", httpHandler.GetMetrics)

	// Listen from a different goroutine
	go func() {
		if err := app.Listen(":" + cfg.Port); err != nil {
			log.Panic(err)
		}
	}()

	c := make(chan os.Signal, 1)                    // Create channel to signify a signal being sent
	signal.Notify(c, os.Interrupt, syscall.SIGTERM) // When an interrupt or termination signal is sent, notify the channel

	_ = <-c // This blocks the main thread until an interrupt is received
	fmt.Println("Gracefully shutting down...")
	_ = app.Shutdown()

	fmt.Println("Running cleanup tasks...")

	// Shutdown event service batcher (flushes remaining events)
	if err := services.ShutdownEventService(eventService); err != nil {
		log.Printf("Error shutting down event service batcher: %v", err)
	}

	// Close database connections
	if err := database.CloseClickHouse(); err != nil {
		log.Printf("Error closing ClickHouse: %v", err)
	}

	if err := database.CloseRedis(); err != nil {
		log.Printf("Error closing Redis: %v", err)
	}

	fmt.Println("Fiber was successful shutdown.")
}
