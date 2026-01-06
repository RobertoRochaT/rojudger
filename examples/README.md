# 🔔 ROJUDGER Webhook Receiver Examples

Este directorio contiene ejemplos completos de cómo implementar receptores de webhooks de ROJUDGER en diferentes lenguajes de programación.

---

## 📂 Contenido

- **`webhook_receiver.js`** - Receptor en Node.js + Express
- **`webhook_receiver.py`** - Receptor en Python + Flask

---

## 🚀 Inicio Rápido

### Node.js (Express)

```bash
# 1. Instalar dependencias
npm install express

# 2. Configurar secreto (opcional)
export WEBHOOK_SECRET="tu-secreto-aqui"

# 3. Ejecutar
node examples/webhook_receiver.js

# El servidor estará en http://localhost:9000
```

### Python (Flask)

```bash
# 1. Instalar dependencias
pip install flask

# 2. Configurar secreto (opcional)
export WEBHOOK_SECRET="tu-secreto-aqui"

# 3. Ejecutar
python examples/webhook_receiver.py

# El servidor estará en http://localhost:9000
```

---

## 🧪 Testing

### 1. Iniciar el receptor

```bash
# Terminal 1
cd examples
node webhook_receiver.js
```

### 2. Enviar una submission con webhook

```bash
# Terminal 2
curl -X POST http://localhost:8080/api/v1/submissions \
  -H "Content-Type: application/json" \
  -d '{
    "language_id": 71,
    "source_code": "print(\"Hello from webhook test!\")",
    "webhook_url": "http://localhost:9000/webhooks/rojudger"
  }'
```

### 3. Ver el webhook recibido

El receptor mostrará algo como:

```
============================================================
📨 Webhook recibido: 2024-01-15T10:30:05.123Z
============================================================
✅ Firma HMAC verificada
📋 Submission ID: abc-123-def-456
🏷️  Event: submission.completed
📊 Status: completed
🔢 Exit Code: 0
⏱️  Time: 0.123s
💾 Memory: 8192 KB
📤 Stdout:
Hello from webhook test!

🎉 Submission completada exitosamente
✅ Webhook procesado correctamente
```

---

## 🔒 Seguridad (HMAC)

### ¿Por qué HMAC?

Las firmas HMAC permiten verificar que el webhook realmente viene de ROJUDGER y no ha sido modificado.

### Configuración

**1. En el worker de ROJUDGER:**

```bash
export WEBHOOK_SECRET="mi-secreto-super-seguro-123"
./worker
```

**2. En tu receptor:**

```bash
export WEBHOOK_SECRET="mi-secreto-super-seguro-123"
node webhook_receiver.js
```

**⚠️ Importante:** El secreto debe ser idéntico en ambos lados.

### Generar un secreto seguro

```bash
# Linux/Mac
openssl rand -hex 32

# Output: a1b2c3d4e5f6... (usar esto como WEBHOOK_SECRET)
```

---

## 📊 Estructura del Payload

Cada webhook incluye:

```json
{
  "event": "submission.completed",
  "timestamp": "2024-01-15T10:30:05.123Z",
  "submission": {
    "id": "abc-123-def-456",
    "language_id": 71,
    "source_code": "print(\"Hello!\")",
    "status": "completed",
    "stdout": "Hello!\n",
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

### Headers

```http
POST /webhooks/rojudger HTTP/1.1
Content-Type: application/json
User-Agent: ROJUDGER-Webhook/1.0
X-Rojudger-Event: submission.completed
X-Rojudger-Submission-Id: abc-123-def-456
X-Rojudger-Delivery: 1705318205
X-Rojudger-Signature: a1b2c3d4e5f6... (HMAC-SHA256)
```

---

## 🔧 Personalización

### Modificar el puerto

```bash
# Node.js
PORT=8000 node webhook_receiver.js

# Python
PORT=8000 python webhook_receiver.py
```

### Agregar lógica personalizada

Edita las funciones `handle*Submission()` en los archivos:

```javascript
// Node.js
function handleCompletedSubmission(submission) {
  console.log('🎉 Submission completada');
  
  // Tu lógica aquí:
  // - Guardar en DB
  // - Enviar email
  // - Actualizar leaderboard
  // - etc.
}
```

```python
# Python
def handle_completed_submission(submission: dict):
    print('🎉 Submission completada')
    
    # Tu lógica aquí:
    # - Guardar en DB
    # - Enviar email
    # - Actualizar leaderboard
    # - etc.
