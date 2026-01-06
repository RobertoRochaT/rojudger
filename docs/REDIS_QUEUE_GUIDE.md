# ROJUDGER - Guía de Colas con Redis

## 🎯 ¿Qué Lograste?

Has implementado un sistema de **colas asíncronas con Redis** en ROJUDGER que permite:

✅ **Escalabilidad**: Múltiples workers procesando código en paralelo  
✅ **Asincronía**: API responde instantáneamente sin esperar ejecución  
✅ **Confiabilidad**: Si un worker falla, otro puede retomar  
✅ **Priorización**: Colas de alta/normal/baja prioridad  
✅ **Monitoreo**: Estadísticas en tiempo real de la cola  

---

## 🏗️ Arquitectura

```
┌──────────┐
│  Cliente │
└────┬─────┘
     │ POST /submissions
     ↓
┌──────────────┐
│  API Server  │ ← Encola en Redis
└──────┬───────┘
       │
       ↓
┌──────────────┐
│ Redis Queue  │ ← Lista de trabajos
└──────┬───────┘
       │
       ↓
┌──────────────┐
│  Worker(s)   │ ← Procesan código
│  (1...N)     │
└──────┬───────┘
       │
       ↓
┌──────────────┐
│  PostgreSQL  │ ← Guarda resultados
└──────────────┘
```

---

## 🚀 Cómo Usar

### Opción 1: Modo DIRECTO (sin cola, síncro)

El API ejecuta el código directamente y responde cuando termina.

```bash
# Ejecutar API en modo directo
go run cmd/api/main.go

# O con variable de entorno
USE_QUEUE=false go run cmd/api/main.go
```

**Características:**
- ⚡ Respuesta inmediata con resultados
- 🔒 Limitado a 5 ejecuciones concurrentes
- 🎯 Útil para desarrollo/testing

### Opción 2: Modo QUEUE (con cola, async)

El API encola el trabajo y los workers lo procesan.

```bash
# Terminal 1: API en modo queue
USE_QUEUE=true go run cmd/api/main.go

# Terminal 2+: Workers (puedes abrir varios)
go run cmd/worker/main.go
go run cmd/worker/main.go  # Worker adicional
go run cmd/worker/main.go  # Otro más...
```

**Características:**
- 🚀 API responde instantáneamente
- 📈 Escala horizontalmente (más workers = más capacidad)
- 🔄 Workers procesan en background
- 💪 Robusto ante fallos

---

## 📊 Endpoints del API

### POST /api/v1/submissions

**Modo Directo (USE_QUEUE=false):**
```bash
curl -X POST http://localhost:8080/api/v1/submissions \
  -H "Content-Type: application/json" \
  -d '{
    "language_id": 71,
    "source_code": "print(123)"
  }'

# Respuesta: Resultado completo (espera ~2 segundos)
```

**Modo Queue (USE_QUEUE=true):**
```bash
# Enviar y recibir ID inmediatamente
curl -X POST http://localhost:8080/api/v1/submissions \
  -H "Content-Type: application/json" \
  -d '{
    "language_id": 71,
    "source_code": "print(123)"
  }'

# Respuesta inmediata:
{
  "id": "abc-123",
  "status": "queued",
  ...
}

# Consultar resultado después
curl http://localhost:8080/api/v1/submissions/abc-123
```

**Modo Queue con wait=true (híbrido):**
```bash
curl -X POST "http://localhost:8080/api/v1/submissions?wait=true" \
  -H "Content-Type: application/json" \
  -d '{"language_id": 71, "source_code": "print(123)"}'

# Se encola pero espera hasta 30s por el resultado
```

### GET /api/v1/queue/stats (solo modo queue)

```bash
curl http://localhost:8080/api/v1/queue/stats
```

**Respuesta:**
```json
{
  "queue_high": 0,
  "queue_default": 5,
  "queue_low": 2,
  "processing": 3,
  "total_pending": 7,
  "total_enqueued": "1250",
  "total_completed": "1200",
  "total_failed": "5"
}
```

---

## ⚙️ Configuración

### Variables de Entorno

Añade a tu `.env`:

```bash
# Modo de operación
USE_QUEUE=true          # true=queue, false=direct

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# Workers
EXECUTOR_MAX_CONCURRENT=5  # Workers por proceso
```

---

## 🔄 Flujo Completo de Ejemplo

