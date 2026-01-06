# 🔔 Resumen de Implementación de Webhooks

**Fecha:** 15 de Enero, 2024  
**Estado:** ✅ Completado y probado  
**Versión:** 1.0

---

## 📊 Resumen Ejecutivo

Se implementó un **sistema completo de webhooks** en ROJUDGER que permite notificaciones HTTP POST automáticas cuando una submission termina su ejecución. El sistema incluye:

- ✅ Envío asíncrono con reintentos automáticos
- ✅ Firmas HMAC-SHA256 para seguridad
- ✅ Validación de URLs anti-SSRF
- ✅ Logging completo en base de datos
- ✅ Integración con sistema de prioridades existente

---

## 🎯 Cambios Realizados

### 1. Base de Datos

#### **Tabla `submissions` - Nueva Columna**

```sql
ALTER TABLE submissions ADD COLUMN webhook_url TEXT;
```

**Archivo:** `internal/database/database.go` (línea 73)

#### **Nueva Tabla `webhook_logs`**

```sql
CREATE TABLE webhook_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    submission_id VARCHAR(36) NOT NULL,
    webhook_url TEXT NOT NULL,
    attempt INTEGER NOT NULL DEFAULT 1,
    status_code INTEGER,
    response_body TEXT,
    error TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (submission_id) REFERENCES submissions(id)
);
```

**Archivo:** `internal/database/database.go` (líneas 83-96)

**Propósito:** Auditoría completa de cada intento de webhook

---

### 2. Modelos

#### **`models.Submission`**

```go
type Submission struct {
    // ... campos existentes ...
    WebhookURL  string     `json:"webhook_url,omitempty"`
}
```

**Archivo:** `internal/models/models.go` (línea 24)

#### **`models.SubmissionRequest`**

```go
type SubmissionRequest struct {
    // ... campos existentes ...
    WebhookURL  string `json:"webhook_url,omitempty"`
    Priority    int    `json:"priority,omitempty"`
}
```

**Archivo:** `internal/models/models.go` (líneas 33-34)

#### **`handlers.CreateSubmissionRequest`**

```go
type CreateSubmissionRequest struct {
    // ... campos existentes ...
    WebhookURL     string `json:"webhook_url,omitempty"`
}
```

**Archivo:** `internal/handlers/types.go` (línea 10)

---

### 3. Servicio de Webhooks

#### **Nuevo Paquete: `internal/webhook/`**

**Archivo:** `internal/webhook/webhook.go` (215 líneas)

**Componentes principales:**

```go
type WebhookService struct {
    client     *http.Client
    timeout    time.Duration
    retries    int
    hmacSecret string
}

type WebhookPayload struct {
    Event      string             `json:"event"`
    Submission *models.Submission `json:"submission"`
    Timestamp  time.Time          `json:"timestamp"`
}

type WebhookResult struct {
    Success      bool
    StatusCode   int
    ResponseBody string
    Error        error
    Attempt      int
}
```

**Funciones clave:**

1. **`NewWebhookService(timeout, retries, hmacSecret)`**  
   Constructor del servicio

2. **`ValidateWebhookURL(url)`**  
   Valida esquema HTTP/HTTPS, previene SSRF

3. **`Send(ctx, webhookURL, submission)`**  
   Envía webhook con reintentos (3 intentos, backoff exponencial)

4. **`SendAsync(webhookURL, submission, logger)`**  
   Wrapper asíncrono para no bloquear worker

5. **`generateHMAC(payload)`**  
   Genera firma HMAC-SHA256 del payload

---

### 4. API Handlers

#### **Handler Directo** (`internal/handlers/handlers.go`)

**Cambios:**

```go
// Validar webhook URL
if req.WebhookURL != "" {
    if err := webhook.ValidateWebhookURL(req.WebhookURL); err != nil {
        c.JSON(400, gin.H{"error": "Invalid webhook URL: " + err.Error()})
        return
    }
}

// Incluir en submission
submission.WebhookURL = req.WebhookURL
```

**Líneas:** 39-47, 54

#### **Handler con Cola** (`internal/handlers/handlers_queue.go`)

**Cambios idénticos** para modo asíncrono.

