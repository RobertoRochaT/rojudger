/#!/bin/bash

# ROJUDGER - Script de inicio
# Este script inicia tanto el backend API como el frontend IDE

set -e

echo "🚀 =================================="
echo "🚀   ROJUDGER - Sistema Completo"
echo "🚀 =================================="
echo ""

# Colores
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Función para verificar si un puerto está en uso
check_port() {
    if lsof -Pi :$1 -sTCP:LISTEN -t >/dev/null 2>&1 ; then
        return 0
    else
        return 1
    fi
}

# Función para limpiar procesos al salir
cleanup() {
    echo ""
    echo -e "${YELLOW}🛑 Deteniendo servicios...${NC}"

    if [ ! -z "$API_PID" ]; then
        kill $API_PID 2>/dev/null || true
    fi

    if [ ! -z "$IDE_PID" ]; then
        kill $IDE_PID 2>/dev/null || true
    fi

    # Limpiar procesos huérfanos
    pkill -f "go run ./cmd/api" 2>/dev/null || true
    pkill -f "python3 -m http.server 3000" 2>/dev/null || true

    echo -e "${GREEN}✅ Servicios detenidos${NC}"
    exit 0
}

trap cleanup SIGINT SIGTERM EXIT

echo -e "${BLUE}📋 Verificando requisitos...${NC}"

# Verificar Go
if ! command -v go &> /dev/null; then
    echo -e "${RED}❌ Go no está instalado${NC}"
    echo "   Instala Go desde: https://go.dev/dl/"
    exit 1
fi

# Verificar Docker
if ! command -v docker &> /dev/null; then
    echo -e "${RED}❌ Docker no está instalado${NC}"
    echo "   Instala Docker desde: https://docs.docker.com/get-docker/"
    exit 1
fi

# Verificar que Docker está corriendo
if ! docker ps &> /dev/null; then
    echo -e "${RED}❌ Docker no está corriendo${NC}"
    echo "   Inicia Docker con: sudo systemctl start docker"
    exit 1
fi

# Verificar Python3
if ! command -v python3 &> /dev/null; then
    echo -e "${RED}❌ Python3 no está instalado${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Todos los requisitos están instalados${NC}"
echo ""

# Verificar si los puertos están disponibles
echo -e "${BLUE}🔍 Verificando puertos...${NC}"

if check_port 8080; then
    echo -e "${YELLOW}⚠️  Puerto 8080 está en uso. Limpiando...${NC}"
    lsof -ti:8080 | xargs kill -9 2>/dev/null || true
    sleep 2
fi

if check_port 3000; then
    echo -e "${YELLOW}⚠️  Puerto 3000 está en uso. Limpiando...${NC}"
    lsof -ti:3000 | xargs kill -9 2>/dev/null || true
    sleep 2
fi

echo -e "${GREEN}✅ Puertos disponibles${NC}"
echo ""

# Verificar y descargar imágenes de Docker necesarias
echo -e "${BLUE}🐳 Verificando imágenes de Docker...${NC}"

REQUIRED_IMAGES=("python:3.11-slim" "node:20-slim" "golang:1.21-alpine" "gcc:11")
MISSING_IMAGES=()

for image in "${REQUIRED_IMAGES[@]}"; do
    if ! docker image inspect "$image" > /dev/null 2>&1; then
        MISSING_IMAGES+=("$image")
    fi
done

