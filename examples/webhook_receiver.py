#!/usr/bin/env python3
"""
ROJUDGER Webhook Receiver - Python + Flask

Este ejemplo muestra cómo recibir y verificar webhooks de ROJUDGER
con validación de firma HMAC-SHA256.

Instalación:
    pip install flask

Uso:
    WEBHOOK_SECRET="tu-secreto" python webhook_receiver.py
"""

import os
import hmac
import hashlib
import json
from datetime import datetime
from flask import Flask, request, jsonify

app = Flask(__name__)
WEBHOOK_SECRET = os.getenv('WEBHOOK_SECRET', '').encode()
PORT = int(os.getenv('PORT', 9000))


def verify_webhook_signature(body: bytes, signature: str) -> bool:
    """
    Verifica la firma HMAC del webhook

    Args:
        body: Raw body del request (bytes)
        signature: Firma recibida en header

    Returns:
        True si la firma es válida, False en caso contrario
    """
    if not WEBHOOK_SECRET:
        print('⚠️  WEBHOOK_SECRET no configurado. Saltando verificación.')
        return True

    expected = hmac.new(WEBHOOK_SECRET, body, hashlib.sha256).hexdigest()

    # Comparación segura contra timing attacks
    return hmac.compare_digest(signature, expected)


def handle_completed_submission(submission: dict):
    """Procesar submission completada exitosamente"""
    print('🎉 Submission completada exitosamente')

    # Aquí puedes:
    # - Actualizar base de datos local
    # - Enviar notificación al usuario
    # - Calcular estadísticas
    # - Actualizar leaderboard
    # - etc.

    # Ejemplo: guardar en DB (pseudo-código)
    # db.submissions.update(
    #     id=submission['id'],
    #     status='completed',
    #     result=submission['stdout'],
    #     time=submission['time']
    # )


def handle_error_submission(submission: dict):
    """Procesar submission con error"""
    print('❌ Submission falló')

    # Aquí puedes:
    # - Notificar al usuario del error
    # - Registrar para debugging
    # - Ofrecer retry automático
    # - etc.

    if submission.get('compile_output'):
        print('Falló en compilación')
    elif submission.get('exit_code', 0) != 0:
        print('Error en runtime')


def handle_timeout_submission(submission: dict):
    """Procesar submission con timeout"""
    print('⏱️  Submission excedió tiempo límite')

    # Aquí puedes:
    # - Notificar al usuario
    # - Sugerir optimización
    # - etc.


@app.route('/webhooks/rojudger', methods=['POST'])
def webhook_handler():
    """Handler principal del webhook"""

    signature = request.headers.get('X-Rojudger-Signature', '')
    submission_id = request.headers.get('X-Rojudger-Submission-Id')
    event = request.headers.get('X-Rojudger-Event')
    body = request.get_data()

    print('\n' + '=' * 60)
    print(f'📨 Webhook recibido: {datetime.now().isoformat()}')
    print('=' * 60)

    # 1. Verificar firma HMAC
    if WEBHOOK_SECRET:
        if not verify_webhook_signature(body, signature):
            print('❌ Firma HMAC inválida!')
            return jsonify({'error': 'Invalid signature'}), 401
        print('✅ Firma HMAC verificada')

    # 2. Parsear payload
    try:
        payload = json.loads(body)
    except json.JSONDecodeError as e:
        print(f'❌ JSON inválido: {e}')
        return jsonify({'error': 'Invalid JSON'}), 400

    # 3. Validar estructura
    if 'submission' not in payload or 'id' not in payload.get('submission', {}):
        print('❌ Payload inválido: falta submission.id')
        return jsonify({'error': 'Invalid payload'}), 400

    submission = payload['submission']

    # 4. Log de información
    print(f"📋 Submission ID: {submission['id']}")
    print(f"🏷️  Event: {event or payload.get('event')}")
    print(f"📊 Status: {submission['status']}")
    print(f"🔢 Exit Code: {submission.get('exit_code')}")
    print(f"⏱️  Time: {submission.get('time')}s")
    print(f"💾 Memory: {submission.get('memory')} KB")

    if submission.get('stdout'):
        print(f"📤 Stdout:\n{submission['stdout'][:200]}")

    if submission.get('stderr'):
        print(f"⚠️  Stderr:\n{submission['stderr'][:200]}")

    if submission.get('compile_output'):
        print(f"🔧 Compile Output:\n{submission['compile_output'][:200]}")

    if submission.get('message'):
        print(f"💬 Message: {submission['message']}")

    # 5. Procesar según status
    status = submission.get('status')
    if status == 'completed':
        handle_completed_submission(submission)
    elif status == 'error':
        handle_error_submission(submission)
    elif status == 'timeout':
        handle_timeout_submission(submission)
    else:
        print(f"⚠️  Status desconocido: {status}")

    # 6. Responder rápidamente
    print('✅ Webhook procesado correctamente\n')

    return jsonify({
        'status': 'received',
        'submission_id': submission['id'],
        'timestamp': datetime.now().isoformat()
    }), 200


@app.route('/health', methods=['GET'])
def health_check():
    """Health check endpoint"""
    return jsonify({
        'status': 'healthy',
        'timestamp': datetime.now().isoformat(),
        'webhook_secret_configured': bool(WEBHOOK_SECRET)
    })


@app.route('/test-webhook', methods=['POST'])
def test_webhook():
    """Endpoint de prueba manual"""
    print('🧪 Test webhook recibido:', request.get_json())
    return jsonify({'status': 'test received'})


@app.errorhandler(Exception)
def handle_error(error):
    """Manejo global de errores"""
    print(f'❌ Error: {error}')
    return jsonify({'error': str(error)}), 500


def print_banner():
    """Imprime banner de inicio"""
    print('╔════════════════════════════════════════════════════╗')
    print('║     ROJUDGER Webhook Receiver (Python)             ║')
    print('╚════════════════════════════════════════════════════╝')
    print(f'🚀 Servidor escuchando en http://0.0.0.0:{PORT}')
    print(f"🔒 HMAC Secret: {'✅ Configurado' if WEBHOOK_SECRET else '❌ No configurado'}")
    print('')
    print('📡 Endpoints:')
    print(f'   POST http://localhost:{PORT}/webhooks/rojudger')
    print(f'   GET  http://localhost:{PORT}/health')
    print('')
    print('💡 Para testear:')
    print('   curl -X POST http://localhost:8080/api/v1/submissions \\')
    print('     -d \'{"language_id": 71, "source_code": "print(\\"test\\")", \\')
    print(f'          "webhook_url": "http://localhost:{PORT}/webhooks/rojudger"}\'')
    print('')
    print('⏳ Esperando webhooks...\n')


if __name__ == '__main__':
    print_banner()

    # Ejecutar servidor
    app.run(
        host='0.0.0.0',
        port=PORT,
        debug=False  # Cambiar a True para desarrollo
    )
