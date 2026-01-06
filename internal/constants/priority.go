package constants

// Niveles de prioridad predefinidos
const (
	// Prioridades Altas (> 5)
	PriorityCritical  = 10  // Emergencias, competencias en vivo
	PriorityUrgent    = 8   // Exámenes, evaluaciones importantes
	PriorityHigh      = 6   // Usuarios premium, tareas importantes
	
	// Prioridades Normales (0-5)
	PriorityNormal    = 0   // Por defecto
	
	// Prioridades Bajas (< 0)
	PriorityLow       = -3  // Background tasks
	PriorityBatch     = -5  // Procesamiento batch
	PriorityMaintenance = -10 // Mantenimiento, limpieza
)

// GetPriorityName retorna el nombre descriptivo de una prioridad
func GetPriorityName(priority int) string {
	switch {
	case priority >= 10:
		return "Critical"
	case priority >= 8:
		return "Urgent"
	case priority >= 6:
		return "High"
	case priority > 0:
		return "Normal+"
	case priority == 0:
		return "Normal"
	case priority > -5:
		return "Low"
	case priority >= -10:
		return "Batch"
	default:
		return "Maintenance"
	}
}

// GetQueueName retorna el nombre de la cola según la prioridad
func GetQueueName(priority int) string {
	if priority > 5 {
		return "high"
	} else if priority < 0 {
		return "low"
	}
	return "default"
}
