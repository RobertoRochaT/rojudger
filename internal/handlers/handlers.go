package handlers

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/RobertoRochaT/rojudger/internal/database"
	"github.com/RobertoRochaT/rojudger/internal/executor"
	"github.com/RobertoRochaT/rojudger/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler maneja las peticiones HTTP (modo directo/sincrono)
type Handler struct {
	db       *database.DB
	executor *executor.Executor
}

// NewHandler crea una nueva instancia del handler
func NewHandler(db *database.DB, exec *executor.Executor) *Handler {
	return &Handler{
		db:       db,
		executor: exec,
	}
}

// CreateSubmission maneja POST /submissions (modo directo)
func (h *Handler) CreateSubmission(c *gin.Context) {
	var req CreateSubmissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Crear submission
	submission := &models.Submission{
		ID:          uuid.New().String(),
		LanguageID:  req.LanguageID,
		SourceCode:  req.SourceCode,
		Stdin:       req.Stdin,
		ExpectedOut: req.ExpectedOutput,
		Status:      "processing",
		ExitCode:    -1,
		CreatedAt:   time.Now(),
	}

	// Guardar en base de datos
	if err := h.db.CreateSubmission(submission); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create submission"})
		return
	}

	// Obtener información del lenguaje
	language, err := h.db.GetLanguage(req.LanguageID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid language_id"})
		return
	}

	// Ejecutar código directamente (síncron)
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	result := h.executor.Execute(ctx, submission, language)

	// Actualizar submission con resultados
	now := time.Now()
	submission.FinishedAt = &now
	submission.Stdout = result.Stdout
	submission.Stderr = result.Stderr
	submission.ExitCode = result.ExitCode
	submission.Time = result.Time
	submission.Memory = result.Memory
	submission.CompileOut = result.CompileOut

	if result.TimedOut {
		submission.Status = "timeout"
		submission.Message = "Execution timed out"
	} else if result.Error != "" {
		submission.Status = "error"
		submission.Message = result.Error
	} else {
		submission.Status = "completed"
	}

	// Guardar resultados
	if err := h.db.UpdateSubmission(submission); err != nil {
		log.Printf("Failed to update submission: %v", err)
	}

	c.JSON(http.StatusOK, submission)
}

// GetSubmission maneja GET /submissions/:id
func (h *Handler) GetSubmission(c *gin.Context) {
	id := c.Param("id")

	submission, err := h.db.GetSubmission(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Submission not found"})
		return
	}

	c.JSON(http.StatusOK, submission)
}

// GetSubmissions maneja GET /submissions
func (h *Handler) GetSubmissions(c *gin.Context) {
	status := c.Query("status")
	limit := 100

	if status == "" {
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
func (h *Handler) GetLanguages(c *gin.Context) {
	languages, err := h.db.GetAllLanguages()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get languages"})
		return
	}

	c.JSON(http.StatusOK, languages)
}

// HealthCheck maneja GET /health
func (h *Handler) HealthCheck(c *gin.Context) {
	status := "healthy"
	dbStatus := "ok"

	if err := h.db.Health(); err != nil {
		status = "unhealthy"
		dbStatus = "error"
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    status,
		"database":  dbStatus,
		"timestamp": time.Now().Unix(),
	})
}
