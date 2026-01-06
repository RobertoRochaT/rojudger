# 🔔 Webhooks en ROJUDGER

Esta guía documenta el sistema completo de webhooks implementado en ROJUDGER, que permite notificaciones en tiempo real cuando una submission se completa.

---

## 📋 Tabla de Contenidos

1. [Descripción General](#descripción-general)
2. [Características](#características)
3. [Uso Básico](#uso-básico)
4. [Payload del Webhook](#payload-del-webhook)
5. [Seguridad (HMAC)](#seguridad-hmac)
6. [Validación de URLs](#validación-de-urls)
7. [Logs y Auditoría](#logs-y-auditoría)
8. [Implementar un Receptor](#implementar-un-receptor)
9. [Testing](#testing)
10. [Troubleshooting](#troubleshooting)

---

## Descripción General

El sistema de webhooks de ROJUDGER envía notificaciones HTTP POST cuando una submission completa su ejecución. Esto permite:

- **Notificaciones en tiempo real** sin polling
- **Integración con otros sistemas** (Discord, Slack, apps personalizadas)
- **Auditoría completa** con logs en base de datos
- **Seguridad robusta** con firmas HMAC-SHA256

### Arquitectura

```
┌─────────────┐
│   Cliente   │
└──────┬──────┘
       │ POST /submissions
       │ { webhook_url: "..." }
       ↓
┌─────────────┐
│ API Server  │
└──────┬──────┘
       │ Encola job
       ↓
┌─────────────┐
│   Worker    │ ──── Ejecuta código
└──────┬──────┘
       │ Al terminar
       ↓
┌─────────────┐
│  Webhook    │ ──── POST a webhook_url
│  Service    │      (3 reintentos)
└──────┬──────┘
       │
       ↓
┌─────────────┐
│ webhook_logs│ ──── Auditoría en DB
└─────────────┘
```

---

## Características

### ✅ Implementadas

- **Envío asíncrono**: No bloquea el worker
- **Reintentos automáticos**: Hasta 3 intentos con backoff exponencial
- **Firmas HMAC**: Autenticación criptográfica del payload
- **Validación de URLs**: Previene ataques SSRF
- **Logging completo**: Tabla `webhook_logs` registra cada intento
- **Headers personalizados**: Metadatos útiles en cada request
- **Timeout configurable**: 30 segundos por defecto

### 🔄 Estrategia de Reintentos

```
Intento 1: Inmediato
Intento 2: +1 segundo (backoff)
Intento 3: +2 segundos (backoff)
Total: 3 intentos con backoff exponencial
```

---

## Uso Básico

### 1. Crear Submission con Webhook

```bash
curl -X POST http://localhost:8080/api/v1/submissions \
  -H "Content-Type: application/json" \
  -d '{
    "language_id": 71,
    "source_code": "print(\"Hello World!\")",
    "webhook_url": "https://your-app.com/webhooks/rojudger"
  }'
```

**Respuesta:**

```json
{
  "id": "abc-123-def-456",
  "status": "queued",
  "language_id": 71,
  "created_at": "2024-01-15T10:30:00Z"
}
```

### 2. Con Prioridad

```bash
curl -X POST http://localhost:8080/api/v1/submissions \
  -H "Content-Type: application/json" \
  -d '{
    "language_id": 71,
    "source_code": "print(\"VIP task\")",
    "priority": 10,
    "webhook_url": "https://your-app.com/webhooks"
  }'
```

### 3. Sin Webhook (Opcional)

El campo `webhook_url` es **opcional**. Si no se proporciona, no se envía ningún webhook.

```bash
curl -X POST http://localhost:8080/api/v1/submissions \
  -H "Content-Type: application/json" \
  -d '{
    "language_id": 71,
    "source_code": "print(\"No webhook\")"
  }'
```

---

## Payload del Webhook

### Estructura JSON

Cuando la submission termina, se envía este payload:

```json
{
  "event": "submission.completed",
  "timestamp": "2024-01-15T10:30:05.123Z",
  "submission": {
    "id": "abc-123-def-456",
    "language_id": 71,
    "source_code": "print(\"Hello World!\")",
    "stdin": "",
    "expected_output": "",
    "status": "completed",
    "stdout": "Hello World!\n",
    "stderr": "",
    "exit_code": 0,
    "time": 0.123,
    "memory": 8192,
    "compile_output": "",
    "message": "",
    "created_at": "2024-01-15T10:30:00Z",
    "finished_at": "2024-01-15T10:30:05Z"
  }
}
```

### Campos Importantes

| Campo | Tipo | Descripción |
|-------|------|-------------|
| `event` | string | Siempre `"submission.completed"` |
| `timestamp` | string | UTC timestamp del webhook |
| `submission.id` | string | UUID de la submission |
| `submission.status` | string | `completed`, `error`, o `timeout` |
| `submission.stdout` | string | Salida estándar del programa |
| `submission.stderr` | string | Salida de error |
| `submission.exit_code` | int | Código de salida (0 = éxito) |
| `submission.time` | float | Tiempo de ejecución en segundos |
| `submission.memory` | int | Memoria usada en KB |

### Ejemplo: Error de Compilación

```json
{
  "event": "submission.completed",
  "timestamp": "2024-01-15T10:31:00Z",
  "submission": {
    "id": "def-456-ghi-789",
    "status": "error",
    "compile_output": "error: expected ';' at line 5",
    "exit_code": 1,
    "message": "Compilation failed"
  }
}
```

---

## Seguridad (HMAC)

### ¿Por qué HMAC?

Las firmas HMAC permiten:
1. **Verificar que el webhook viene de ROJUDGER** (autenticidad)
2. **Detectar alteraciones** en el payload (integridad)
3. **Prevenir replay attacks** (con timestamps)

### Configurar el Secreto

**1. En el servidor (worker):**

```bash
export WEBHOOK_SECRET="tu-secreto-super-seguro-aqui"
./worker
```

**2. En tu aplicación receptora:**

Guarda el mismo secreto de forma segura (variables de entorno, secrets manager, etc.)

### Verificar la Firma

#### Headers del Webhook

```http
POST /webhooks HTTP/1.1
Host: your-app.com
Content-Type: application/json
User-Agent: ROJUDGER-Webhook/1.0
X-Rojudger-Event: submission.completed
X-Rojudger-Submission-Id: abc-123-def-456
X-Rojudger-Delivery: 1705318200
X-Rojudger-Signature: a1b2c3d4e5f6... (HMAC-SHA256)
```

#### Verificación en Node.js

```javascript
const crypto = require('crypto');
const express = require('express');

const app = express();
const WEBHOOK_SECRET = process.env.WEBHOOK_SECRET;

app.post('/webhooks/rojudger', express.raw({type: 'application/json'}), (req, res) => {
  const signature = req.headers['x-rojudger-signature'];
  const body = req.body;

  // Calcular HMAC esperado
  const hmac = crypto.createHmac('sha256', WEBHOOK_SECRET);
  hmac.update(body);
  const expectedSignature = hmac.digest('hex');

  // Comparación segura
  if (signature !== expectedSignature) {
    console.error('❌ Invalid webhook signature!');
    return res.status(401).send('Unauthorized');
  }

  // ✅ Firma válida
  const payload = JSON.parse(body);
  console.log('✅ Webhook verified:', payload.submission.id);

  // Procesar webhook...

  res.json({ status: 'received' });
});

app.listen(9000, () => console.log('Webhook receiver on port 9000'));
```

#### Verificación en Python (Flask)

```python
import hmac
import hashlib
import os
from flask import Flask, request, jsonify

app = Flask(__name__)
WEBHOOK_SECRET = os.getenv('WEBHOOK_SECRET', '').encode()

@app.route('/webhooks/rojudger', methods=['POST'])
def webhook():
    signature = request.headers.get('X-Rojudger-Signature', '')
    body = request.get_data()

    # Calcular HMAC esperado
    expected = hmac.new(WEBHOOK_SECRET, body, hashlib.sha256).hexdigest()

    # Comparación segura
    if not hmac.compare_digest(signature, expected):
        print('❌ Invalid signature!')
        return jsonify({'error': 'Unauthorized'}), 401

    # ✅ Firma válida
    payload = request.get_json()
    print(f'✅ Webhook verified: {payload["submission"]["id"]}')

    # Procesar webhook...

    return jsonify({'status': 'received'})

if __name__ == '__main__':
    app.run(port=9000)
```

#### Verificación en Go

```go
package main

import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "io"
    "log"
    "net/http"
    "os"
)

var webhookSecret = []byte(os.Getenv("WEBHOOK_SECRET"))

func webhookHandler(w http.ResponseWriter, r *http.Request) {
    signature := r.Header.Get("X-Rojudger-Signature")
    
    body, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "Bad request", 400)
        return
    }

    // Calcular HMAC
    h := hmac.New(sha256.New, webhookSecret)
    h.Write(body)
    expected := hex.EncodeToString(h.Sum(nil))

    // Verificar
    if !hmac.Equal([]byte(signature), []byte(expected)) {
        log.Println("❌ Invalid signature!")
        http.Error(w, "Unauthorized", 401)
        return
    }

    // ✅ Válido
    var payload map[string]interface{}
    json.Unmarshal(body, &payload)
    log.Printf("✅ Webhook verified: %v", payload["submission"])

    w.WriteHeader(200)
    json.NewEncoder(w).Encode(map[string]string{"status": "received"})
}

func main() {
    http.HandleFunc("/webhooks/rojudger", webhookHandler)
    log.Println("Listening on :9000")
    http.ListenAndServe(":9000", nil)
}
```

---

## Validación de URLs

### URLs Permitidas

- ✅ `http://example.com/webhook`
- ✅ `https://api.myapp.com/webhooks/rojudger`
- ✅ `http://localhost:9000` (desarrollo)

### URLs Rechazadas

- ❌ `ftp://example.com` (solo HTTP/HTTPS)
- ❌ `javascript:alert(1)` (esquemas no permitidos)
- ❌ `http://` (sin host)
- ❌ URLs malformadas

### En Producción

Para máxima seguridad, puedes bloquear IPs privadas:

```go
// En internal/webhook/webhook.go (línea ~75)
if hostname == "localhost" || hostname == "127.0.0.1" || hostname == "::1" {
    return fmt.Errorf("webhook URL cannot point to localhost")
}

// Bloquear rangos privados
if strings.HasPrefix(hostname, "192.168.") || 
   strings.HasPrefix(hostname, "10.") ||
   strings.HasPrefix(hostname, "172.") {
    return fmt.Errorf("webhook URL cannot point to private IPs")
}
```

---

## Logs y Auditoría

### Tabla `webhook_logs`

Cada intento de envío se registra:

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

### Consultar Logs

```bash
sqlite3 rojudger.db "SELECT * FROM webhook_logs ORDER BY created_at DESC LIMIT 10;"
```

**Ejemplo de salida:**

```
id  submission_id           webhook_url              attempt  status_code  error
1   abc-123                 http://localhost:9000    1        200          
2   def-456                 http://example.com       1        0            connection refused
3   def-456                 http://example.com       2        0            connection refused
4   def-456                 http://example.com       3        0            connection refused
```

### Logs en la Aplicación

```
Worker #1: Sending webhook for submission abc-123 to http://localhost:9000
✅ Webhook delivered to http://localhost:9000 (submission: abc-123, status: 200)
```

---

## Implementar un Receptor

### Servidor Mínimo (Python)

```python
#!/usr/bin/env python3
from http.server import HTTPServer, BaseHTTPRequestHandler
import json

class WebhookHandler(BaseHTTPRequestHandler):
    def do_POST(self):
        length = int(self.headers.get('Content-Length', 0))
        body = self.rfile.read(length)
        
        payload = json.loads(body)
        print(f"Received webhook for submission: {payload['submission']['id']}")
        print(f"Status: {payload['submission']['status']}")
        print(f"Stdout: {payload['submission']['stdout']}")
        
        self.send_response(200)
        self.end_headers()
        self.wfile.write(b'OK')

if __name__ == '__main__':
    server = HTTPServer(('0.0.0.0', 9000), WebhookHandler)
    print('Webhook server on port 9000')
    server.serve_forever()
```

### Servidor con Express (Node.js)

```javascript
const express = require('express');
const app = express();

app.use(express.json());

app.post('/webhooks/rojudger', (req, res) => {
  const { event, submission } = req.body;
  
  console.log(`Webhook received: ${event}`);
  console.log(`Submission ${submission.id}: ${submission.status}`);
  
  if (submission.status === 'completed') {
    console.log(`Output: ${submission.stdout}`);
  } else if (submission.status === 'error') {
    console.log(`Error: ${submission.message}`);
  }
  
  res.json({ status: 'received', id: submission.id });
});

app.listen(9000, () => {
  console.log('Webhook receiver listening on port 9000');
});
```

### Integración con Discord

```javascript
const axios = require('axios');

app.post('/webhooks/rojudger', async (req, res) => {
  const { submission } = req.body;
  
  const discordWebhook = process.env.DISCORD_WEBHOOK_URL;
  
  const embed = {
    title: `Submission ${submission.status}`,
    color: submission.status === 'completed' ? 0x00FF00 : 0xFF0000,
    fields: [
      { name: 'ID', value: submission.id, inline: true },
      { name: 'Exit Code', value: submission.exit_code.toString(), inline: true },
      { name: 'Time', value: `${submission.time}s`, inline: true },
      { name: 'Output', value: '```\n' + (submission.stdout || 'empty') + '\n```' }
    ],
    timestamp: new Date().toISOString()
  };
  
  await axios.post(discordWebhook, { embeds: [embed] });
  
  res.json({ status: 'sent to discord' });
});
```

---

## Testing

### 1. Script Automatizado

```bash
cd ROJUDGER
./scripts/test_webhooks.sh
```

Este script:
- ✅ Inicia un servidor webhook de prueba en puerto 9000
- ✅ Ejecuta 6 tests diferentes
- ✅ Verifica firmas HMAC
- ✅ Muestra logs completos
- ✅ Consulta la tabla `webhook_logs`

### 2. Webhook.site (Servicio Online)

```bash
# 1. Ir a https://webhook.site y copiar la URL
# 2. Enviar submission:

curl -X POST http://localhost:8080/api/v1/submissions \
  -H "Content-Type: application/json" \
  -d '{
    "language_id": 71,
    "source_code": "print(\"Test\")",
    "webhook_url": "https://webhook.site/tu-uuid-unico"
  }'

# 3. Ver el webhook en tiempo real en webhook.site
```

### 3. ngrok (URLs Públicas para Testing Local)

```bash
# Terminal 1: Iniciar receptor local
python3 webhook_receiver.py

# Terminal 2: Exponer con ngrok
ngrok http 9000

# Terminal 3: Enviar submission
curl -X POST http://localhost:8080/api/v1/submissions \
  -d '{
    "language_id": 71,
    "source_code": "print(\"ngrok test\")",
    "webhook_url": "https://abc123.ngrok.io/webhook"
  }'
```

### 4. Test Manual con HMAC

```bash
# 1. Configurar secreto
export WEBHOOK_SECRET="mi-secreto-123"

# 2. Reiniciar worker con el secreto
./worker

# 3. Tu receptor debe verificar la firma (ver ejemplos arriba)
```

---

## Troubleshooting

### ❌ Webhook no se envía

**Posibles causas:**

1. **Worker no está corriendo**
   ```bash
   ps aux | grep worker
   # Si no hay output, iniciar worker:
   ./worker
   ```

2. **Submission no terminó**
   ```bash
   curl http://localhost:8080/api/v1/submissions/abc-123
   # Verificar que status sea "completed", "error" o "timeout"
   ```

3. **URL inválida**
   ```bash
   # Revisar logs del API:
   # "Invalid webhook URL: ..."
   ```

### ❌ Firma HMAC no coincide

1. **Secreto diferente entre worker y receptor**
   ```bash
   # Worker:
   echo $WEBHOOK_SECRET
   
   # Tu app:
   # Verificar que sea idéntico
   ```

2. **Orden incorrecto al calcular HMAC**
   ```javascript
   // ❌ INCORRECTO: parsear JSON primero
   const payload = JSON.parse(body);
   const hmac = crypto.createHmac('sha256', secret);
   hmac.update(JSON.stringify(payload)); // ¡Diferente!
   
   // ✅ CORRECTO: usar raw body
   const hmac = crypto.createHmac('sha256', secret);
   hmac.update(body); // Body sin parsear
   ```

### ❌ Timeout al enviar webhook

1. **URL no responde**
   ```bash
   curl -X POST https://your-webhook-url.com/webhook \
     -d '{"test": true}'
   # Verificar que responda < 30s
   ```

2. **Firewall bloqueando**
   ```bash
   # Verificar que el puerto esté abierto
   telnet your-server.com 9000
   ```

### ❌ Reintentos excesivos

Si ves muchos reintentos en los logs:

```bash
# Revisar logs de webhook
sqlite3 rojudger.db "
  SELECT submission_id, webhook_url, attempt, status_code, error 
  FROM webhook_logs 
  WHERE attempt > 1 
  ORDER BY created_at DESC 
  LIMIT 20;
"
```

**Solución:** Asegúrate de que tu webhook receptor:
- Responda con status 200-299
- Responda en < 30 segundos
- No tenga rate limiting muy agresivo

---

## Mejores Prácticas

### 1. Idempotencia

Tu receptor debe ser idempotente (manejar duplicados):

```javascript
const processedIds = new Set();

app.post('/webhooks', (req, res) => {
  const submissionId = req.body.submission.id;
  
  if (processedIds.has(submissionId)) {
    console.log('Already processed, skipping');
    return res.json({ status: 'duplicate' });
  }
  
  processedIds.add(submissionId);
  
  // Procesar...
  
  res.json({ status: 'received' });
});
```

### 2. Respuesta Rápida

```javascript
app.post('/webhooks', async (req, res) => {
  // ✅ Responder inmediatamente
  res.json({ status: 'received' });
  
  // ❌ NO esperar procesamiento largo
  // await longRunningTask(); // Esto puede causar timeout
  
  // ✅ Procesar en background
  setImmediate(() => processWebhook(req.body));
});
```

### 3. Validación del Payload

```javascript
app.post('/webhooks', (req, res) => {
  const { event, submission } = req.body;
  
  if (event !== 'submission.completed') {
    return res.status(400).json({ error: 'Unknown event' });
  }
  
  if (!submission || !submission.id) {
    return res.status(400).json({ error: 'Invalid payload' });
  }
  
  // Procesar...
});
```

### 4. Rate Limiting

```javascript
const rateLimit = require('express-rate-limit');

const limiter = rateLimit({
  windowMs: 1 * 60 * 1000, // 1 minuto
  max: 100, // 100 requests por minuto
  message: 'Too many webhooks'
});

app.post('/webhooks', limiter, (req, res) => {
  // ...
});
```

---

## Próximas Mejoras

### En Roadmap

- [ ] Dead Letter Queue (DLQ) para webhooks fallidos
- [ ] Webhook retry policy configurable
- [ ] Soporte para múltiples webhooks por submission
- [ ] Eventos adicionales: `submission.queued`, `submission.processing`
- [ ] Dashboard web para ver webhook logs
- [ ] Webhook replay (reenviar manualmente)

### Contribuir

¿Tienes ideas? Abre un issue en GitHub o envía un PR.

---

## Referencias

- **Código fuente**: `internal/webhook/webhook.go`
- **Handlers**: `internal/handlers/handlers_queue.go`
- **Worker**: `cmd/worker/main.go`
- **Tests**: `scripts/test_webhooks.sh`
- **Tabla DB**: `internal/database/database.go` (línea 83)

---

## Soporte

Si tienes problemas:

1. Revisa esta documentación
2. Ejecuta `./scripts/test_webhooks.sh` para diagnosticar
3. Consulta los logs: `sqlite3 rojudger.db "SELECT * FROM webhook_logs"`
4. Abre un issue en GitHub con logs completos

---

**¡Happy webhook coding! 🚀**