```

---

## 🌐 Testing con URLs Públicas

### Opción 1: webhook.site

1. Ir a https://webhook.site
2. Copiar la URL única
3. Usar en ROJUDGER:

```bash
curl -X POST http://localhost:8080/api/v1/submissions \
  -d '{
    "language_id": 71,
    "source_code": "print(\"test\")",
    "webhook_url": "https://webhook.site/tu-uuid-unico"
  }'
```

4. Ver el webhook en tiempo real en webhook.site

### Opción 2: ngrok

```bash
# Terminal 1: Receptor local
node webhook_receiver.js

# Terminal 2: Exponer con ngrok
ngrok http 9000

# Terminal 3: Usar la URL de ngrok
curl -X POST http://localhost:8080/api/v1/submissions \
  -d '{
    "webhook_url": "https://abc123.ngrok.io/webhooks/rojudger",
    ...
  }'
```

---

## 📚 Integración con Otros Servicios

### Discord

```javascript
const axios = require('axios');

async function sendToDiscord(submission) {
  const DISCORD_WEBHOOK = process.env.DISCORD_WEBHOOK_URL;
  
  const embed = {
    title: `Submission ${submission.status}`,
    color: submission.status === 'completed' ? 0x00FF00 : 0xFF0000,
    fields: [
      { name: 'ID', value: submission.id },
      { name: 'Exit Code', value: submission.exit_code.toString() },
      { name: 'Time', value: `${submission.time}s` }
    ]
  };
  
  await axios.post(DISCORD_WEBHOOK, { embeds: [embed] });
}
```

### Slack

```javascript
const axios = require('axios');

async function sendToSlack(submission) {
  const SLACK_WEBHOOK = process.env.SLACK_WEBHOOK_URL;
  
  const message = {
    text: `Submission ${submission.id}: ${submission.status}`,
    attachments: [{
      color: submission.status === 'completed' ? 'good' : 'danger',
      fields: [
        { title: 'Exit Code', value: submission.exit_code, short: true },
        { title: 'Time', value: `${submission.time}s`, short: true }
      ]
    }]
  };
  
  await axios.post(SLACK_WEBHOOK, message);
}
```

---

## 🐛 Troubleshooting

### El webhook no llega

1. **Verificar que el worker esté corriendo:**
   ```bash
   ps aux | grep worker
   ```

2. **Verificar que la submission terminó:**
   ```bash
   curl http://localhost:8080/api/v1/submissions/abc-123
   # status debe ser "completed", "error" o "timeout"
   ```

3. **Verificar logs del worker:**
   ```bash
   # Buscar líneas como:
   # "Sending webhook for submission abc-123 to http://..."
   ```

### Error: Invalid signature

1. **Secretos diferentes:**
   ```bash
   # Worker
   echo $WEBHOOK_SECRET
   
   # Receptor
   echo $WEBHOOK_SECRET
   # Deben ser idénticos
   ```

2. **Body parseado antes de verificar:**
   ```javascript
   // ❌ INCORRECTO
   app.use(express.json()); // Para todas las rutas
   
   // ✅ CORRECTO
   app.use('/webhooks', express.raw({type: 'application/json'}));
   ```

### Timeout

1. **Responder rápidamente:**
   ```javascript
   app.post('/webhook', (req, res) => {
     // ✅ Responder primero
     res.json({ status: 'received' });
     
     // ✅ Procesar después
     processWebhook(req.body);
   });
   ```

2. **Verificar firewall:**
   ```bash
   telnet your-server.com 9000
   ```

---

## 📖 Documentación Completa

Ver [docs/WEBHOOKS.md](../docs/WEBHOOKS.md) para:

- Arquitectura detallada
- Guía de seguridad completa
- Mejores prácticas
- Casos de uso avanzados
- Troubleshooting exhaustivo

---

## 💡 Tips

1. **Siempre usa HMAC en producción** para verificar autenticidad
2. **Responde rápido** (< 5 segundos) para evitar timeouts
3. **Maneja duplicados** (el webhook puede reenviarse)
4. **Loggea todo** para debugging
5. **Rate limiting** para evitar abuse
6. **Usa HTTPS** en producción

---

## 🤝 Contribuir

¿Tienes un ejemplo en otro lenguaje? ¡Contribuye!

Lenguajes bienvenidos:
- Go
- Ruby
- PHP
- Java
- Rust
- etc.

---

**¡Happy webhook coding! 🚀**