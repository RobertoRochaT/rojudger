#!/bin/bash

echo "🎯 Test del Sistema de Prioridades"
echo "===================================="
echo ""

API_URL="http://localhost:8080/api/v1/submissions"

echo "📝 Enviando submissions con diferentes prioridades..."
echo ""

# 1. Prioridad BAJA (-1)
echo "1️⃣ Enviando tarea BAJA (priority: -1)..."
RESPONSE1=$(curl -s -X POST $API_URL \
  -H "Content-Type: application/json" \
  -d '{
    "language_id": 71,
    "source_code": "import time\nprint(\"[LOW] Starting...\")\ntime.sleep(3)\nprint(\"[LOW] Finished!\")",
    "priority": -1
  }')
ID1=$(echo "$RESPONSE1" | jq -r '.id')
echo "   ✅ ID: $ID1"
echo ""

sleep 0.5

# 2. Prioridad NORMAL (0)
echo "2️⃣ Enviando tarea NORMAL (priority: 0)..."
RESPONSE2=$(curl -s -X POST $API_URL \
  -H "Content-Type: application/json" \
  -d '{
    "language_id": 71,
    "source_code": "import time\nprint(\"[NORMAL] Starting...\")\ntime.sleep(3)\nprint(\"[NORMAL] Finished!\")",
    "priority": 0
  }')
ID2=$(echo "$RESPONSE2" | jq -r '.id')
echo "   ✅ ID: $ID2"
echo ""

sleep 0.5

# 3. Prioridad ALTA (10)
echo "3️⃣ Enviando tarea ALTA/VIP (priority: 10)..."
RESPONSE3=$(curl -s -X POST $API_URL \
  -H "Content-Type: application/json" \
  -d '{
    "language_id": 71,
    "source_code": "import time\nprint(\"[HIGH/VIP] Starting...\")\ntime.sleep(3)\nprint(\"[HIGH/VIP] Finished!\")",
    "priority": 10
  }')
ID3=$(echo "$RESPONSE3" | jq -r '.id')
echo "   ✅ ID: $ID3"
echo ""

# 4. Más tareas NORMALES
echo "4️⃣ Enviando más tareas NORMALES..."
for i in {1..3}; do
  curl -s -X POST $API_URL \
    -H "Content-Type: application/json" \
    -d "{
      \"language_id\": 71,
      \"source_code\": \"print('[NORMAL-$i] Quick task')\",
      \"priority\": 0
    }" > /dev/null
  echo "   ✅ Normal task $i enqueued"
  sleep 0.2
done
echo ""

# 5. Una tarea URGENTE (prioridad 8)
echo "5️⃣ Enviando tarea URGENTE (priority: 8)..."
RESPONSE5=$(curl -s -X POST $API_URL \
  -H "Content-Type: application/json" \
  -d '{
    "language_id": 71,
    "source_code": "print(\"[URGENT] Emergency task!\")",
    "priority": 8
  }')
ID5=$(echo "$RESPONSE5" | jq -r '.id')
echo "   ✅ ID: $ID5"
echo ""

echo "⏳ Esperando 10 segundos para que se procesen..."
sleep 10
echo ""

echo "📊 Estadísticas de la Cola:"
curl -s http://localhost:8080/api/v1/queue/stats | jq '.'
echo ""

echo "📋 Resultados (orden de ejecución esperado):"
echo ""

echo "   🔥 ALTA/VIP (debería ejecutarse PRIMERO):"
curl -s http://localhost:8080/api/v1/submissions/$ID3 | jq '{id, status, stdout}'
echo ""

echo "   ⚡ URGENTE (debería ejecutarse SEGUNDO):"
curl -s http://localhost:8080/api/v1/submissions/$ID5 | jq '{id, status, stdout}'
echo ""

echo "   📌 NORMAL:"
curl -s http://localhost:8080/api/v1/submissions/$ID2 | jq '{id, status, stdout}'
echo ""

echo "   🐌 BAJA (debería ejecutarse AL FINAL):"
curl -s http://localhost:8080/api/v1/submissions/$ID1 | jq '{id, status, stdout}'
echo ""

echo "✅ Test de prioridades completado!"
echo ""
echo "💡 Verifica los logs del worker para ver el orden real de ejecución"
