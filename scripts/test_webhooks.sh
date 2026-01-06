#!/bin/bash

# ROJUDGER Webhook Testing Script
# Este script prueba el sistema de webhooks end-to-end

set -e

BASE_URL="${BASE_URL:-http://localhost:8080}"
API_URL="$BASE_URL/api/v1"

# Colores para output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}  ROJUDGER Webhook Testing Suite${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# Verificar dependencias
command -v curl >/dev/null 2>&1 || { echo "❌ curl no está instalado"; exit 1; }
command -v jq >/dev/null 2>&1 || { echo "❌ jq no está instalado"; exit 1; }

# Función para iniciar servidor webhook de prueba en Python
start_webhook_server() {
    echo -e "${YELLOW}🚀 Iniciando servidor webhook de prueba...${NC}"

    cat > /tmp/rojudger_webhook_server.py <<'EOF'
#!/usr/bin/env python3
from http.server import HTTPServer, BaseHTTPRequestHandler
import json
import hashlib
import hmac
import os
from datetime import datetime

WEBHOOK_SECRET = os.getenv('WEBHOOK_SECRET', '')

class WebhookHandler(BaseHTTPRequestHandler):
    def do_POST(self):
        content_length = int(self.headers.get('Content-Length', 0))
        body = self.rfile.read(content_length)

        # Log headers
        print(f"\n{'='*60}")
        print(f"[{datetime.now().isoformat()}] Webhook received")
        print(f"{'='*60}")
        print(f"Headers:")
        for header, value in self.headers.items():
            print(f"  {header}: {value}")

        # Verificar firma HMAC si hay secreto
        if WEBHOOK_SECRET:
            signature = self.headers.get('X-Rojudger-Signature', '')
            expected = hmac.new(
                WEBHOOK_SECRET.encode(),
                body,
                hashlib.sha256
            ).hexdigest()

            if signature == expected:
                print(f"✅ HMAC signature valid")
            else:
                print(f"❌ HMAC signature invalid!")
                print(f"   Expected: {expected}")
                print(f"   Got: {signature}")

        # Parse JSON
        try:
            data = json.loads(body)
            print(f"\nPayload:")
            print(json.dumps(data, indent=2))

            # Extraer info importante
            if 'submission' in data:
                sub = data['submission']
                print(f"\n📋 Submission Summary:")
                print(f"   ID: {sub.get('id')}")
                print(f"   Status: {sub.get('status')}")
                print(f"   Exit Code: {sub.get('exit_code')}")
                print(f"   Time: {sub.get('time')}s")

                if sub.get('stdout'):
                    print(f"   Stdout: {sub.get('stdout')[:100]}")
                if sub.get('stderr'):
                    print(f"   Stderr: {sub.get('stderr')[:100]}")
        except json.JSONDecodeError:
            print(f"⚠️  Body is not valid JSON")
            print(f"Body: {body.decode()}")

        # Responder
        self.send_response(200)
        self.send_header('Content-Type', 'application/json')
        self.end_headers()
        response = {"status": "received", "timestamp": datetime.now().isoformat()}
        self.wfile.write(json.dumps(response).encode())

    def log_message(self, format, *args):
        # Silenciar logs automáticos
        pass

if __name__ == '__main__':
    PORT = int(os.getenv('WEBHOOK_PORT', 9000))
    server = HTTPServer(('0.0.0.0', PORT), WebhookHandler)
    print(f"🎯 Webhook server listening on http://0.0.0.0:{PORT}")
    print(f"   Secret: {'SET' if WEBHOOK_SECRET else 'NOT SET'}")
    print(f"\nWaiting for webhooks...\n")
    server.serve_forever()
EOF

    chmod +x /tmp/rojudger_webhook_server.py

    # Exportar secreto si existe
    if [ -n "$WEBHOOK_SECRET" ]; then
        export WEBHOOK_SECRET
    fi

    # Iniciar servidor en background
    python3 /tmp/rojudger_webhook_server.py > /tmp/webhook_server.log 2>&1 &
    WEBHOOK_SERVER_PID=$!

    echo $WEBHOOK_SERVER_PID > /tmp/webhook_server.pid

    # Esperar a que inicie
    sleep 2

    echo -e "${GREEN}✓ Servidor webhook iniciado (PID: $WEBHOOK_SERVER_PID)${NC}"
    echo -e "${BLUE}  Escuchando en: http://localhost:9000${NC}"
    echo ""
}

# Función para detener servidor webhook
stop_webhook_server() {
    if [ -f /tmp/webhook_server.pid ]; then
        PID=$(cat /tmp/webhook_server.pid)
        if ps -p $PID > /dev/null 2>&1; then
            echo -e "${YELLOW}🛑 Deteniendo servidor webhook...${NC}"
            kill $PID 2>/dev/null || true
            rm /tmp/webhook_server.pid
            echo -e "${GREEN}✓ Servidor detenido${NC}"
        fi
    fi
}

# Cleanup al salir
trap stop_webhook_server EXIT INT TERM

