package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aminofox/zenc-server/internal/api/httpapi"
	"github.com/aminofox/zenc-server/internal/api/websocket"
	"github.com/aminofox/zenc-server/internal/room"
	"github.com/aminofox/zenc-server/internal/store"
	"github.com/aminofox/zenc-server/pkg/config"
	"github.com/aminofox/zenc-server/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// Load configuration
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger
	lgr, err := logger.New(cfg.Logging.Level, cfg.Logging.Format)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer lgr.Sync()

	lgr.Info("Starting Zenc Server",
		"version", "0.1.0",
		"server_name", cfg.Server.Name,
	)

	// Initialize Redis store
	redisStore, err := store.NewRedisStore(cfg.Redis)
	if err != nil {
		lgr.Fatal("Failed to initialize Redis store", "error", err)
	}
	defer redisStore.Close()

	lgr.Info("Redis store initialized",
		"mode", cfg.Redis.Mode,
		"addresses", cfg.Redis.Addresses,
	)

	// Initialize Room Manager
	roomManager := room.NewManager(redisStore, lgr)

	// Configure WebSocket origin checking
	originChecker := websocket.MakeOriginChecker(cfg.WebSocket.AllowedOrigins)
	websocket.SetOriginChecker(originChecker)

	if len(cfg.WebSocket.AllowedOrigins) == 0 {
		lgr.Warn("WebSocket: allowing all origins (not recommended for production)")
	} else {
		lgr.Info("WebSocket: origin checking enabled", "allowed_origins", cfg.WebSocket.AllowedOrigins)
	}

	// Setup HTTP server
	if cfg.Logging.Level == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(logger.GinMiddleware(lgr))

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"server": cfg.Server.Name,
			"time":   time.Now().Unix(),
		})
	})

	// API routes
	apiGroup := router.Group("/api/v1")
	httpapi.RegisterRoutes(apiGroup, roomManager, lgr)

	// WebSocket endpoint
	router.GET("/ws", func(c *gin.Context) {
		websocket.HandleConnection(c.Writer, c.Request, roomManager, lgr)
	})

	// Start metrics server if enabled
	if cfg.Metrics.Enabled {
		go func() {
			metricsRouter := gin.New()
			metricsRouter.GET(cfg.Metrics.Path, gin.WrapH(promhttp.Handler()))

			lgr.Info("Starting metrics server",
				"port", cfg.Metrics.Port,
				"path", cfg.Metrics.Path,
			)

			if err := metricsRouter.Run(fmt.Sprintf(":%d", cfg.Metrics.Port)); err != nil {
				lgr.Error("Metrics server error", "error", err)
			}
		}()
	}

	// Start HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	// Start server in goroutine
	go func() {
		lgr.Info("Starting HTTP server", "address", addr)

		if cfg.Server.TLS.Enabled {
			if err := srv.ListenAndServeTLS(cfg.Server.TLS.CertFile, cfg.Server.TLS.KeyFile); err != nil && err != http.ErrServerClosed {
				lgr.Fatal("Failed to start HTTPS server", "error", err)
			}
		} else {
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				lgr.Fatal("Failed to start HTTP server", "error", err)
			}
		}
	}()

	lgr.Info("Zenc Server started successfully",
		"http_address", addr,
		"metrics_enabled", cfg.Metrics.Enabled,
	)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	lgr.Info("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		lgr.Fatal("Server forced to shutdown", "error", err)
	}

	lgr.Info("Server exited")
}
