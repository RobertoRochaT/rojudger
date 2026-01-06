/**
 * ROJUDGER Webhook Receiver - Node.js + Express
 *
 * Este ejemplo muestra cómo recibir y verificar webhooks de ROJUDGER
 * con validación de firma HMAC-SHA256.
 *
 * Instalación:
 *   npm install express
 *
 * Uso:
 *   WEBHOOK_SECRET="tu-secreto" node webhook_receiver.js
 */

const express = require('express');
const crypto = require('crypto');

const app = express();
const PORT = process.env.PORT || 9000;
const WEBHOOK_SECRET = process.env.WEBHOOK_SECRET || '';

// ⚠️ Importante: usar raw body para verificar HMAC
app.use('/webhooks/rojudger', express.raw({ type: 'application/json' }));
app.use(express.json()); // Para otras rutas

/**
 * Verifica la firma HMAC del webhook
 */
function verifyWebhookSignature(body, signature) {
  if (!WEBHOOK_SECRET) {
    console.warn('⚠️  WEBHOOK_SECRET no configurado. Saltando verificación.');
    return true;
  }

  const hmac = crypto.createHmac('sha256', WEBHOOK_SECRET);
  hmac.update(body);
  const expectedSignature = hmac.digest('hex');

  // Comparación segura contra timing attacks
  return crypto.timingSafeEqual(
    Buffer.from(signature),
    Buffer.from(expectedSignature)
  );
}

/**
 * Handler principal del webhook
 */
app.post('/webhooks/rojudger', (req, res) => {
  const signature = req.headers['x-rojudger-signature'] || '';
  const submissionId = req.headers['x-rojudger-submission-id'];
  const event = req.headers['x-rojudger-event'];
  const body = req.body;

  console.log('\n' + '='.repeat(60));
  console.log(`📨 Webhook recibido: ${new Date().toISOString()}`);
  console.log('='.repeat(60));

  // 1. Verificar firma HMAC
  if (WEBHOOK_SECRET) {
    if (!verifyWebhookSignature(body, signature)) {
      console.error('❌ Firma HMAC inválida!');
      return res.status(401).json({ error: 'Invalid signature' });
    }
    console.log('✅ Firma HMAC verificada');
  }

  // 2. Parsear payload
  let payload;
  try {
    payload = JSON.parse(body);
  } catch (err) {
    console.error('❌ JSON inválido:', err.message);
    return res.status(400).json({ error: 'Invalid JSON' });
  }

  // 3. Validar estructura
  if (!payload.submission || !payload.submission.id) {
    console.error('❌ Payload inválido: falta submission.id');
    return res.status(400).json({ error: 'Invalid payload' });
  }

  const { submission } = payload;

  // 4. Log de información
  console.log(`📋 Submission ID: ${submission.id}`);
  console.log(`🏷️  Event: ${event || payload.event}`);
  console.log(`📊 Status: ${submission.status}`);
  console.log(`🔢 Exit Code: ${submission.exit_code}`);
  console.log(`⏱️  Time: ${submission.time}s`);
  console.log(`💾 Memory: ${submission.memory} KB`);

  if (submission.stdout) {
    console.log(`📤 Stdout:\n${submission.stdout.substring(0, 200)}`);
  }

  if (submission.stderr) {
    console.log(`⚠️  Stderr:\n${submission.stderr.substring(0, 200)}`);
  }

  if (submission.compile_output) {
    console.log(`🔧 Compile Output:\n${submission.compile_output.substring(0, 200)}`);
  }

  if (submission.message) {
    console.log(`💬 Message: ${submission.message}`);
  }

  // 5. Procesar según status
  switch (submission.status) {
    case 'completed':
      handleCompletedSubmission(submission);
      break;
    case 'error':
      handleErrorSubmission(submission);
      break;
    case 'timeout':
      handleTimeoutSubmission(submission);
      break;
    default:
      console.warn(`⚠️  Status desconocido: ${submission.status}`);
  }

  // 6. Responder rápidamente
  res.status(200).json({
    status: 'received',
    submission_id: submission.id,
    timestamp: new Date().toISOString()
  });

  console.log('✅ Webhook procesado correctamente\n');
});

/**
 * Procesar submission completada exitosamente
 */
function handleCompletedSubmission(submission) {
  console.log('🎉 Submission completada exitosamente');

  // Aquí puedes:
  // - Actualizar base de datos local
  // - Enviar notificación al usuario
  // - Calcular estadísticas
  // - Actualizar leaderboard
  // - etc.

  // Ejemplo: guardar en DB (pseudo-código)
  // db.submissions.update({
  //   id: submission.id,
  //   status: 'completed',
  //   result: submission.stdout,
  //   time: submission.time
  // });
}

/**
 * Procesar submission con error
 */
function handleErrorSubmission(submission) {
  console.log('❌ Submission falló');

  // Aquí puedes:
  // - Notificar al usuario del error
  // - Registrar para debugging
  // - Ofrecer retry automático
  // - etc.

  if (submission.compile_output) {
    console.log('Falló en compilación');
  } else if (submission.exit_code !== 0) {
    console.log('Error en runtime');
  }
}

/**
 * Procesar submission con timeout
 */
function handleTimeoutSubmission(submission) {
  console.log('⏱️  Submission excedió tiempo límite');

  // Aquí puedes:
  // - Notificar al usuario
  // - Sugerir optimización
  // - etc.
}

/**
 * Health check endpoint
 */
app.get('/health', (req, res) => {
  res.json({
    status: 'healthy',
    timestamp: new Date().toISOString(),
    webhook_secret_configured: !!WEBHOOK_SECRET
  });
});

/**
 * Endpoint de prueba manual
 */
app.post('/test-webhook', express.json(), (req, res) => {
  console.log('🧪 Test webhook recibido:', req.body);
  res.json({ status: 'test received' });
});

/**
 * Iniciar servidor
 */
app.listen(PORT, () => {
  console.log('╔════════════════════════════════════════════════════╗');
  console.log('║     ROJUDGER Webhook Receiver (Node.js)            ║');
  console.log('╚════════════════════════════════════════════════════╝');
  console.log(`🚀 Servidor escuchando en http://0.0.0.0:${PORT}`);
  console.log(`🔒 HMAC Secret: ${WEBHOOK_SECRET ? '✅ Configurado' : '❌ No configurado'}`);
  console.log('');
  console.log('📡 Endpoints:');
  console.log(`   POST http://localhost:${PORT}/webhooks/rojudger`);
  console.log(`   GET  http://localhost:${PORT}/health`);
  console.log('');
  console.log('💡 Para testear:');
  console.log('   curl -X POST http://localhost:8080/api/v1/submissions \\');
  console.log('     -d \'{"language_id": 71, "source_code": "print(\\"test\\")", \\');
  console.log(`          "webhook_url": "http://localhost:${PORT}/webhooks/rojudger"}\'`);
  console.log('');
  console.log('⏳ Esperando webhooks...\n');
});

// Manejo de errores global
process.on('uncaughtException', (err) => {
  console.error('❌ Uncaught Exception:', err);
});

process.on('unhandledRejection', (reason, promise) => {
  console.error('❌ Unhandled Rejection at:', promise, 'reason:', reason);
});

// Graceful shutdown
process.on('SIGINT', () => {
  console.log('\n👋 Deteniendo servidor...');
  process.exit(0);
});

module.exports = app; // Para testing
