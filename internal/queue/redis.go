package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/RobertoRochaT/rojudger/internal/config"
)

// Queue maneja la cola de trabajos con Redis
type Queue struct {
	client *redis.Client
	config *config.Config
}

// Job representa un trabajo en la cola
type Job struct {
	SubmissionID string    `json:"submission_id"`
	Priority     int       `json:"priority"`
	CreatedAt    time.Time `json:"created_at"`
}

const (
	QueueKeyDefault  = "rojudger:queue:default"
	QueueKeyHigh     = "rojudger:queue:high"
	QueueKeyLow      = "rojudger:queue:low"
	ProcessingSetKey = "rojudger:processing"
	StatsKey         = "rojudger:stats"
)

// NewQueue crea una nueva instancia del queue
func NewQueue(cfg *config.Config) (*Queue, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	// Verificar conexión
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	log.Println("Redis queue connected successfully")

	return &Queue{
		client: client,
		config: cfg,
	}, nil
}

// Enqueue añade un trabajo a la cola
func (q *Queue) Enqueue(ctx context.Context, submissionID string, priority int) error {
	job := Job{
		SubmissionID: submissionID,
		Priority:     priority,
		CreatedAt:    time.Now(),
	}

	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	// Seleccionar cola según prioridad
	queueKey := QueueKeyDefault
	if priority > 5 {
		queueKey = QueueKeyHigh
	} else if priority < 0 {
		queueKey = QueueKeyLow
	}

	// Añadir a la cola (LPUSH para añadir al inicio)
	if err := q.client.LPush(ctx, queueKey, data).Err(); err != nil {
		return fmt.Errorf("failed to enqueue job: %w", err)
	}

	// Incrementar contador de trabajos encolados
	q.client.HIncrBy(ctx, StatsKey, "total_enqueued", 1)

	log.Printf("Job enqueued: %s (priority: %d, queue: %s)", submissionID, priority, queueKey)
	return nil
}

// Dequeue toma un trabajo de la cola (bloqueante)
func (q *Queue) Dequeue(ctx context.Context, timeout time.Duration) (*Job, error) {
	// BRPOP revisa múltiples colas por prioridad
	// Orden: high → default → low
	result, err := q.client.BRPop(ctx, timeout, 
		QueueKeyHigh, 
		QueueKeyDefault, 
		QueueKeyLow,
	).Result()

	if err == redis.Nil {
		// Timeout, no hay trabajos
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to dequeue job: %w", err)
	}

	// result[0] = nombre de la cola
	// result[1] = datos del job
	var job Job
	if err := json.Unmarshal([]byte(result[1]), &job); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job: %w", err)
	}

	// Marcar como en procesamiento
	q.client.SAdd(ctx, ProcessingSetKey, job.SubmissionID)
	q.client.HIncrBy(ctx, StatsKey, "total_dequeued", 1)

	log.Printf("Job dequeued: %s from %s", job.SubmissionID, result[0])
	return &job, nil
}

// MarkComplete marca un trabajo como completado
func (q *Queue) MarkComplete(ctx context.Context, submissionID string) error {
	// Remover del set de procesamiento
	q.client.SRem(ctx, ProcessingSetKey, submissionID)
	q.client.HIncrBy(ctx, StatsKey, "total_completed", 1)
	
	log.Printf("Job marked complete: %s", submissionID)
	return nil
}

// MarkFailed marca un trabajo como fallido y lo reencola (opcionalmente)
func (q *Queue) MarkFailed(ctx context.Context, submissionID string, retry bool) error {
	// Remover del set de procesamiento
	q.client.SRem(ctx, ProcessingSetKey, submissionID)
	q.client.HIncrBy(ctx, StatsKey, "total_failed", 1)

	if retry {
		// Reencolar con baja prioridad
		return q.Enqueue(ctx, submissionID, -1)
	}

	log.Printf("Job marked failed: %s (retry: %v)", submissionID, retry)
	return nil
}

// GetStats obtiene estadísticas de la cola
func (q *Queue) GetStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Tamaño de cada cola
	highSize, _ := q.client.LLen(ctx, QueueKeyHigh).Result()
	defaultSize, _ := q.client.LLen(ctx, QueueKeyDefault).Result()
	lowSize, _ := q.client.LLen(ctx, QueueKeyLow).Result()
	processing, _ := q.client.SCard(ctx, ProcessingSetKey).Result()

	stats["queue_high"] = highSize
	stats["queue_default"] = defaultSize
	stats["queue_low"] = lowSize
	stats["processing"] = processing
	stats["total_pending"] = highSize + defaultSize + lowSize

	// Contadores totales
	allStats, _ := q.client.HGetAll(ctx, StatsKey).Result()
	for k, v := range allStats {
		stats[k] = v
	}

	return stats, nil
}

// QueueLength retorna el tamaño total de la cola
func (q *Queue) QueueLength(ctx context.Context) (int64, error) {
	high, _ := q.client.LLen(ctx, QueueKeyHigh).Result()
	default_, _ := q.client.LLen(ctx, QueueKeyDefault).Result()
	low, _ := q.client.LLen(ctx, QueueKeyLow).Result()
	
	return high + default_ + low, nil
}

// Close cierra la conexión con Redis
func (q *Queue) Close() error {
	return q.client.Close()
}

// Health verifica el estado de Redis
func (q *Queue) Health(ctx context.Context) error {
	return q.client.Ping(ctx).Err()
}
