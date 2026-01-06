package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/RobertoRochaT/rojudger/internal/database"
	"github.com/RobertoRochaT/rojudger/internal/executor"
	"github.com/RobertoRochaT/rojudger/internal/models"
	"github.com/RobertoRochaT/rojudger/internal/queue"
	"github.com/RobertoRochaT/rojudger/internal/webhook"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// HandlerWithQueue maneja las peticiones HTTP con cola Redis
type HandlerWithQueue struct {
	db       *database.DB
	executor *executor.Executor
	queue    *queue.Queue
}

// NewHandlerWithQueue crea una nueva instancia del handler con queue
func NewHandlerWithQueue(db *database.DB, exec *executor.Executor, q *queue.Queue) *HandlerWithQueue {
	return &HandlerWithQueue{
		db:       db,
		executor: exec,
		queue:    q,
	}
}

// CreateSubmissionAsync maneja POST /submissions con cola
func (h *HandlerWithQueue) CreateSubmissionAsync(c *gin.Context) {
	var req CreateSubmissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validar webhook URL si se proporciona
	if req.WebhookURL != "" {
		if err := webhook.ValidateWebhookURL(req.WebhookURL); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid webhook URL: " + err.Error()})
			return
		}
	}

	// Crear submission con status "queued"
	submission := &models.Submission{
		ID:          uuid.New().String(),
		LanguageID:  req.LanguageID,
		SourceCode:  req.SourceCode,
		Stdin:       req.Stdin,
		ExpectedOut: req.ExpectedOutput,
		WebhookURL:  req.WebhookURL,
		Status:      "queued",
		ExitCode:    -1,
		CreatedAt:   time.Now(),
	}

	// Guardar en base de datos
	if err := h.db.CreateSubmission(submission); err != nil {
		log.Printf("ERROR creating submission: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create submission", "details": err.Error()})
		return
	}

	// Obtener y validar prioridad
	priority := req.Priority

	// Validar rango de prioridad (opcionalmente limitar)
	if priority > 10 {
		priority = 10 // Máximo permitido
		log.Printf("Priority capped to 10 for submission %s", submission.ID)
	}
	if priority < -10 {
		priority = -10 // Mínimo permitido
		log.Printf("Priority floored to -10 for submission %s", submission.ID)
	}

	// Encolar para procesamiento asíncrono
	if err := h.queue.Enqueue(c.Request.Context(), submission.ID, priority); err != nil {
		log.Printf("Failed to enqueue submission %s: %v", submission.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to enqueue submission"})
		return
	}

	// Log con información de prioridad
	queueName := "default"
	if priority > 5 {
		queueName = "high"
	} else if priority < 0 {
		queueName = "low"
	}
	log.Printf("Submission %s enqueued successfully (priority: %d, queue: %s)", submission.ID, priority, queueName)

	// Verificar si el cliente quiere esperar por el resultado (modo síncrono)
	if c.Query("wait") == "true" {
		// Polling: esperar hasta que el trabajo termine
		timeout := time.After(30 * time.Second)
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-timeout:
				c.JSON(http.StatusOK, submission) // Retornar con status "queued"
				return
			case <-ticker.C:
				// Revisar si ya terminó
				updated, err := h.db.GetSubmission(submission.ID)
				if err == nil && updated.Status != "queued" && updated.Status != "processing" {
					c.JSON(http.StatusOK, updated)
					return
				}
			}
		}
	}

	// Modo asíncrono: retornar inmediatamente
	c.JSON(http.StatusCreated, submission)
}

// GetQueueStats retorna estadísticas de la cola
func (h *HandlerWithQueue) GetQueueStats(c *gin.Context) {
	stats, err := h.queue.GetStatsTyped(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get queue stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetSubmission maneja GET /submissions/:id
func (h *HandlerWithQueue) GetSubmission(c *gin.Context) {
	id := c.Param("id")

	submission, err := h.db.GetSubmission(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Submission not found"})
		return
	}

	c.JSON(http.StatusOK, submission)
}

// GetSubmissions maneja GET /submissions
func (h *HandlerWithQueue) GetSubmissions(c *gin.Context) {
	status := c.Query("status")
	limit := 100

	if status == "" {
		// Podríamos implementar una query general, por ahora retornar error
		c.JSON(http.StatusBadRequest, gin.H{"error": "status query parameter is required"})
		return
	}

	submissions, err := h.db.GetSubmissionsByStatus(status, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get submissions"})
		return
	}

	c.JSON(http.StatusOK, submissions)
}

// GetLanguages maneja GET /languages
func (h *HandlerWithQueue) GetLanguages(c *gin.Context) {
	languages, err := h.db.GetAllLanguages()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get languages"})
		return
	}

	c.JSON(http.StatusOK, languages)
}

// HealthCheck maneja GET /health
func (h *HandlerWithQueue) HealthCheck(c *gin.Context) {
	// Verificar database
	if err := h.db.Health(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":   "unhealthy",
			"database": "error",
			"queue":    "unknown",
		})
		return
	}

	// Verificar queue
	queueStatus := "ok"
	if err := h.queue.Health(c.Request.Context()); err != nil {
		queueStatus = "error"
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"database":  "ok",
		"queue":     queueStatus,
		"timestamp": time.Now().Unix(),
	})
}