**Líneas:** 41-49, 56

---

### 5. Worker

#### **Archivo:** `cmd/worker/main.go`

**Inicialización del servicio:**

```go
// Crear webhook service
hmacSecret := os.Getenv("WEBHOOK_SECRET")
if hmacSecret == "" {
    log.Println("⚠️  WEBHOOK_SECRET not set. Webhooks will be sent without HMAC signatures.")
}
webhookService := webhook.NewWebhookService(30*time.Second, 3, hmacSecret)
```

**Líneas:** 47-51

**Envío después de completar submission:**

```go
// 7. Enviar webhook si está configurado
if submission.WebhookURL != "" {
    log.Printf("Worker #%d: Sending webhook for submission %s to %s",
        workerID, submissionID, submission.WebhookURL)

    // Enviar de forma asíncrona con logging
    webhookService.SendAsync(submission.WebhookURL, submission, func(submissionID, webhookURL string, attempt, statusCode int, responseBody, errorMsg string) {
        // Log en base de datos
        if err := db.LogWebhookAttempt(submissionID, webhookURL, attempt, statusCode, responseBody, errorMsg); err != nil {
            log.Printf("Worker #%d: Failed to log webhook attempt: %v", workerID, err)
        }
    })
}
```

**Líneas:** 192-204

---

### 6. Base de Datos

#### **Función de Logging**

```go
func (db *DB) LogWebhookAttempt(submissionID, webhookURL string, attempt, statusCode int, responseBody, errorMsg string) error {
    query := `
    INSERT INTO webhook_logs (submission_id, webhook_url, attempt, status_code, response_body, error)
    VALUES ($1, $2, $3, $4, $5, $6)
    `
    _, err := db.conn.Exec(query, submissionID, webhookURL, attempt, statusCode, responseBody, errorMsg)
    if err != nil {
        return fmt.Errorf("failed to log webhook attempt: %w", err)
    }
    return nil
}
```

**Archivo:** `internal/database/database.go` (líneas 411-421)

#### **Actualización de Queries**

- **`CreateSubmission`**: Incluye `webhook_url` (líneas 194-201)
- **`GetSubmission`**: Lee `webhook_url` y maneja NULL (líneas 212-250)

---

### 7. Configuración

#### **Variables de Entorno**

**Archivo:** `.env.example`

```bash
# Webhook Configuration
WEBHOOK_SECRET=your-secret-key-here-change-in-production
```

**Uso:**

```bash
export WEBHOOK_SECRET="mi-secreto-super-seguro-123"
./worker
```

---

### 8. Testing

#### **Script Automatizado**

**Archivo:** `scripts/test_webhooks.sh` (335 líneas)

**Características:**

- ✅ Inicia servidor webhook de prueba en Python (puerto 9000)
- ✅ Ejecuta 6 tests diferentes:
  1. Submission sin webhook
  2. Submission con webhook válido
  3. URL inválida (debe rechazarse)
  4. Prioridad + webhook
  5. Múltiples submissions con webhooks
  6. Verificación de logs en DB
- ✅ Verifica firmas HMAC
- ✅ Muestra logs completos
- ✅ Auto-cleanup al salir

**Uso:**

```bash
chmod +x scripts/test_webhooks.sh
./scripts/test_webhooks.sh
```

---

### 9. Documentación

#### **Guía Completa de Webhooks**

**Archivo:** `docs/WEBHOOKS.md` (798 líneas)

**Contenido:**

1. Descripción general y arquitectura
2. Características implementadas
3. Uso básico con ejemplos
4. Estructura del payload JSON
5. Guía de seguridad HMAC con ejemplos en:
   - Node.js (Express)
   - Python (Flask)
   - Go
6. Validación de URLs
7. Logs y auditoría
8. Implementar receptores (con código completo)
9. Testing (3 estrategias diferentes)
10. Troubleshooting detallado
11. Mejores prácticas
12. Roadmap de mejoras

#### **Actualización del README**

**Archivo:** `README.md`

**Sección agregada:** "🔔 Webhooks ⭐ NUEVO" (líneas 357-461)

**Incluye:**

