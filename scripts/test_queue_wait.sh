#!/bin/bash

echo "🧪 Test de Sistema de Colas con wait=true"
echo "==========================================="
echo ""

# Iniciar API
echo "1️⃣ Iniciando API en modo QUEUE..."
USE_QUEUE=true go run ./cmd/api > /tmp/rojudger-api-queue.log 2>&1 &
API_PID=$!
sleep 3
echo "✅ API corriendo (PID: $API_PID)"
echo ""

# Iniciar Worker
echo "2️⃣ Iniciando Worker..."
go run ./cmd/worker > /tmp/rojudger-worker.log 2>&1 &
WORKER_PID=$!
sleep 2
echo "✅ Worker corriendo (PID: $WORKER_PID)"
echo ""

# Test con wait=true
echo "3️⃣ Enviando código con wait=true (modo híbrido)..."
RESPONSE=$(curl -s -X POST "http://localhost:8080/api/v1/submissions?wait=true" \
  -H "Content-Type: application/json" \
  -d '{"language_id": 71, "source_code": "print(\"¡Funcionando con wait=true!\")"}')

echo "$RESPONSE" | python3 -m json.tool
echo ""

# Ver estadísticas
echo "4️⃣ Estadísticas de la cola:"
curl -s http://localhost:8080/api/v1/queue/stats | python3 -m json.tool
echo ""

# Cleanup
echo "🧹 Limpiando..."
kill $API_PID $WORKER_PID 2>/dev/null
wait $API_PID $WORKER_PID 2>/dev/null

echo ""
echo "✅ Test completado!"
