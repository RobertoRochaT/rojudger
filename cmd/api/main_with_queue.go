package main

import (
	"log"

	"github.com/RobertoRochaT/rojudger/internal/config"
	"github.com/RobertoRochaT/rojudger/internal/database"
	"github.com/RobertoRochaT/rojudger/internal/executor"
	"github.com/RobertoRochaT/rojudger/internal/handlers"
	"github.com/RobertoRochaT/rojudger/internal/queue"
	"github.com/gin-gonic/gin"
)

func mainWithQueue() {
	log.Println("🚀 Starting ROJUDGER API Server with Queue...")

	// Cargar configuración
	cfg := config.Load()

	// Conectar a la base de datos
	db, err := database.NewDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Inicializar esquema y datos
	if err := db.InitSchema(); err != nil {
		log.Fatalf("Failed to initialize schema: %v", err)
	}

	if err := db.SeedLanguages(); err != nil {
		log.Fatalf("Failed to seed languages: %v", err)
	}

	// Conectar a Redis queue
	q, err := queue.NewQueue(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to queue: %v", err)
	}
	defer q.Close()

	// Crear executor (opcional para el API, realmente lo usan los workers)
	exec, err := executor.NewExecutor(cfg)
	if err != nil {
		log.Fatalf("Failed to create executor: %v", err)
	}
	defer exec.Close()

	// Crear handler con queue
	handler := handlers.NewHandlerWithQueue(db, exec, q)

	// Configurar router Gin
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// CORS middleware
	router.Use(corsMiddleware())

	// API v1
	v1 := router.Group("/api/v1")
	{
		v1.POST("/submissions", handler.CreateSubmissionAsync)
		v1.GET("/submissions/:id", handler.GetSubmission)
		v1.GET("/submissions", handler.GetSubmissions)
		v1.GET("/languages", handler.GetLanguages)
		v1.GET("/queue/stats", handler.GetQueueStats)  // ← NUEVO endpoint
	}

	// Health check
	router.GET("/health", handler.HealthCheck)

	// Iniciar servidor
	addr := cfg.ServerHost + ":" + cfg.ServerPort
	log.Printf("✅ Server listening on %s", addr)
	log.Printf("📊 Queue stats endpoint: http://%s/api/v1/queue/stats", addr)
	log.Printf("🎯 API endpoint: http://%s/api/v1/submissions", addr)

	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