if [ ${#MISSING_IMAGES[@]} -gt 0 ]; then
    echo -e "${YELLOW}⚠️  Faltan ${#MISSING_IMAGES[@]} imágenes de Docker${NC}"
    echo -e "${BLUE}📥 Descargando imágenes necesarias...${NC}"
    echo ""

    for image in "${MISSING_IMAGES[@]}"; do
        echo -e "${YELLOW}   Descargando $image...${NC}"
        if docker pull "$image" 2>&1 | grep -E "Downloaded|Already exists|Pull complete" | head -1; then
            echo -e "${GREEN}   ✅ $image descargada${NC}"
        else
            echo -e "${RED}   ❌ Error descargando $image${NC}"
            exit 1
        fi
    done
    echo ""
    echo -e "${GREEN}✅ Todas las imágenes descargadas${NC}"
else
    echo -e "${GREEN}✅ Todas las imágenes de Docker ya están disponibles${NC}"
fi
echo ""

# Verificar PostgreSQL y Redis
echo -e "${BLUE}🐘 Iniciando PostgreSQL y Redis con Docker...${NC}"

cd "$(dirname "$0")"

# Verificar si docker-compose está disponible
if command -v docker-compose &> /dev/null; then
    DOCKER_COMPOSE="docker-compose"
elif docker compose version &> /dev/null; then
    DOCKER_COMPOSE="docker compose"
else
    echo -e "${RED}❌ docker-compose no está disponible${NC}"
    exit 1
fi

# Levantar solo postgres y redis
$DOCKER_COMPOSE up -d postgres redis 2>&1 | grep -v "WARN.*version"

echo -e "${YELLOW}⏳ Esperando a que PostgreSQL esté listo...${NC}"
sleep 5

echo -e "${GREEN}✅ Base de datos lista${NC}"
echo ""

# Iniciar API Backend
echo -e "${BLUE}🚀 Iniciando ROJUDGER API Backend...${NC}"

# Asegurarse de que las dependencias están descargadas
go mod download 2>&1 | grep -v "^go: downloading" | grep -v "^$" || true

# Iniciar API en background
go run ./cmd/api > /tmp/rojudger-api.log 2>&1 &
API_PID=$!

echo -e "${YELLOW}⏳ Esperando a que la API esté lista...${NC}"
sleep 3

# Verificar que la API está corriendo
if ! kill -0 $API_PID 2>/dev/null; then
    echo -e "${RED}❌ Error al iniciar la API${NC}"
    echo "Ver logs en: /tmp/rojudger-api.log"
    tail -20 /tmp/rojudger-api.log
    exit 1
fi

# Verificar health check
for i in {1..10}; do
    if curl -s http://localhost:8080/health > /dev/null 2>&1; then
        echo -e "${GREEN}✅ API Backend corriendo en http://localhost:8080${NC}"
        break
    fi
    if [ $i -eq 10 ]; then
        echo -e "${RED}❌ API no responde en http://localhost:8080${NC}"
        echo "Ver logs en: /tmp/rojudger-api.log"
        tail -20 /tmp/rojudger-api.log
        exit 1
    fi
    sleep 1
done

echo ""

# Iniciar IDE Frontend
#echo -e "${BLUE}🎨 Iniciando ROJUDGER IDE Frontend...${NC}"

#cd compilador
#python3 -m http.server 3000 --directory public > /tmp/rojudger-ide.log 2>&1 &
#IDE_PID=$!

#echo -e "${YELLOW}⏳ Esperando a que el IDE esté listo...${NC}"
#sleep 2

# Verificar que el IDE está corriendo
#if ! kill -0 $IDE_PID 2>/dev/null; then
#    echo -e "${RED}❌ Error al iniciar el IDE${NC}"
#    exit 1
#fi

#echo -e "${GREEN}✅ IDE Frontend corriendo en http://localhost:3000${NC}"
#echo ""

# Resumen
echo -e "${GREEN}╔════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║                                                ║${NC}"
echo -e "${GREEN}║   ✅ ROJUDGER está corriendo correctamente     ║${NC}"
echo -e "${GREEN}║                                                ║${NC}"
echo -e "${GREEN}╚════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${BLUE}📍 URLs:${NC}"
echo -e "   🌐 IDE Frontend:  ${GREEN}http://localhost:3000${NC}"
echo -e "   🔧 API Backend:   ${GREEN}http://localhost:8080${NC}"
echo -e "   📊 Health Check:  ${GREEN}http://localhost:8080/health${NC}"
echo ""
echo -e "${BLUE}📚 Lenguajes soportados:${NC}"
echo -e "   • Python 3"
echo -e "   • JavaScript (Node.js)"
echo -e "   • Go"
echo -e "   • C (GCC)"
echo -e "   • C++ (G++)"
echo ""
echo -e "${BLUE}💡 Características:${NC}"
echo -e "   ✅ Compilación en tiempo real (auto-compile)"
echo -e "   ✅ Editor de código con syntax highlighting"
echo -e "   ✅ Ejemplos de código integrados"
echo -e "   ✅ Ejecución con entrada (stdin)"
echo -e "   ✅ Estadísticas de ejecución"
echo ""
echo -e "${YELLOW}🔥 Abre tu navegador en:${NC}"
echo -e "   👉 ${GREEN}http://localhost:3000${NC}"
echo ""
echo -e "${BLUE}📝 Logs:${NC}"
echo -e "   • API: ${YELLOW}/tmp/rojudger-api.log${NC}"
echo -e "   • IDE: ${YELLOW}/tmp/rojudger-ide.log${NC}"
echo ""
echo -e "${YELLOW}⚠️  Presiona Ctrl+C para detener todos los servicios${NC}"
echo ""

# Mantener el script corriendo y mostrar logs
tail -f /tmp/rojudger-api.log /tmp/rojudger-ide.log