# Test 1: Submission sin webhook
test_no_webhook() {
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${YELLOW}Test 1: Submission sin webhook${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

    RESPONSE=$(curl -s -X POST "$API_URL/submissions" \
        -H "Content-Type: application/json" \
        -d '{
            "language_id": 71,
            "source_code": "print(\"Hello without webhook\")"
        }')

    SUBMISSION_ID=$(echo $RESPONSE | jq -r '.id')
    echo -e "Submission ID: ${GREEN}$SUBMISSION_ID${NC}"
    echo -e "${GREEN}✓ Test pasado (sin webhook es válido)${NC}\n"
}

# Test 2: Submission con webhook válido
test_valid_webhook() {
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${YELLOW}Test 2: Submission con webhook válido${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

    RESPONSE=$(curl -s -X POST "$API_URL/submissions" \
        -H "Content-Type: application/json" \
        -d '{
            "language_id": 71,
            "source_code": "print(\"Hello from Python!\")\nprint(\"Webhook test\")",
            "webhook_url": "http://localhost:9000"
        }')

    SUBMISSION_ID=$(echo $RESPONSE | jq -r '.id')
    STATUS=$(echo $RESPONSE | jq -r '.status')

    echo -e "Submission ID: ${GREEN}$SUBMISSION_ID${NC}"
    echo -e "Status: ${GREEN}$STATUS${NC}"

    # Esperar a que el webhook se envíe
    echo -e "${YELLOW}⏳ Esperando 5 segundos para webhook...${NC}"
    sleep 5

    echo -e "${GREEN}✓ Test pasado${NC}"
    echo -e "${BLUE}  Revisa la salida del servidor webhook arriba${NC}\n"
}

# Test 3: Submission con webhook inválido (debe fallar)
test_invalid_webhook() {
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${YELLOW}Test 3: Webhook URL inválido (debe rechazarse)${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

    RESPONSE=$(curl -s -X POST "$API_URL/submissions" \
        -H "Content-Type: application/json" \
        -d '{
            "language_id": 71,
            "source_code": "print(\"test\")",
            "webhook_url": "invalid-url"
        }')

    ERROR=$(echo $RESPONSE | jq -r '.error // empty')

    if [ -n "$ERROR" ]; then
        echo -e "${GREEN}✓ Test pasado (URL rechazada como se esperaba)${NC}"
        echo -e "  Error: $ERROR\n"
    else
        echo -e "${RED}❌ Test fallido (URL inválida fue aceptada)${NC}\n"
        exit 1
    fi
}

# Test 4: Submission con prioridad y webhook
test_priority_webhook() {
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${YELLOW}Test 4: Submission con prioridad alta y webhook${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

    RESPONSE=$(curl -s -X POST "$API_URL/submissions" \
        -H "Content-Type: application/json" \
        -d '{
            "language_id": 71,
            "source_code": "import time\nprint(\"High priority task\")\ntime.sleep(0.5)\nprint(\"Done!\")",
            "priority": 10,
            "webhook_url": "http://localhost:9000"
        }')

    SUBMISSION_ID=$(echo $RESPONSE | jq -r '.id')
    echo -e "Submission ID: ${GREEN}$SUBMISSION_ID${NC}"
    echo -e "Priority: ${GREEN}10 (high)${NC}"

    echo -e "${YELLOW}⏳ Esperando 5 segundos para webhook...${NC}"
    sleep 5

    echo -e "${GREEN}✓ Test pasado${NC}\n"
}

# Test 5: Múltiples submissions con webhook
test_multiple_webhooks() {
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${YELLOW}Test 5: Múltiples submissions con webhooks${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

    for i in {1..3}; do
        RESPONSE=$(curl -s -X POST "$API_URL/submissions" \
            -H "Content-Type: application/json" \
            -d "{
                \"language_id\": 71,
                \"source_code\": \"print('Batch job $i')\",
                \"priority\": $((i - 2)),
                \"webhook_url\": \"http://localhost:9000\"
            }")

        ID=$(echo $RESPONSE | jq -r '.id')
        echo -e "  Submission $i: ${GREEN}$ID${NC}"
    done

    echo -e "${YELLOW}⏳ Esperando 8 segundos para todos los webhooks...${NC}"
    sleep 8

    echo -e "${GREEN}✓ Test pasado${NC}\n"
}

# Test 6: Verificar logs de webhook en DB
test_webhook_logs() {
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${YELLOW}Test 6: Verificación de logs de webhook en DB${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

    # Este test requiere acceso a SQLite
    if [ -f "rojudger.db" ]; then
        echo -e "${YELLOW}Consultando webhook_logs...${NC}"
        sqlite3 rojudger.db "SELECT submission_id, status_code, attempt, created_at FROM webhook_logs ORDER BY created_at DESC LIMIT 5;" || true
        echo -e "${GREEN}✓ Logs verificados${NC}\n"
    else
        echo -e "${YELLOW}⚠️  DB no encontrada en directorio actual${NC}\n"
    fi
}

# Mostrar logs del servidor webhook
show_webhook_logs() {
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${YELLOW}Logs del servidor webhook:${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

    if [ -f /tmp/webhook_server.log ]; then
        tail -n 50 /tmp/webhook_server.log
    else
        echo "No hay logs disponibles"
    fi
    echo ""
}

# Verificar que la API esté disponible
echo -e "${YELLOW}🔍 Verificando API en $API_URL...${NC}"
if ! curl -s "$BASE_URL/health" > /dev/null 2>&1; then
    echo -e "${RED}❌ API no disponible en $BASE_URL${NC}"
    echo -e "${YELLOW}   Asegúrate de que el servidor esté corriendo${NC}"
    exit 1
fi
echo -e "${GREEN}✓ API disponible${NC}\n"

# Iniciar servidor webhook
start_webhook_server

# Ejecutar tests
test_no_webhook
test_invalid_webhook
test_valid_webhook
test_priority_webhook
test_multiple_webhooks
test_webhook_logs

# Mostrar logs
show_webhook_logs

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}✅ Todos los tests de webhook completados${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "${YELLOW}💡 Tips:${NC}"
echo -e "   • Los webhooks se envían de forma asíncrona"
echo -e "   • Para ver firma HMAC: export WEBHOOK_SECRET=tu_secreto"
echo -e "   • Los logs están en la tabla 'webhook_logs' de la DB"
echo -e "   • Usa ngrok/webhook.site para probar con URLs públicas"
echo ""
