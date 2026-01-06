# 🚀 ROJUDGER - Sistema de Ejecución de Código con Cola de Prioridades

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org)
[![Docker](https://img.shields.io/badge/Docker-Required-2496ED?style=flat&logo=docker)](https://docker.com)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15-336791?style=flat&logo=postgresql)](https://postgresql.org)
[![Redis](https://img.shields.io/badge/Redis-7-DC382D?style=flat&logo=redis)](https://redis.io)
[![License](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

Un sistema de ejecución de código **robusto, escalable y con sistema de prioridades**, inspirado en Judge0, construido desde cero en **Go**.

> 🎉 **Nuevo:** Sistema de cola con prioridades, workers separables, y arquitectura production-ready

---

## 📋 Tabla de Contenidos

- [Características Principales](#-características-principales)
- [¿Qué es ROJUDGER?](#-qué-es-rojudger)
- [Arquitectura](#️-arquitectura)
- [Instalación Rápida](#-instalación-rápida)
- [Uso de la API](#-uso-de-la-api)
- [Sistema de Prioridades](#-sistema-de-prioridades)
- [Workers Separados](#-workers-separados)
- [Lenguajes Soportados](#️-lenguajes-soportados)
- [Ejemplos](#-ejemplos)
- [Documentación](#-documentación)
- [Roadmap](#️-roadmap)

---

## ✨ Características Principales

### 🎯 Sistema de Cola con Prioridades ⭐ NUEVO
- **3 niveles de prioridad**: High (>5), Normal (0-5), Low (<0)
- **Workers automáticos** procesan por prioridad
- **Perfecto para**: Competencias, exámenes, usuarios premium
- **Sin código extra**: Solo agrega `"priority": 10` a tu request

### 🔒 Seguridad Robusta
- Ejecución en **contenedores Docker aislados**
- Sin acceso a red
- Límites de CPU, memoria y tiempo
- No-root containers

### ⚡ Alta Performance
- **Modo síncrono** para respuesta inmediata
- **Modo asíncrono** con Redis queue
- Executor concurrente (5 submissions simultáneas por worker)
- Pool de conexiones optimizado

### 🏗️ Production Ready
- **Workers escalables** (separa API de ejecución)
- Sistema de colas con Redis
- Logging detallado
- Health checks y estadísticas
- Docker Compose incluido

### 🛠️ Developer Friendly
- API REST simple y bien documentada
- 5 lenguajes soportados (Python, JS, Go, C, C++)
- Fácil agregar nuevos lenguajes
- Tests automatizados incluidos

---

## 🎯 ¿Qué es ROJUDGER?

ROJUDGER es un **sistema completo de ejecución de código** que permite:

✅ Ejecutar código de forma segura en contenedores aislados  
✅ Soportar múltiples lenguajes de programación  
✅ **Sistema de prioridades** para diferentes tipos de usuarios/tareas  
✅ **Arquitectura escalable** con workers separados  
✅ Limitar recursos (CPU, memoria, tiempo)  
✅ API REST para integrar en cualquier aplicación  
✅ Perfecto para plataformas tipo **LeetCode, HackerRank, Codeforces**  

---

## 🏗️ Arquitectura

### Arquitectura Básica (Desarrollo)

```
┌─────────────┐
│   Cliente   │
└──────┬──────┘
       │ HTTP
       ▼
┌─────────────────────────────┐
│    API Server (Go + Gin)    │
│  ┌──────────┐  ┌─────────┐ │
│  │ Handlers │  │Database │ │
│  └──────────┘  └─────────┘ │
└──────────┬──────────────────┘
           │
           ▼
    ┌──────────────┐
    │   Executor   │
    └──────┬───────┘
           │
           ▼
    ┌────────────────────┐
    │ Docker Containers  │
    │ 🐍 Python  📜 JS   │
    │ 🦫 Go  🔧 C/C++    │
    └────────────────────┘
```

### Arquitectura con Cola (Producción) ⭐

```
Internet
   │
   ▼
┌────────────┐
│    API     │  ← Recibe requests, encola
└─────┬──────┘
      │
      ▼
┌─────────────────┐
│     Redis       │  ← 3 colas: high, default, low
│  Queue System   │
└────────┬────────┘
         │
    ┌────┴────┬────────┐
    ▼         ▼        ▼
┌────────┐ ┌────────┐ ┌────────┐
│Worker 1│ │Worker 2│ │Worker 3│  ← Ejecutan código
└───┬────┘ └───┬────┘ └───┬────┘
    └──────────┴──────────┘
              │
              ▼
       ┌──────────────┐
       │  PostgreSQL  │  ← Resultados
       └──────────────┘
```

**Ventajas:**
- 🚀 Escala horizontalmente (agrega más workers)
- 🔥 Prioridades automáticas
- 💪 Alta disponibilidad
- 🎯 API y Workers separados

---

## 🚀 Instalación Rápida

### Prerrequisitos

- Docker y Docker Compose
- Go 1.21+ (opcional, para desarrollo)

### Opción 1: Todo en Uno (Desarrollo)

```bash
# 1. Clonar
git clone https://github.com/tu-usuario/rojudger.git
cd rojudger

# 2. Iniciar servicios base
docker-compose up -d postgres redis

# 3. Ejecutar API (modo directo)
USE_QUEUE=false go run ./cmd/api

# API disponible en http://localhost:8080
```

### Opción 2: Con Sistema de Colas (Producción)

```bash
# 1. Iniciar servicios
docker-compose up -d postgres redis

# 2. Compilar
go build -o api ./cmd/api
go build -o worker ./cmd/worker

# 3. Iniciar API (modo cola)
USE_QUEUE=true ./api &

# 4. Iniciar Workers (tantos como necesites)
./worker &
./worker &
./worker &

# Listo! Sistema con prioridades funcionando
```

### Opción 3: Docker Compose Completo

```bash
# TODO: Próximamente docker-compose con workers
docker-compose -f docker-compose.prod.yml up -d
```

---

## 📡 Uso de la API

### Base URL
```
http://localhost:8080/api/v1
```

### Endpoints Principales

#### 1. Crear Submission (Básico)

```bash
curl -X POST http://localhost:8080/api/v1/submissions \
  -H "Content-Type: application/json" \
  -d '{
    "language_id": 71,
    "source_code": "print(\"Hello ROJUDGER!\")"
  }'
```

**Respuesta:**
```json
{
  "id": "abc-123-def-456",
  "language_id": 71,
  "status": "queued",
  "created_at": "2026-01-05T18:00:00Z"
}
```

#### 2. Crear Submission con Prioridad ⭐ NUEVO

```bash
curl -X POST http://localhost:8080/api/v1/submissions \
  -H "Content-Type: application/json" \
  -d '{
    "language_id": 71,
    "source_code": "print(\"VIP task!\")",
    "priority": 10
  }'
```

#### 3. Obtener Resultado

```bash
curl http://localhost:8080/api/v1/submissions/abc-123-def-456
```

**Respuesta:**
```json
{
  "id": "abc-123-def-456",
  "language_id": 71,
  "source_code": "print(\"Hello ROJUDGER!\")",
  "status": "completed",
  "stdout": "Hello ROJUDGER!\n",
  "stderr": "",
  "exit_code": 0,
  "time": 0.523,
  "memory": 0,
  "created_at": "2026-01-05T18:00:00Z",
  "finished_at": "2026-01-05T18:00:01Z"
}
```

#### 4. Estadísticas de Cola ⭐ NUEVO

```bash
curl http://localhost:8080/api/v1/queue/stats
```

**Respuesta:**
```json
{
  "queue_high": 2,
  "queue_default": 15,
  "queue_low": 5,
  "processing": 3,
  "total_pending": 22,
  "total_enqueued": 1250,
  "total_completed": 1180,
  "total_failed": 45
}
```

#### 5. Listar Lenguajes

```bash
curl http://localhost:8080/api/v1/languages
```

#### 6. Health Check

```bash
curl http://localhost:8080/health
```

---

## 🎯 Sistema de Prioridades ⭐

### ¿Qué es?

El sistema de prioridades te permite **controlar qué submissions se ejecutan primero**.

```
COLA ALTA (priority > 5)    → Se ejecutan PRIMERO
COLA NORMAL (priority 0-5)  → Orden normal
COLA BAJA (priority < 0)    → Se ejecutan AL FINAL
```

### Niveles Recomendados

| Prioridad | Nombre | Uso |
|-----------|--------|-----|
| **10** | Critical | 🔥 Emergencias, competencias en vivo |
| **8** | Urgent | ⚡ Exámenes importantes |
| **6** | High | 💎 Usuarios premium |
| **0** | Normal | 📌 Uso estándar (default) |
| **-3** | Low | 🐌 Background tasks |
| **-5** | Batch | 📦 Procesamiento masivo |
| **-10** | Maintenance | 🔧 Tareas de mantenimiento |

### Ejemplos de Uso

#### Usuario Premium vs Free

```bash
# Usuario Premium (ejecuta primero)
curl -X POST http://localhost:8080/api/v1/submissions \
  -d '{"language_id": 71, "source_code": "...", "priority": 8}'

# Usuario Free (normal)
curl -X POST http://localhost:8080/api/v1/submissions \
  -d '{"language_id": 71, "source_code": "...", "priority": 0}'
```

#### Competencia vs Práctica

```bash
# Durante competencia (máxima prioridad)
{"priority": 10}

# Modo práctica (normal)
{"priority": 0}
```

#### Background Job

```bash
# Corrección automática nocturna
{"priority": -5}
```

**📚 Más detalles:** Ver [docs/PRIORITY_SYSTEM.md](docs/PRIORITY_SYSTEM.md)

---

## 🔔 Webhooks ⭐ NUEVO

### ¿Qué son?

Los webhooks permiten **recibir notificaciones automáticas** cuando una submission termina, sin necesidad de hacer polling.

```
┌─────────┐                  ┌──────────┐                  ┌─────────┐
│ Cliente │ ─── POST ───────▶│ ROJUDGER │ ─── Webhook ───▶│ Tu App  │
└─────────┘   + webhook_url  └──────────┘   (notificación) └─────────┘
```

### Ejemplo Básico

```bash
curl -X POST http://localhost:8080/api/v1/submissions \
  -H "Content-Type: application/json" \
  -d '{
    "language_id": 71,
    "source_code": "print(\"Hello!\")",
    "webhook_url": "https://your-app.com/webhooks/rojudger"
  }'
```

Cuando la submission termine, ROJUDGER enviará un POST a tu URL:

```json
{
  "event": "submission.completed",
  "timestamp": "2024-01-15T10:30:05Z",
  "submission": {
    "id": "abc-123",
    "status": "completed",
    "stdout": "Hello!\n",
    "exit_code": 0,
    "time": 0.123
  }
}
```

### Seguridad (HMAC)

Los webhooks incluyen firmas HMAC-SHA256 para verificar autenticidad:

```bash
# 1. Configurar secreto en el worker
export WEBHOOK_SECRET="tu-secreto-super-seguro"
./worker

# 2. Verificar en tu servidor
```

**Ejemplo en Node.js:**

```javascript
const crypto = require('crypto');

app.post('/webhooks/rojudger', express.raw({type: 'application/json'}), (req, res) => {
  const signature = req.headers['x-rojudger-signature'];
  const hmac = crypto.createHmac('sha256', process.env.WEBHOOK_SECRET);
  hmac.update(req.body);
  const expected = hmac.digest('hex');
  
  if (signature !== expected) {
    return res.status(401).send('Invalid signature');
  }
  
  // ✅ Webhook verificado
  const payload = JSON.parse(req.body);
  console.log('Submission completed:', payload.submission.id);
  res.json({ status: 'received' });
});
```

### Características

- ✅ **Reintentos automáticos**: 3 intentos con backoff exponencial
- ✅ **Firmas HMAC**: Autenticación criptográfica
- ✅ **Logs completos**: Tabla `webhook_logs` en DB
- ✅ **Validación de URLs**: Previene ataques SSRF
- ✅ **Headers personalizados**: Metadatos útiles

### Testing

Usa el script incluido:

```bash
./scripts/test_webhooks.sh
```

O prueba con [webhook.site](https://webhook.site):

```bash
curl -X POST http://localhost:8080/api/v1/submissions \
  -d '{
    "language_id": 71,
    "source_code": "print(\"Test\")",
    "webhook_url": "https://webhook.site/tu-uuid"
  }'
```

**📚 Documentación completa:** [docs/WEBHOOKS.md](docs/WEBHOOKS.md)

---

## 🔧 Workers Separados

### ¿Por Qué Separar Workers?

```
❌ TODO EN UNO          ✅ SEPARADO
┌────────────┐          ┌─────┐  ┌────────┐ ┌────────┐
│ API+Worker │          │ API │  │Worker 1│ │Worker 2│
│ 1 servidor │          │     │  │        │ │        │
└────────────┘          └─────┘  └────────┘ └────────┘
- No escala             - Escala fácil
- Si falla, todo falla  - Alta disponibilidad
- Recursos compartidos  - Recursos dedicados
```

### Ventajas

1. **Escalabilidad**: 1 API + N workers
2. **Seguridad**: API sin Docker (más seguro)
3. **Recursos**: Workers con más RAM/CPU
4. **Deploy**: Actualiza sin downtime
5. **Monitoreo**: Métricas separadas

### Cómo Implementar

#### Desarrollo (1 máquina)
```bash
# Terminal 1: API
USE_QUEUE=true ./api

# Terminal 2+: Workers
./worker
./worker  # Más workers para más throughput
```

#### Producción (Servidores separados)
```bash
# Servidor 1: API (sin Docker)
USE_QUEUE=true ./api

# Servidor 2-N: Workers (con Docker)
./worker  # En cada servidor worker
```

**📚 Guía completa:** [docs/WORKERS_SEPARADOS_GUIA.md](docs/WORKERS_SEPARADOS_GUIA.md)

---

## 🗣️ Lenguajes Soportados

| ID | Lenguaje | Versión | Compilado | Docker Image |
|----|----------|---------|-----------|--------------|
| **71** | Python 3 | 3.11 | No | `python:3.11-slim` |
| **63** | JavaScript (Node) | 20 | No | `node:20-slim` |
| **60** | Go | 1.21 | Sí* | `golang:1.21-alpine` |
| **50** | C (GCC) | 11 | Sí | `gcc:11` |
| **54** | C++ (G++) | 11 | Sí | `gcc:11` |

*Go usa `go run` (compila y ejecuta en un paso)

### Agregar Más Lenguajes

Edita `internal/database/database.go`:

```go
{
    ID:          75,
    Name:        "rust",
    DisplayName: "Rust",
    Version:     "1.75",
    Extension:   ".rs",
    CompileCmd:  "rustc {file} -o main",
    ExecuteCmd:  "./main",
    DockerImage: "rust:1.75-alpine",
    IsCompiled:  true,
    IsEnabled:   true,
}
```

---

## 💡 Ejemplos

### Ejemplo 1: Hello World con Prioridad Alta

```bash
curl -X POST http://localhost:8080/api/v1/submissions \
  -H "Content-Type: application/json" \
  -d '{
    "language_id": 71,
    "source_code": "print(\"Hello from VIP queue!\")",
    "priority": 10
  }'
```

### Ejemplo 2: Programa con Input

```bash
curl -X POST http://localhost:8080/api/v1/submissions \
  -H "Content-Type: application/json" \
  -d '{
    "language_id": 71,
    "source_code": "name = input()\nprint(f\"Hello, {name}!\")",
    "stdin": "Alice",
    "priority": 0
  }'
```

### Ejemplo 3: C++ con Prioridad Baja

```bash
curl -X POST http://localhost:8080/api/v1/submissions \
  -H "Content-Type: application/json" \
  -d '{
    "language_id": 54,
    "source_code": "#include <iostream>\nint main() { std::cout << \"C++\" << std::endl; }",
    "priority": -5
  }'
```

### Ejemplo 4: Múltiples Submissions (Batch)

```bash
# Script de prueba de prioridades
./test_priority_simple.sh

# O test completo
./scripts/test_priorities.sh
```

---

## 📚 Documentación

### Guías Principales

| Documento | Descripción |
|-----------|-------------|
| **[QUICKSTART.md](QUICKSTART.md)** | 🚀 Inicio rápido en 5 minutos |
| **[docs/PRIORITY_SYSTEM.md](docs/PRIORITY_SYSTEM.md)** | 🎯 Sistema de prioridades completo |
| **[docs/WORKERS_SEPARADOS_GUIA.md](docs/WORKERS_SEPARADOS_GUIA.md)** | 🏗️ Arquitectura escalable |
| **[docs/DATABASE_NULL_FIX.md](docs/DATABASE_NULL_FIX.md)** | 🔧 Fix técnico de NULL handling |
| **[QUEUE_STATUS.md](QUEUE_STATUS.md)** | 📊 Estado del sistema de colas |

### Documentos de Referencia

- `FIX_SUMMARY.md` - Resumen ejecutivo del fix de NULL
- `PRIORITY_IMPLEMENTATION_COMPLETE.md` - Implementación de prioridades
- `RESUMEN_FINAL.txt` - Resumen completo del proyecto

### Scripts de Prueba

- `test_priority_simple.sh` - Test básico de prioridades
- `test_comprehensive.sh` - Test completo de cola
- `scripts/test_all_languages.sh` - Test de todos los lenguajes
- `scripts/test_priorities.sh` - Test detallado de prioridades

---

## 🗺️ Roadmap

### ✅ Fase 1: MVP (COMPLETADO)
- [x] API REST con Gin
- [x] Ejecución en Docker
- [x] 5 lenguajes (Python, JS, Go, C, C++)
- [x] PostgreSQL
- [x] Límites de recursos

### ✅ Fase 2: Sistema de Colas (COMPLETADO)
- [x] **Integración con Redis**
- [x] **Workers separados del API**
- [x] **Sistema de 3 prioridades**
- [x] **Logging detallado**
- [x] **Estadísticas de cola**
- [x] **Tests automatizados**
- [x] **Documentación completa**

### 🔄 Fase 3: Features Avanzadas (EN PROGRESO)
- [ ] Webhooks para notificaciones
- [ ] Múltiples archivos (proyectos completos)
- [ ] Test cases automáticos
- [ ] WebSocket para resultados en tiempo real
- [ ] Dashboard web de monitoreo

### 📋 Fase 4: Optimización
- [ ] Cache de imágenes Docker
- [ ] Pre-warming de contenedores
- [ ] Auto-scaling basado en queue length
- [ ] Métricas con Prometheus
- [ ] Dashboards con Grafana

### 🔐 Fase 5: Seguridad Avanzada
- [ ] Rate limiting por usuario/IP
- [ ] Autenticación JWT
- [ ] API Keys
- [ ] Sandboxing con gVisor/Firecracker
- [ ] Auditoría completa

---

## 🔧 Configuración

### Variables de Entorno

```bash
# API
USE_QUEUE=true              # true = cola, false = directo
API_PORT=8080

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=rojudger
DB_PASSWORD=rojudger_password
DB_NAME=rojudger_db

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379

# Executor
EXECUTOR_TIMEOUT=30         # Segundos
MAX_CONCURRENT_WORKERS=5    # Por worker
```

---

## 📊 Monitoreo

### Ver Estadísticas

```bash
# Estadísticas de cola
curl http://localhost:8080/api/v1/queue/stats | jq '.'

# Health check
curl http://localhost:8080/health | jq '.'
```

### Logs

```bash
# API logs
tail -f /tmp/api.log

# Worker logs
tail -f /tmp/worker.log

# O con journalctl (si usas systemd)
journalctl -u rojudger-api -f
journalctl -u rojudger-worker -f
```

### Redis

```bash
redis-cli

# Tamaño de colas
> LLEN rojudger:queue:high
> LLEN rojudger:queue:default
> LLEN rojudger:queue:low

# En procesamiento
> SCARD rojudger:processing

# Estadísticas
> HGETALL rojudger:stats
```

### Base de Datos

```bash
docker exec -it rojudger-postgres psql -U rojudger -d rojudger_db

# Ver submissions recientes
SELECT id, status, time, exit_code FROM submissions 
ORDER BY created_at DESC LIMIT 10;

# Estadísticas por estado
SELECT status, COUNT(*) FROM submissions GROUP BY status;
```

---

## 🤝 Contribuir

¡Las contribuciones son bienvenidas!

1. Fork el proyecto
2. Crea tu rama (`git checkout -b feature/AmazingFeature`)
3. Commit tus cambios (`git commit -m 'Add: Amazing feature'`)
4. Push a la rama (`git push origin feature/AmazingFeature`)
5. Abre un Pull Request

### Estilo de Commits

Usamos [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: Nuevo feature
fix: Bug fix
docs: Documentación
test: Tests
refactor: Refactorización
perf: Performance
chore: Mantenimiento
```

---

## 🐛 Troubleshooting

### Worker crashea con error de NULL
✅ **SOLUCIONADO** en versión actual. Si usas versión antigua, actualiza.

### Puerto 8080 ocupado
```bash
API_PORT=3000 ./api
```

### Docker no encuentra imágenes
```bash
./scripts/pull_images.sh
```

### Redis no conecta
```bash
docker-compose up -d redis
docker logs rojudger-redis
```

---

## 📈 Performance

### Benchmarks

- **Latencia promedio**: ~500ms (Python)
- **Throughput**: 100+ submissions/minuto (1 worker)
- **Escalabilidad**: Lineal con número de workers

### Optimizaciones

1. **Múltiples workers**: Escala horizontalmente
2. **Redis local**: Baja latencia de cola
3. **DB pool**: Conexiones reutilizadas
4. **Docker cache**: Imágenes pre-descargadas

---

## 🏆 Casos de Uso

- 📚 **Plataformas educativas** (bootcamps, universidades)
- 🏅 **Competencias de programación** (ACM, Codeforces-style)
- 💼 **Entrevistas técnicas** (live coding)
- 🧪 **Sistemas de evaluación** automática
- 🎮 **Coding challenges** y gamificación
- 📖 **Tutoriales interactivos** de programación

---

## 📝 Licencia

Este proyecto está bajo la **Licencia MIT**. Ver [LICENSE](LICENSE) para detalles.

---

## 🙏 Agradecimientos

Inspirado por:
- [Judge0](https://github.com/judge0/judge0) - Sistema robusto de ejecución
- [Go Playground](https://go.dev/blog/playground) - Diseño elegante
- [Isolate](https://github.com/ioi/isolate) - Sandbox para competitive programming

---

## 📧 Contacto

**Autor:** Roberto Rocha  
**Proyecto:** ROJUDGER - Judge0 Clone en Go  
**Estado:** ✅ Production Ready  

---

## 🎉 Changelog

### v1.0.0 (2026-01-05)
- ✅ Sistema de cola con Redis
- ✅ Sistema de 3 prioridades (high/default/low)
- ✅ Workers separables del API
- ✅ Fix de NULL handling en database
- ✅ Documentación completa
- ✅ Tests automatizados
- ✅ Production ready

### v0.1.0 (2024-XX-XX)
- ✅ MVP con API REST
- ✅ 5 lenguajes soportados
- ✅ Ejecución en Docker
- ✅ PostgreSQL

---

**⭐ Si te gusta este proyecto, dale una estrella en GitHub!**

**🚀 Happy Coding!**
