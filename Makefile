.PHONY: help build run test clean docker-up docker-down docker-logs db-migrate db-seed dev install lint fmt

# Variables
APP_NAME=rojudger-api
DOCKER_COMPOSE=docker-compose
GO=go
GOFLAGS=-v

# Default target
help: ## Mostrar esta ayuda
	@echo "🚀 ROJUDGER - Makefile Commands"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""

# Development
install: ## Instalar dependencias de Go
	@echo "📦 Installing dependencies..."
	$(GO) mod download
	$(GO) mod tidy
	@echo "✅ Dependencies installed"

build: ## Compilar el binario
	@echo "🔨 Building $(APP_NAME)..."
	$(GO) build $(GOFLAGS) -o bin/$(APP_NAME) ./cmd/api
	@echo "✅ Build complete: bin/$(APP_NAME)"

run: ## Ejecutar la aplicación localmente
	@echo "🚀 Running $(APP_NAME)..."
	$(GO) run ./cmd/api/main.go

dev: ## Ejecutar en modo desarrollo con hot-reload (requiere air)
	@echo "🔥 Starting development server..."
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "⚠️  Air not installed. Installing..."; \
		go install github.com/cosmtrek/air@latest; \
		air; \
	fi

# Testing
test: ## Ejecutar tests de Go
	@echo "🧪 Running Go tests..."
	$(GO) test -v ./...

test-coverage: ## Ejecutar tests con coverage
	@echo "🧪 Running tests with coverage..."
	$(GO) test -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report generated: coverage.html"

test-api: ## Ejecutar pruebas de la API
	@echo "🧪 Testing API endpoints..."
	@bash scripts/test_api.sh

benchmark: ## Ejecutar benchmarks
	@echo "⚡ Running benchmarks..."
	$(GO) test -bench=. -benchmem ./...

# Docker
docker-build: ## Construir imagen Docker
	@echo "🐳 Building Docker image..."
	docker build -t $(APP_NAME):latest .
	@echo "✅ Docker image built"

docker-up: ## Levantar todos los servicios con Docker Compose
	@echo "🐳 Starting services with Docker Compose..."
	$(DOCKER_COMPOSE) up -d
	@echo "✅ Services started"
	@echo "🌐 API: http://localhost:8080"
	@echo "📊 PostgreSQL: localhost:5432"
	@echo "💾 Redis: localhost:6379"

docker-down: ## Detener todos los servicios
	@echo "🛑 Stopping services..."
	$(DOCKER_COMPOSE) down
	@echo "✅ Services stopped"

docker-restart: docker-down docker-up ## Reiniciar todos los servicios

docker-logs: ## Ver logs de los servicios
	$(DOCKER_COMPOSE) logs -f

docker-logs-api: ## Ver logs solo del API
	$(DOCKER_COMPOSE) logs -f api

docker-clean: ## Limpiar contenedores, imágenes y volúmenes
	@echo "🧹 Cleaning Docker resources..."
	$(DOCKER_COMPOSE) down -v --remove-orphans
	docker system prune -f
	@echo "✅ Cleanup complete"

# Database
db-shell: ## Conectar a PostgreSQL shell
	@echo "🗄️  Connecting to database..."
	docker exec -it rojudger-postgres psql -U rojudger -d rojudger_db

db-reset: ## Resetear base de datos (⚠️  elimina todos los datos)
	@echo "⚠️  Resetting database..."
	$(DOCKER_COMPOSE) stop postgres
	docker volume rm rojudger_postgres_data || true
	$(DOCKER_COMPOSE) up -d postgres
	@sleep 3
	@echo "✅ Database reset complete"

db-backup: ## Crear backup de la base de datos
	@echo "💾 Creating database backup..."
	@mkdir -p backups
	docker exec rojudger-postgres pg_dump -U rojudger rojudger_db > backups/backup_$$(date +%Y%m%d_%H%M%S).sql
	@echo "✅ Backup created in backups/"

# Code Quality
lint: ## Ejecutar linter (golangci-lint)
	@echo "🔍 Running linter..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "⚠️  golangci-lint not installed. Install with:"; \
		echo "    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

fmt: ## Formatear código
	@echo "💅 Formatting code..."
	$(GO) fmt ./...
	@echo "✅ Code formatted"

vet: ## Ejecutar go vet
	@echo "🔍 Running go vet..."
	$(GO) vet ./...

# Utility
clean: ## Limpiar archivos generados
	@echo "🧹 Cleaning..."
	rm -rf bin/
	rm -rf build/
	rm -f coverage.out coverage.html
	rm -rf tmp/
	$(GO) clean
	@echo "✅ Cleanup complete"

env: ## Copiar .env.example a .env
	@if [ ! -f .env ]; then \
		echo "📝 Creating .env from .env.example..."; \
		cp .env.example .env; \
		echo "✅ .env created. Please edit it with your configuration."; \
	else \
		echo "⚠️  .env already exists. Not overwriting."; \
	fi

# Monitoring
status: ## Ver estado de los servicios
	@echo "📊 Service Status:"
	@echo ""
	$(DOCKER_COMPOSE) ps
	@echo ""
	@echo "🔍 Health Check:"
	@curl -s http://localhost:8080/health | python3 -m json.tool || echo "❌ API not responding"

stats: ## Ver estadísticas de Docker
	@echo "📈 Docker Stats:"
	docker stats --no-stream rojudger-api rojudger-postgres rojudger-redis

# Production
prod-build: ## Build optimizado para producción
	@echo "🏗️  Building for production..."
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 $(GO) build -a -installsuffix cgo -ldflags="-w -s" -o bin/$(APP_NAME) ./cmd/api
	@echo "✅ Production build complete"

# Quick start
quick-start: env docker-up ## Setup rápido del proyecto
	@echo ""
	@echo "✅ ROJUDGER is ready!"
	@echo ""
	@echo "📚 Next steps:"
	@echo "  1. Wait a few seconds for services to start"
	@echo "  2. Test the API: make test-api"
	@echo "  3. View logs: make docker-logs"
	@echo ""
	@echo "🌐 API URL: http://localhost:8080"
	@echo "📖 Try: curl http://localhost:8080/api/v1/languages"

# Full setup
setup: install env docker-up ## Setup completo del entorno de desarrollo
	@echo "⏳ Waiting for services to be ready..."
	@sleep 5
	@echo ""
	@echo "✅ Development environment ready!"
	@echo ""
	@make status

# All in one
all: clean install build test ## Limpiar, instalar, compilar y testear

.DEFAULT_GOAL := help
