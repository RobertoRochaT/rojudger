#!/bin/bash

echo "🧪 Test de Sistema de Colas"
echo "============================"
echo ""

# Verificar que servicios base están corriendo
echo "1️⃣ Verificando servicios base..."
docker ps | grep -E "rojudger-(postgres|redis)" || {
    echo "❌ Servicios base no están corriendo"
    echo "Ejecuta: docker-compose up -d postgres redis"
    exit 1
}
echo "✅ PostgreSQL y Redis corriendo"
echo ""

# Iniciar API en modo queue
echo "2️⃣ Iniciando API en modo QUEUE..."
USE_QUEUE=true go run ./cmd/api > /tmp/rojudger-api-queue.log 2>&1 &
API_PID=$!
sleep 3

# Verificar que API inició
if ! curl -s http://localhost:8080/health > /dev/null 2>&1; then
    echo "❌ API no inició correctamente"
    cat /tmp/rojudger-api-queue.log
    kill $API_PID 2>/dev/null
    exit 1
fi
echo "✅ API corriendo en modo QUEUE (PID: $API_PID)"
echo ""

# Iniciar Worker
echo "3️⃣ Iniciando Worker..."
go run ./cmd/worker > /tmp/rojudger-worker.log 2>&1 &
WORKER_PID=$!
sleep 2
echo "✅ Worker corriendo (PID: $WORKER_PID)"
echo ""

# Enviar submission
echo "4️⃣ Enviando código a la cola..."
RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/submissions \
  -H "Content-Type: application/json" \
  -d '{
    "language_id": 71,
    "source_code": "print(\"¡Hola desde la cola de Redis!\")"
  }')

SUBMISSION_ID=$(echo "$RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin)['id'])" 2>/dev/null)

if [ -z "$SUBMISSION_ID" ]; then
    echo "❌ Error al enviar submission"
    echo "$RESPONSE"
    kill $API_PID $WORKER_PID 2>/dev/null
    exit 1
fi

echo "✅ Submission encolada: $SUBMISSION_ID"
echo ""

# Ver estadísticas de la cola
echo "5️⃣ Estadísticas de la cola:"
curl -s http://localhost:8080/api/v1/queue/stats | python3 -m json.tool
echo ""

# Esperar procesamiento
echo "6️⃣ Esperando procesamiento (5 segundos)..."
sleep 5
echo ""

# Ver resultado
echo "7️⃣ Resultado de la ejecución:"
curl -s http://localhost:8080/api/v1/submissions/$SUBMISSION_ID | python3 -m json.tool
echo ""

# Cleanup
echo ""
echo "🧹 Limpiando..."
kill $API_PID $WORKER_PID 2>/dev/null
wait $API_PID $WORKER_PID 2>/dev/null

echo ""
echo "✅ Test completado!"
echo ""
echo "📝 Logs disponibles en:"
echo "   - API: /tmp/rojudger-api-queue.log"
echo "   - Worker: /tmp/rojudger-worker.log"
