package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/RobertoRochaT/rojudger/internal/models"
)

// WebhookService maneja el envío de webhooks
type WebhookService struct {
	client     *http.Client
	timeout    time.Duration
	retries    int
	hmacSecret string
}

// NewWebhookService crea un nuevo servicio de webhooks
func NewWebhookService(timeout time.Duration, retries int, hmacSecret string) *WebhookService {
	return &WebhookService{
		client: &http.Client{
			Timeout: timeout,
		},
		timeout:    timeout,
		retries:    retries,
		hmacSecret: hmacSecret,
	}
}

// WebhookPayload es el cuerpo del webhook
type WebhookPayload struct {
	Event      string             `json:"event"`
	Submission *models.Submission `json:"submission"`
	Timestamp  time.Time          `json:"timestamp"`
}

// WebhookResult contiene el resultado de un intento de webhook
type WebhookResult struct {
	Success      bool
	StatusCode   int
	ResponseBody string
	Error        error
	Attempt      int
}

// ValidateWebhookURL valida que la URL del webhook sea segura
func ValidateWebhookURL(webhookURL string) error {
	if webhookURL == "" {
		return nil // No webhook es válido
	}

	parsedURL, err := url.Parse(webhookURL)
	if err != nil {
		return fmt.Errorf("invalid webhook URL: %w", err)
	}

	// Solo permitir HTTP y HTTPS
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("webhook URL must use http or https scheme")
	}

	// Validar que tenga un host
	if parsedURL.Host == "" {
		return fmt.Errorf("webhook URL must have a host")
	}

	// Bloquear IPs privadas en producción (opcional)
	hostname := strings.ToLower(parsedURL.Hostname())
	if hostname == "localhost" || hostname == "127.0.0.1" || hostname == "::1" {
		// Permitir localhost solo en desarrollo
		// En producción deberías descomentar esto:
		// return fmt.Errorf("webhook URL cannot point to localhost")
	}

	return nil
}

// generateHMAC genera una firma HMAC-SHA256 del payload
func (ws *WebhookService) generateHMAC(payload []byte) string {
	if ws.hmacSecret == "" {
		return ""
	}

	h := hmac.New(sha256.New, []byte(ws.hmacSecret))
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}

// Send envía un webhook con reintentos
func (ws *WebhookService) Send(ctx context.Context, webhookURL string, submission *models.Submission) *WebhookResult {
	result := &WebhookResult{}

	if webhookURL == "" {
		return result // No hay webhook, éxito silencioso
	}

	// Validar URL
	if err := ValidateWebhookURL(webhookURL); err != nil {
		result.Error = err
		return result
	}

	// Preparar payload
	payload := WebhookPayload{
		Event:      "submission.completed",
		Submission: submission,
		Timestamp:  time.Now().UTC(),
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		result.Error = fmt.Errorf("failed to marshal webhook: %w", err)
		return result
	}

	// Intentar enviar con reintentos
	var lastErr error
	for attempt := 1; attempt <= ws.retries+1; attempt++ {
		result.Attempt = attempt

		if attempt > 1 {
			backoff := time.Second * time.Duration(attempt-1)
			log.Printf("Webhook retry %d/%d for submission %s (backoff: %v)",
				attempt-1, ws.retries, submission.ID, backoff)

			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				result.Error = ctx.Err()
				return result
			}
		}

		req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewBuffer(jsonData))
		if err != nil {
			lastErr = err
			continue
		}

		// Headers
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "ROJUDGER-Webhook/1.0")
		req.Header.Set("X-Rojudger-Event", "submission.completed")
		req.Header.Set("X-Rojudger-Submission-Id", submission.ID)
		req.Header.Set("X-Rojudger-Delivery", fmt.Sprintf("%d", time.Now().Unix()))

		// HMAC signature
		if ws.hmacSecret != "" {
			signature := ws.generateHMAC(jsonData)
			req.Header.Set("X-Rojudger-Signature", signature)
		}

		resp, err := ws.client.Do(req)
		if err != nil {
			lastErr = err
			log.Printf("Webhook error (attempt %d): %v", attempt, err)
			continue
		}

		// Leer respuesta
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1024*10)) // Max 10KB
		resp.Body.Close()

		result.StatusCode = resp.StatusCode
		result.ResponseBody = string(bodyBytes)

		// Éxito si es 2xx
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			result.Success = true
			log.Printf("✅ Webhook delivered to %s (submission: %s, status: %d)",
				webhookURL, submission.ID, resp.StatusCode)
			return result
		}

		lastErr = fmt.Errorf("webhook returned status %d", resp.StatusCode)
		log.Printf("⚠️  Webhook attempt %d failed: status %d", attempt, resp.StatusCode)
	}

	result.Error = fmt.Errorf("failed after %d attempts: %w", ws.retries+1, lastErr)
	return result
}

// SendAsync envía un webhook de forma asíncrona
func (ws *WebhookService) SendAsync(webhookURL string, submission *models.Submission, logger func(submissionID, webhookURL string, attempt, statusCode int, responseBody, errorMsg string)) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), ws.timeout)
		defer cancel()

		result := ws.Send(ctx, webhookURL, submission)

		// Log del resultado
		if logger != nil {
			errorMsg := ""
			if result.Error != nil {
				errorMsg = result.Error.Error()
			}
			logger(submission.ID, webhookURL, result.Attempt, result.StatusCode, result.ResponseBody, errorMsg)
		}

		if result.Error != nil {
			log.Printf("❌ Webhook failed for submission %s: %v", submission.ID, result.Error)
		}
	}()
}