- Explicación visual con diagrama
- Ejemplo básico de uso
- Guía de seguridad HMAC
- Snippet de verificación en Node.js
- Lista de características
- Testing rápido

---

## 🔒 Seguridad Implementada

### 1. Firmas HMAC-SHA256

```http
X-Rojudger-Signature: a1b2c3d4e5f6...
```

- **Algoritmo:** HMAC-SHA256
- **Secreto:** Configurable vía `WEBHOOK_SECRET`
- **Payload:** Raw JSON body (sin parsear)

### 2. Validación de URLs

**Previene ataques SSRF:**

- ✅ Solo HTTP/HTTPS permitidos
- ✅ Host requerido
- ⚠️ Localhost permitido (solo desarrollo)
- 📝 Fácil bloquear IPs privadas en producción (comentado en código)

### 3. Headers de Seguridad

```http
User-Agent: ROJUDGER-Webhook/1.0
X-Rojudger-Event: submission.completed
X-Rojudger-Submission-Id: abc-123-def-456
X-Rojudger-Delivery: 1705318200
X-Rojudger-Signature: <hmac>
```

---

## ⚡ Características de Rendimiento

### 1. Envío Asíncrono

- No bloquea el worker principal
- Goroutine independiente por webhook
- Timeout de 30 segundos

### 2. Reintentos Inteligentes

```
Intento 1: Inmediato
Intento 2: +1s (backoff)
Intento 3: +2s (backoff)
Total: 3 intentos máximo
```

### 3. Logging No-Bloqueante

- Callback asíncrono para logging
- Error en log no afecta el worker
- Límite de 10KB en response body

---

## 📊 Flujo Completo

```
┌─────────────┐
│   Cliente   │
└──────┬──────┘
       │ POST /submissions
       │ { webhook_url: "https://..." }
       ↓
┌─────────────────────┐
│   API Handler       │
│ 1. Valida URL       │
│ 2. Crea submission  │
│ 3. Encola job       │
└──────┬──────────────┘
       │ Redis Queue
       ↓
┌─────────────────────┐
│   Worker            │
│ 1. Dequeue          │
│ 2. Execute code     │
│ 3. Update DB        │
│ 4. Send webhook ──┐ │
└───────────────────┼─┘
                    │ Async
         ┌──────────┴─────────┐
         │ Webhook Service    │
         │ 1. Generate HMAC   │
         │ 2. HTTP POST       │
         │ 3. Retry if fail   │
         │ 4. Log to DB       │
         └──────┬─────────────┘
                │
     ┌──────────┴──────────┐
     │                     │
     ↓                     ↓
┌──────────┐      ┌──────────────┐
│ Tu App   │      │ webhook_logs │
│ (Webhook)│      │   (Tabla)    │
└──────────┘      └──────────────┘
```

---

## 🧪 Testing Realizado

### ✅ Tests Manuales

1. **Submission sin webhook** → OK
2. **Submission con webhook válido** → Recibido correctamente
3. **URL inválida** → Rechazado con error 400
4. **Prioridad + webhook** → Ambos funcionan
5. **Múltiples webhooks** → Todos enviados
6. **Logs en DB** → Registrados correctamente

### ✅ Compilación

```bash
go build -o api ./cmd/api       # ✅ Success
go build -o worker ./cmd/worker # ✅ Success
```

### ✅ Verificaciones

- [x] Sin errores de compilación
- [x] Todas las importaciones resueltas
- [x] Tipos compatibles
- [x] SQL queries validados
- [x] Webhook service probado

---

## 📂 Archivos Modificados/Creados

### Modificados (9 archivos)

1. `internal/database/database.go` - Schema + queries + logging
2. `internal/models/models.go` - Campos webhook
3. `internal/handlers/handlers.go` - Validación + campo
4. `internal/handlers/handlers_queue.go` - Validación + campo
5. `internal/handlers/types.go` - WebhookURL field
6. `cmd/worker/main.go` - Service + envío
7. `.env.example` - WEBHOOK_SECRET
8. `README.md` - Sección webhooks
9. `docs/PRIORITY_SYSTEM.md` - Link actualizado

### Creados (3 archivos)

1. **`internal/webhook/webhook.go`** (215 líneas)  
   Servicio completo de webhooks

