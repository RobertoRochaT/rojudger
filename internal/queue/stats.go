package queue

import (
	"context"
	"strconv"
)

// Stats representa las estadísticas de la cola
type Stats struct {
	QueueHigh      int64  `json:"queue_high"`
	QueueDefault   int64  `json:"queue_default"`
	QueueLow       int64  `json:"queue_low"`
	Processing     int64  `json:"processing"`
	TotalPending   int64  `json:"total_pending"`
	TotalEnqueued  int64  `json:"total_enqueued"`
	TotalDequeued  int64  `json:"total_dequeued"`
	TotalCompleted int64  `json:"total_completed"`
	TotalFailed    int64  `json:"total_failed"`
}

// GetStatsTyped retorna estadísticas con tipos correctos
func (q *Queue) GetStatsTyped(ctx context.Context) (*Stats, error) {
	stats := &Stats{}

	// Tamaño de cada cola
	stats.QueueHigh, _ = q.client.LLen(ctx, QueueKeyHigh).Result()
	stats.QueueDefault, _ = q.client.LLen(ctx, QueueKeyDefault).Result()
	stats.QueueLow, _ = q.client.LLen(ctx, QueueKeyLow).Result()
	stats.Processing, _ = q.client.SCard(ctx, ProcessingSetKey).Result()
	stats.TotalPending = stats.QueueHigh + stats.QueueDefault + stats.QueueLow

	// Contadores totales (convertir strings a int64)
	allStats, _ := q.client.HGetAll(ctx, StatsKey).Result()
	
	if val, ok := allStats["total_enqueued"]; ok {
		stats.TotalEnqueued, _ = strconv.ParseInt(val, 10, 64)
	}
	if val, ok := allStats["total_dequeued"]; ok {
		stats.TotalDequeued, _ = strconv.ParseInt(val, 10, 64)
	}
	if val, ok := allStats["total_completed"]; ok {
		stats.TotalCompleted, _ = strconv.ParseInt(val, 10, 64)
	}
	if val, ok := allStats["total_failed"]; ok {
		stats.TotalFailed, _ = strconv.ParseInt(val, 10, 64)
	}

	return stats, nil
}
