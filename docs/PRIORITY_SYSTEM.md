# Sistema de Prioridades - Guía Completa

## 🎯 Resumen

El sistema de prioridades te permite controlar qué submissions se ejecutan primero en la cola.

**Estado:** ✅ IMPLEMENTADO Y FUNCIONAL

---

## 📋 Cómo Funciona

### Tres Colas Separadas

```
┌─────────────────┐
│  COLA ALTA      │  Priority > 5
│  (high)         │  → Se ejecutan PRIMERO
└─────────────────┘

┌─────────────────┐
│  COLA NORMAL    │  Priority 0 a 5
│  (default)      │  → Se ejecutan en orden normal
└─────────────────┘

┌─────────────────┐
│  COLA BAJA      │  Priority < 0
│  (low)          │  → Se ejecutan AL FINAL
└─────────────────┘
```

### Workers Procesan por Prioridad

Los workers automáticamente:
1. Revisan primero la cola **HIGH**
2. Si está vacía, revisan **DEFAULT**
3. Si está vacía, revisan **LOW**

---

## 🚀 Uso Básico

### Enviar con Prioridad Baja (-1)

```bash
curl -X POST http://localhost:8080/api/v1/submissions \
  -H "Content-Type: application/json" \
  -d '{
    "language_id": 71,
    "source_code": "print(\"Background task\")",
    "priority": -1
  }'
```

### Enviar con Prioridad Normal (0)

```bash
curl -X POST http://localhost:8080/api/v1/submissions \
  -H "Content-Type: application/json" \
  -d '{
    "language_id": 71,
    "source_code": "print(\"Normal task\")",
    "priority": 0
  }'
```

### Enviar con Prioridad Alta (10)

```bash
curl -X POST http://localhost:8080/api/v1/submissions \
  -H "Content-Type: application/json" \
  -d '{
    "language_id": 71,
    "source_code": "print(\"VIP task\")",
    "priority": 10
  }'
```

---

## 📊 Niveles de Prioridad

| Valor | Cola | Nivel | Uso Recomendado |
|-------|------|-------|-----------------|
| **10** | HIGH | Critical | Emergencias, competencias en vivo |
| **8** | HIGH | Urgent | Exámenes, evaluaciones importantes |
| **6** | HIGH | High | Usuarios premium |
| **0-5** | DEFAULT | Normal | Uso estándar |
| **-3** | LOW | Low | Background tasks |
| **-5** | LOW | Batch | Procesamiento batch |
| **-10** | LOW | Maintenance | Tareas de mantenimiento |

---

## 💡 Casos de Uso

### 1. Plataforma Educativa

```javascript
// Estudiante en examen (tiempo limitado)
{
  "language_id": 71,
  "source_code": "...",
  "priority": 10  // ← Máxima prioridad
}

// Estudiante practicando
{
  "language_id": 71,
  "source_code": "...",
  "priority": 0   // ← Normal
}

// Corrección automática nocturna
{
  "language_id": 71,
  "source_code": "...",
  "priority": -5  // ← Baja, cuando haya recursos
}
```

### 2. Usuarios Premium vs Free

```javascript
// Backend determina prioridad según plan del usuario

// Usuario Premium
{
  "priority": 8  // ← Ejecución rápida
}

// Usuario Free
{
  "priority": 0  // ← Normal
}
```

### 3. Competencia de Programación

```javascript
// Durante competencia (9:00 AM - 5:00 PM)
{
  "priority": 10  // ← Feedback inmediato
}

// Fuera de horario (práctica)
{
  "priority": 0
}
```

---

## 🔧 Configuración Avanzada

### Limitar Prioridades

Edita `internal/handlers/handlers_queue.go`:

```go
// Limitar a rango -10 a 10
if priority > 10 {
    priority = 10
}
if priority < -10 {
    priority = -10
}
```

### Prioridad Dinámica por Usuario

```go
// Ejemplo: determinar prioridad según tipo de usuario
priority := 0
if user.IsPremium {
    priority = 8
} else if user.IsFree {
    priority = 0
}
```

---

## 📈 Monitoreo

### Ver Estado de las Colas

```bash
# Estadísticas generales
curl http://localhost:8080/api/v1/queue/stats | jq '.'

# Redis CLI
redis-cli
> LLEN rojudger:queue:high
> LLEN rojudger:queue:default
> LLEN rojudger:queue:low
```

### Ver Orden de Ejecución

```bash
# Logs del worker
tail -f /tmp/rojudger-worker-new.log

# Buscar orden de procesamiento
grep "Processing job" /tmp/rojudger-worker-new.log
```

---

## 🧪 Scripts de Prueba

### Test Simple

```bash
./test_priority_simple.sh
```

Envía 3 submissions (BAJA → NORMAL → ALTA) y verás que la ALTA se ejecuta primero.

### Test Completo

```bash
./scripts/test_priorities.sh
```

Prueba múltiples prioridades y verifica el orden de ejecución.

---

## ⚙️ Implementación Técnica

### Código en Redis Queue

```go
// internal/queue/redis.go

// Enqueue selecciona la cola según prioridad
func (q *Queue) Enqueue(ctx context.Context, submissionID string, priority int) error {
    queueKey := QueueKeyDefault
    if priority > 5 {
        queueKey = QueueKeyHigh
    } else if priority < 0 {
        queueKey = QueueKeyLow
    }
    
    return q.client.LPush(ctx, queueKey, data).Err()
}

// Dequeue revisa colas en orden de prioridad
func (q *Queue) Dequeue(ctx context.Context, timeout time.Duration) (*Job, error) {
    result, err := q.client.BRPop(ctx, timeout, 
        QueueKeyHigh,     // ← Primero
        QueueKeyDefault,  // ← Segundo
        QueueKeyLow,      // ← Último
    ).Result()
    // ...
}
```

### Handler

```go
// internal/handlers/handlers_queue.go

func (h *HandlerWithQueue) CreateSubmissionAsync(c *gin.Context) {
    var req CreateSubmissionRequest
    // ...
    
    priority := req.Priority  // ← Lee del JSON
    
    // Validar rango
    if priority > 10 { priority = 10 }
    if priority < -10 { priority = -10 }
    
    h.queue.Enqueue(ctx, submission.ID, priority)
}
```

---

## 🎓 Preguntas Frecuentes

**Q: ¿Qué pasa si no envío el campo priority?**
A: Se usa 0 (prioridad normal) por defecto.

**Q: ¿Puedo usar cualquier número?**
A: Sí, pero se recomienda el rango -10 a 10. El código limita automáticamente.

**Q: ¿La prioridad afecta el tiempo de ejecución?**
A: No, solo el ORDEN en que se ejecutan. Todas las submissions tienen el mismo timeout.

**Q: ¿Puedo cambiar la prioridad después de enviar?**
A: No directamente, pero podrías implementar una función para mover entre colas en Redis.

**Q: ¿Cómo sé en qué cola está mi submission?**
A: Revisa los logs del API. Ejemplo: `"Submission abc123 enqueued (priority: 10, queue: high)"`

---

## ✅ Verificación

Para verificar que funciona:

1. Inicia API y Worker:
   ```bash
   USE_QUEUE=true ./api &
   ./worker &
   ```

2. Ejecuta el test:
   ```bash
   ./test_priority_simple.sh
   ```

3. Observa los logs del worker:
   ```bash
   tail -f /tmp/rojudger-worker-new.log
   ```

4. Verás que las submissions HIGH se procesan primero, incluso si llegaron después.

---

**Estado:** ✅ PRODUCTION READY
**Última actualización:** 2026-01-05