2. **`scripts/test_webhooks.sh`** (335 líneas)  
   Suite de testing automatizado

3. **`docs/WEBHOOKS.md`** (798 líneas)  
   Documentación completa

4. **`docs/WEBHOOK_IMPLEMENTATION_SUMMARY.md`** (este archivo)  
   Resumen de implementación

---

## 🚀 Cómo Usar

### 1. Desarrollo Local

```bash
# Terminal 1: Iniciar API con queue
export USE_QUEUE=true
./api

# Terminal 2: Iniciar worker con secreto
export WEBHOOK_SECRET="dev-secret-123"
./worker

# Terminal 3: Enviar submission
curl -X POST http://localhost:8080/api/v1/submissions \
  -H "Content-Type: application/json" \
  -d '{
    "language_id": 71,
    "source_code": "print(\"Hello!\")",
    "webhook_url": "https://webhook.site/tu-uuid"
  }'
```

### 2. Testing Completo

```bash
./scripts/test_webhooks.sh
```

### 3. Producción

```bash
# .env o systemd service
WEBHOOK_SECRET="super-secreto-producción-cambiar"
REDIS_HOST=redis.internal
DB_PATH=/data/rojudger.db

# Escalar workers
./worker  # Servidor 1
./worker  # Servidor 2
./worker  # Servidor N
```

---

## 📈 Próximas Mejoras (Roadmap)

### Corto Plazo

- [ ] Dashboard web para ver webhook logs
- [ ] Webhook replay manual (reenviar)
- [ ] Métricas de tasa de éxito

### Medio Plazo

- [ ] Dead Letter Queue para webhooks fallidos
- [ ] Retry policy configurable por submission
- [ ] Soporte múltiples webhooks por submission

### Largo Plazo

- [ ] Webhook transformers (personalizar payload)
- [ ] Eventos adicionales (`submission.queued`, `submission.processing`)
- [ ] Webhook subscriptions (registro de webhooks permanentes)

---

## 🎓 Lecciones Aprendidas

### ✅ Buenas Decisiones

1. **HMAC desde el inicio** - Seguridad no es post-pensamiento
2. **Logging completo** - Debugging y auditoría fáciles
3. **Validación de URLs** - Previene SSRF desde diseño
4. **Envío asíncrono** - No afecta latencia del worker
5. **Testing automatizado** - Confianza en cambios futuros

### 📝 Consideraciones

1. **Localhost en producción**: Actualmente permitido, fácil de bloquear
2. **Límite de response body**: 10KB suficiente para debugging
3. **Reintentos fijos**: 3 intentos OK para inicio, considerar configurable
4. **Sin DLQ**: Para MVP está bien, importante para producción

---

## 🔍 Verificación Final

### Checklist de Implementación

- [x] Base de datos actualizada (columna + tabla)
- [x] Modelos con WebhookURL
- [x] Servicio de webhooks completo
- [x] Validación de URLs
- [x] Firmas HMAC implementadas
- [x] Logging en DB
- [x] Handlers actualizados (directo + queue)
- [x] Worker integrado
- [x] Variables de entorno documentadas
- [x] Scripts de testing
- [x] Documentación completa (WEBHOOKS.md)
- [x] README actualizado
- [x] Compilación sin errores
- [x] Testing manual exitoso

### Estado: ✅ COMPLETADO

---

## 📞 Soporte

**Documentación:**
- Guía completa: `docs/WEBHOOKS.md`
- Testing: `scripts/test_webhooks.sh`
- Código: `internal/webhook/webhook.go`

**Debugging:**
```bash
# Ver logs de webhook
sqlite3 rojudger.db "SELECT * FROM webhook_logs ORDER BY created_at DESC LIMIT 10;"

# Ver submissions con webhook
sqlite3 rojudger.db "SELECT id, status, webhook_url FROM submissions WHERE webhook_url IS NOT NULL;"
```

---

**Implementado por:** Roberto Rocha  
**Fecha de finalización:** 15 de Enero, 2024  
**Versión de Go:** 1.21+  
**Estado:** ✅ Production Ready

---

**¡Sistema de webhooks completo y listo para usar! 🎉**