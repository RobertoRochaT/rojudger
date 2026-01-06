package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/RobertoRochaT/rojudger/internal/config"
	"github.com/RobertoRochaT/rojudger/internal/database"
	"github.com/RobertoRochaT/rojudger/internal/executor"
	"github.com/RobertoRochaT/rojudger/internal/handlers"
	"github.com/gin-gonic/gin"
)

func mainDirect() {
	log.Println("🚀 Starting ROJUDGER API Server...")

	// Cargar configuración
	cfg := config.Load()
	log.Printf("Environment: %s", cfg.Environment)

	// Configurar Gin
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	// Conectar a la base de datos
	db, err := database.NewDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Inicializar schema
	if err := db.InitSchema(); err != nil {
		log.Fatalf("Failed to initialize database schema: %v", err)
	}

	// Seed de lenguajes
	if err := db.SeedLanguages(); err != nil {
		log.Fatalf("Failed to seed languages: %v", err)
	}

	// Crear executor
	exec, err := executor.NewExecutor(cfg)
	if err != nil {
		log.Fatalf("Failed to create executor: %v", err)
	}
	defer exec.Close()

	// Crear handlers
	h := handlers.NewHandler(db, exec)

	// Configurar router
	router := setupRouter(h)

	// Servidor con graceful shutdown
	addr := cfg.ServerHost + ":" + cfg.ServerPort
	log.Printf("🌐 Server listening on http://%s", addr)
	log.Println("📚 API Documentation:")
	log.Println("  POST   /api/v1/submissions       - Create submission")
	log.Println("  GET    /api/v1/submissions/:id   - Get submission")
	log.Println("  GET    /api/v1/submissions       - List submissions by status")
	log.Println("  GET    /api/v1/languages         - Get supported languages")
	log.Println("  GET    /health                   - Health check")

	// Capturar señales de sistema para graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := router.Run(addr); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	<-quit
	log.Println("🛑 Shutting down server...")
}

func setupRouter(h *handlers.Handler) *gin.Engine {
	router := gin.Default()

	// Middleware CORS
	router.Use(corsMiddleware())

	// Health check (sin versión)
	router.GET("/health", h.HealthCheck)

	// API v1
	v1 := router.Group("/api/v1")
	{
		// Submissions
		submissions := v1.Group("/submissions")
		{
			submissions.POST("", h.CreateSubmission)
			submissions.GET("/:id", h.GetSubmission)
			submissions.GET("", h.GetSubmissions)
		}

		// Languages
		v1.GET("/languages", h.GetLanguages)
	}

	return router
}

// corsMiddleware añade headers CORS para desarrollo
