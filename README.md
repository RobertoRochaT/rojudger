# 🚀 ROJUDGER - Code Execution System

Un sistema de ejecución de código robusto, escalable y seguro inspirado en Judge0 y el Go Playground, construido desde cero en **Go**.

## 📋 Tabla de Contenidos

- [¿Qué es ROJUDGER?](#qué-es-rojudger)
- [Características](#características)
- [Arquitectura](#arquitectura)
- [Tecnologías](#tecnologías)
- [Instalación Rápida](#instalación-rápida)
- [Uso de la API](#uso-de-la-api)
- [Lenguajes Soportados](#lenguajes-soportados)
- [Ejemplos](#ejemplos)
- [Roadmap](#roadmap)

## 🎯 ¿Qué es ROJUDGER?

ROJUDGER es un sistema de ejecución de código en línea que permite:
- ✅ Ejecutar código de forma segura en contenedores Docker aislados
- ✅ Soportar múltiples lenguajes de programación
- ✅ Limitar recursos (CPU, memoria, tiempo)
- ✅ API REST simple para integrar en cualquier aplicación
- ✅ Modo síncrono y asíncrono
- ✅ Perfecto para plataformas tipo LeetCode, HackerRank, etc.

## ✨ Características

### Seguridad
- 🔒 Ejecución en contenedores Docker aislados
- 🔒 Sin acceso a red
- 🔒 Límites de CPU y memoria configurables
- 🔒 Timeout automático
- 🔒 Sin privilegios (no-root containers)

### Performance
- ⚡ Executor concurrente con rate limiting
- ⚡ Pool de conexiones a base de datos
- ⚡ Modo síncrono para respuesta inmediata
- ⚡ Modo asíncrono con sistema de colas (próximamente)

### Developer Friendly
- 📚 API REST bien documentada
- 📚 Fácil de integrar
- 📚 Docker Compose para desarrollo local
- 📚 Logs detallados

## 🏗️ Arquitectura

```
┌─────────────┐
│   Cliente   │
│  (Browser)  │
└──────┬──────┘
       │ HTTP
       ▼
┌─────────────────────────────────┐
│      API Server (Gin)           │
│  ┌──────────┐  ┌──────────┐    │
│  │ Handlers │  │ Database │    │
│  └──────────┘  └──────────┘    │
└──────────┬──────────────────────┘
           │
           ▼
    ┌──────────────┐
    │   Executor   │
    │   (Docker)   │
    └──────────────┘
           │
           ▼
    ┌──────────────────────────┐
    │  Docker Containers       │
    │  ┌────┐ ┌────┐ ┌────┐   │
    │  │ Py │ │ JS │ │ Go │   │
    │  └────┘ └────┘ └────┘   │
    └──────────────────────────┘
```

### Componentes

1. **API Server**: Recibe peticiones HTTP y maneja la lógica de negocio
2. **Database**: PostgreSQL para almacenar submissions y resultados
3. **Executor**: Ejecuta código en contenedores Docker con límites de recursos
4. **Redis**: Para sistema de colas (próximamente)

## 🛠️ Tecnologías

- **Go 1.21+** - Lenguaje principal
- **Gin** - Framework HTTP
- **Docker** - Aislamiento de ejecución
- **PostgreSQL** - Base de datos
- **Redis** - Sistema de colas (futuro)

## 🚀 Instalación Rápida

### Prerrequisitos

- Docker y Docker Compose instalados
- Go 1.21+ (solo para desarrollo local sin Docker)

### Opción 1: Con Docker Compose (Recomendado)

```bash
# 1. Clonar el repositorio
git clone https://github.com/rocha/rojudger.git
cd rojudger

# 2. Copiar archivo de configuración
cp .env.example .env

# 3. Levantar todos los servicios
docker-compose up -d

# 4. Ver logs
docker-compose logs -f api

# 5. La API estará disponible en http://localhost:8080
```

### Opción 2: Desarrollo Local

```bash
# 1. Instalar dependencias
go mod download

# 2. Levantar PostgreSQL y Redis
docker-compose up -d postgres redis

# 3. Configurar variables de entorno
cp .env.example .env
# Editar .env si es necesario

# 4. Ejecutar servidor
go run cmd/api/main.go
```

## 📡 Uso de la API

### Base URL
```
http://localhost:8080/api/v1
```

### Endpoints

#### 1. Crear Submission (Modo Síncrono)

```bash
curl -X POST http://localhost:8080/api/v1/submissions?wait=true \
  -H "Content-Type: application/json" \
  -d '{
    "language_id": 71,
    "source_code": "print(\"Hello, ROJUDGER!\")",
    "stdin": ""
  }'
```

**Respuesta:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "language_id": 71,
  "source_code": "print(\"Hello, ROJUDGER!\")",
  "status": "completed",
  "stdout": "Hello, ROJUDGER!\n",
  "stderr": "",
  "exit_code": 0,
  "time": 0.523,
  "memory": 12800,
  "created_at": "2024-01-15T10:30:00Z",
  "finished_at": "2024-01-15T10:30:01Z"
}
```

#### 2. Crear Submission (Modo Asíncrono)

```bash
curl -X POST http://localhost:8080/api/v1/submissions \
  -H "Content-Type: application/json" \
  -d '{
    "language_id": 71,
    "source_code": "import time\ntime.sleep(2)\nprint(\"Done!\")"
  }'
```

**Respuesta:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "queued",
  "token": "550e8400-e29b-41d4-a716-446655440000"
}
```

#### 3. Obtener Resultado de Submission

```bash
curl http://localhost:8080/api/v1/submissions/550e8400-e29b-41d4-a716-446655440000
```

#### 4. Listar Lenguajes Disponibles

```bash
curl http://localhost:8080/api/v1/languages
```

**Respuesta:**
```json
[
  {
    "id": 71,
    "name": "python3",
    "display_name": "Python 3",
    "version": "3.11",
    "extension": ".py"
  },
  {
    "id": 63,
    "name": "javascript",
    "display_name": "JavaScript (Node.js)",
    "version": "20",
    "extension": ".js"
  }
]
```

#### 5. Health Check

```bash
curl http://localhost:8080/health
```

## 🗣️ Lenguajes Soportados

| ID  | Lenguaje         | Versión | Compilado |
|-----|------------------|---------|-----------|
| 71  | Python 3         | 3.11    | No        |
| 63  | JavaScript       | Node 20 | No        |
| 60  | Go               | 1.21    | Sí        |
| 50  | C (GCC)          | 11      | Sí        |
| 54  | C++ (G++)        | 11      | Sí        |

### Agregar Más Lenguajes

Para agregar un nuevo lenguaje, modifica `internal/database/database.go` en la función `SeedLanguages()`:

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

## 💡 Ejemplos

### Ejemplo 1: Hello World en Python

```bash
curl -X POST "http://localhost:8080/api/v1/submissions?wait=true" \
  -H "Content-Type: application/json" \
  -d '{
    "language_id": 71,
    "source_code": "name = input()\nprint(f\"Hello, {name}!\")",
    "stdin": "Alice"
  }'
```

### Ejemplo 2: Suma de dos números en JavaScript

```bash
curl -X POST "http://localhost:8080/api/v1/submissions?wait=true" \
  -H "Content-Type: application/json" \
  -d '{
    "language_id": 63,
    "source_code": "const readline = require(\"readline\");\nconst rl = readline.createInterface({input: process.stdin});\nlet numbers = [];\nrl.on(\"line\", (line) => numbers.push(parseInt(line)));\nrl.on(\"close\", () => console.log(numbers[0] + numbers[1]));",
    "stdin": "5\n10"
  }'
```

### Ejemplo 3: Programa en C++

```bash
curl -X POST "http://localhost:8080/api/v1/submissions?wait=true" \
  -H "Content-Type: application/json" \
  -d '{
    "language_id": 54,
    "source_code": "#include <iostream>\nusing namespace std;\nint main() {\n    int n;\n    cin >> n;\n    cout << \"Number: \" << n << endl;\n    return 0;\n}",
    "stdin": "42"
  }'
```

## 🔧 Configuración Avanzada

### Variables de Entorno

Edita `.env` para personalizar:

```bash
# Límites de ejecución
EXECUTOR_TIMEOUT=10s          # Timeout máximo
EXECUTOR_MEMORY_LIMIT=256m    # Memoria máxima
EXECUTOR_CPU_LIMIT=0.5        # 50% de un CPU
EXECUTOR_MAX_CONCURRENT=5     # Ejecuciones concurrentes máximas
```

## 🗺️ Roadmap

### Fase 1: MVP ✅
- [x] API REST básica
- [x] Ejecución en Docker
- [x] Soporte para Python, JavaScript, Go, C, C++
- [x] Límites de recursos
- [x] Base de datos PostgreSQL

### Fase 2: Cola de Trabajos 🔄
- [ ] Integrar Redis para colas
- [ ] Workers separados del API
- [ ] Sistema de prioridades
- [ ] Retry automático

### Fase 3: Features Avanzadas 📋
- [ ] Webhooks
- [ ] Archivos adicionales (multi-file projects)
- [ ] Custom test cases
- [ ] Batch submissions
- [ ] WebSocket para resultados en tiempo real

### Fase 4: Optimización 🚀
- [ ] Cache de imágenes Docker
- [ ] Pre-warming de contenedores
- [ ] Métricas con Prometheus
- [ ] Dashboards con Grafana

### Fase 5: Seguridad Avanzada 🔐
- [ ] Rate limiting por IP
- [ ] Autenticación con JWT
- [ ] API Keys
- [ ] Sandboxing con gVisor

## 📊 Monitoreo

### Ver logs de la API

```bash
docker-compose logs -f api
```

### Ver submissions en la base de datos

```bash
docker exec -it rojudger-postgres psql -U rojudger -d rojudger_db

# Dentro de psql:
SELECT id, language_id, status, time, memory FROM submissions;
```

## 🤝 Contribuir

¡Las contribuciones son bienvenidas! Por favor:

1. Fork el proyecto
2. Crea una rama para tu feature (`git checkout -b feature/AmazingFeature`)
3. Commit tus cambios (`git commit -m 'Add some AmazingFeature'`)
4. Push a la rama (`git push origin feature/AmazingFeature`)
5. Abre un Pull Request

## 📝 Licencia

Este proyecto está bajo la Licencia MIT. Ver `LICENSE` para más detalles.

## 🙏 Agradecimientos

Inspirado por:
- [Judge0](https://github.com/judge0/judge0) - Sistema de ejecución de código robusto
- [Go Playground](https://go.dev/blog/playground) - Diseño elegante del playground de Go
- [Isolate](https://github.com/ioi/isolate) - Sandbox para competitive programming

## 📧 Contacto

Proyecto creado por **Rocha** como parte de un sistema más grande similar a LeetCode.

---

**¡Happy Coding! 🎉**