```bash
# 1. Iniciar servicios base
docker-compose up -d postgres redis

# 2. Iniciar API en modo queue
USE_QUEUE=true go run cmd/api/main.go &

# 3. Iniciar 3 workers
for i in {1..3}; do
  go run cmd/worker/main.go &
done

# 4. Enviar código
SUBMISSION_ID=$(curl -s -X POST http://localhost:8080/api/v1/submissions \
  -H "Content-Type: application/json" \
  -d '{
    "language_id": 71,
    "source_code": "import time\ntime.sleep(2)\nprint(\"Done!\")"
  }' | jq -r '.id')

echo "Submission ID: $SUBMISSION_ID"

# 5. Ver stats
curl -s http://localhost:8080/api/v1/queue/stats | jq

# 6. Esperar y ver resultado
sleep 5
curl -s http://localhost:8080/api/v1/submissions/$SUBMISSION_ID | jq
```

---

## 📈 Escalamiento

### Escalar Verticalmente (Más workers por máquina)

Edita `.env`:
```bash
EXECUTOR_MAX_CONCURRENT=10  # Cada worker process ejecuta 10 concurrentes
```

### Escalar Horizontalmente (Más máquinas)

```bash
# Servidor 1: API + Redis + PostgreSQL
USE_QUEUE=true go run cmd/api/main.go

# Servidor 2-N: Solo workers
DB_HOST=servidor1 REDIS_HOST=servidor1 go run cmd/worker/main.go
```

---

## 🔍 Monitoreo

### Ver Cola en Redis

```bash
# Conectar a Redis
docker exec -it rojudger-redis redis-cli

# Ver tamaño de colas
LLEN rojudger:queue:high
LLEN rojudger:queue:default
LLEN rojudger:queue:low

# Ver trabajos en procesamiento
SMEMBERS rojudger:processing

# Ver estadísticas
HGETALL rojudger:stats
```

### Ver Logs

```bash
# API
tail -f /tmp/rojudger-api.log

# Workers
# Los workers loguean en stdout, redirige a archivo si quieres:
go run cmd/worker/main.go > /tmp/worker1.log 2>&1 &
tail -f /tmp/worker1.log
```

---

## 🛠️ Casos de Uso

### Desarrollo Local
```bash
# Modo directo, más simple
USE_QUEUE=false go run cmd/api/main.go
```

### Producción Pequeña
```bash
# 1 API + 2 workers
USE_QUEUE=true go run cmd/api/main.go &
go run cmd/worker/main.go &
go run cmd/worker/main.go &
```

### Producción Grande
```bash
# 3 APIs (load balanced) + 10 workers distribuidos
# Servidor 1:
USE_QUEUE=true ./rojudger-api &

# Servidores 2-5 (workers):
for i in {1..10}; do
  ./rojudger-worker &
done
```

---

## 🧪 Testing

```bash
# Test de carga
for i in {1..100}; do
  curl -X POST http://localhost:8080/api/v1/submissions \
    -H "Content-Type: application/json" \
    -d '{"language_id": 71, "source_code": "print('$i')"}' &
done

# Ver estadísticas
watch -n 1 'curl -s http://localhost:8080/api/v1/queue/stats | jq'
```

---

## ❓ FAQ

### ¿Cuándo usar modo directo vs queue?

- **Directo**: Desarrollo, testing, tráfico bajo (<10 req/min)
- **Queue**: Producción, tráfico medio/alto (>10 req/min)

### ¿Cuántos workers necesito?

Regla general: `workers = peticiones_por_segundo * tiempo_promedio_ejecución`

Ejemplo: 10 req/s, 5 seg promedio → 50 workers

### ¿Qué pasa si un worker se cae?

El trabajo queda en Redis. Cuando vuelva a subir (o otro worker), lo retoma.

### ¿Puedo mezclar ambos modos?

Sí! Puedes tener instancias del API en modo directo Y en modo queue simultáneamente.

---

## 🎯 Próximos Pasos

1. **Dashboard**: Crear interfaz web para ver estadísticas
2. **Webhooks**: Notificar cuando un trabajo termine
3. **Prioridades**: Permitir al usuario elegir prioridad
4. **Reintentos**: Reintentar automáticamente trabajos fallidos
5. **TTL**: Limpiar trabajos viejos automáticamente

---

**¡Felicitaciones!** 🎉 Ahora tienes un sistema de ejecución de código con colas que puede escalar horizontalmente.
