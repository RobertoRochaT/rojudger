package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/RobertoRochaT/rojudger/internal/database"
	"github.com/RobertoRochaT/rojudger/internal/executor"
	"github.com/RobertoRochaT/rojudger/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler maneja las peticiones HTTP
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

// CreateSubmission maneja POST /submissions
// @Summary Crear una nueva submission
// @Description Envía código para ejecutar (modo síncrono con ?wait=true)
// @Accept json
// @Produce json
// @Param submission body models.SubmissionRequest true "Submission Request"
// @Param wait query bool false "Wait for execution"
// @Success 201 {object} models.Submission
// @Router /submissions [post]
func (h *Handler) CreateSubmission(c *gin.Context) {
	var req models.SubmissionRequest

	// Validar request body
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Validar que el lenguaje existe
	language, err := h.db.GetLanguage(req.LanguageID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid language_id",
			"details": err.Error(),
		})
		return
	}

	// Crear submission
	submissionID := uuid.New().String()
	submission := models.NewSubmission(req, submissionID)

	// Guardar en base de datos
	if err := h.db.CreateSubmission(submission); err != nil {
		log.Printf("Error creating submission: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create submission",
		})
		return
	}

	// Verificar si el cliente quiere esperar el resultado (modo síncrono)
	wait := c.Query("wait") == "true"

	if wait {
		// Ejecutar inmediatamente y esperar resultado
		h.executeSubmission(submission, language)

		// Devolver resultado completo
		c.JSON(http.StatusCreated, submission)
	} else {
		// Modo asíncrono - solo devolver ID y status
		// En el futuro, esto se enviará a una cola (Redis)
		go h.executeSubmission(submission, language)

		c.JSON(http.StatusCreated, models.SubmissionResponse{
			ID:     submission.ID,
			Status: submission.Status,
			Token:  submission.ID,
		})
	}
}

// GetSubmission maneja GET /submissions/:id
// @Summary Obtener una submission por ID
// @Description Obtiene el estado y resultado de una submission
// @Produce json
// @Param id path string true "Submission ID"
// @Success 200 {object} models.Submission
// @Router /submissions/{id} [get]
func (h *Handler) GetSubmission(c *gin.Context) {
	id := c.Param("id")

	submission, err := h.db.GetSubmission(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Submission not found",
		})
		return
	}

	c.JSON(http.StatusOK, submission)
}

// GetLanguages maneja GET /languages
// @Summary Obtener todos los lenguajes disponibles
// @Description Lista todos los lenguajes de programación soportados
// @Produce json
// @Success 200 {array} models.Language
// @Router /languages [get]
func (h *Handler) GetLanguages(c *gin.Context) {
	languages, err := h.db.GetAllLanguages()
	if err != nil {
		log.Printf("Error getting languages: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get languages",
		})
		return
	}

	c.JSON(http.StatusOK, languages)
}

// HealthCheck maneja GET /health
// @Summary Health check endpoint
// @Description Verifica que el servicio está funcionando
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /health [get]
func (h *Handler) HealthCheck(c *gin.Context) {
	// Verificar base de datos
	dbHealth := "ok"
	if err := h.db.Health(); err != nil {
		dbHealth = "error: " + err.Error()
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"database":  dbHealth,
		"timestamp": time.Now().Unix(),
	})
}

// executeSubmission ejecuta una submission y actualiza su estado
func (h *Handler) executeSubmission(submission *models.Submission, language *models.Language) {
	// Marcar como procesando
	submission.MarkAsProcessing()
	h.db.UpdateSubmission(submission)

	// Crear contexto con timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Ejecutar código
	result := h.executor.Execute(ctx, submission, language)

	// Actualizar submission con resultado
	if result.Error != "" {
		submission.MarkAsError(result.Error)
	} else {
		submission.MarkAsCompleted(result)
	}

	// Guardar en base de datos
	if err := h.db.UpdateSubmission(submission); err != nil {
		log.Printf("Error updating submission %s: %v", submission.ID, err)
	}
}

// GetSubmissionsByStatus maneja GET /submissions?status=queued
// @Summary Obtener submissions por estado
// @Description Lista submissions filtradas por estado
// @Produce json
// @Param status query string true "Status" Enums(queued, processing, completed, error, timeout)
// @Param limit query int false "Limit" default(10)
// @Success 200 {array} models.Submission
// @Router /submissions [get]
func (h *Handler) GetSubmissionsByStatus(c *gin.Context) {
	status := c.Query("status")
	if status == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "status query parameter is required",
		})
		return
	}

	// Validar status
	validStatuses := map[string]bool{
		models.StatusQueued:     true,
		models.StatusProcessing: true,
		models.StatusCompleted:  true,
		models.StatusError:      true,
		models.StatusTimeout:    true,
	}

	if !validStatuses[status] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid status value",
		})
		return
	}

	// Obtener limit (default 10)
	limit := 10
	if limitStr := c.Query("limit"); limitStr != "" {
		if _, err := fmt.Sscanf(limitStr, "%d", &limit); err != nil {
			limit = 10
		}
	}

	submissions, err := h.db.GetSubmissionsByStatus(status, limit)
	if err != nil {
		log.Printf("Error getting submissions: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get submissions",
		})
		return
	}

	c.JSON(http.StatusOK, submissions)
}
