# ROJUDGER - Guía de Base de Datos

Esta guía te muestra todas las formas de ver y consultar la base de datos de ROJUDGER.

---

## 📊 Opción 1: pgAdmin (Interfaz Gráfica) - RECOMENDADO

### Paso 1: Iniciar pgAdmin
```bash
docker-compose up -d pgadmin
```

### Paso 2: Abrir en el navegador
```
http://localhost:5050
```

### Paso 3: Credenciales de acceso
- **Email:** `admin@rojudger.com`
- **Password:** `admin123`

### Paso 4: Conectar al servidor PostgreSQL

1. **Click derecho en "Servers" → "Register" → "Server"**

2. **Pestaña "General":**
   - Name: `ROJUDGER`

3. **Pestaña "Connection":**
   - Host name/address: `postgres` (o `rojudger-postgres`)
   - Port: `5432`
   - Maintenance database: `rojudger_db`
   - Username: `rojudger`
   - Password: `rojudger_password`
   - Save password: ✅ (marcar)

4. **Click "Save"**

### Paso 5: Navegar por la base de datos

```
ROJUDGER
  └── Databases
      └── rojudger_db
          └── Schemas
              └── public
                  └── Tables
                      ├── languages
                      └── submissions
```

### Consultas comunes en pgAdmin

Click derecho en una tabla → "View/Edit Data" → "All Rows"

O usa la pestaña "Query Tool" para ejecutar SQL personalizado.

---

## 💻 Opción 2: psql (Línea de Comandos)

### Método rápido con Makefile
```bash
make db-shell
```

### Método manual
```bash
docker exec -it rojudger-postgres psql -U rojudger -d rojudger_db
```

### Comandos útiles dentro de psql

#### Ver estructura de la base de datos
```sql
-- Listar todas las tablas
\dt

-- Ver estructura de la tabla submissions
\d submissions

-- Ver estructura de la tabla languages
\d languages

-- Listar todos los índices
\di

-- Ver todas las bases de datos
\l

-- Cambiar a otra base de datos
\c nombre_de_base_de_datos

-- Salir
\q
```

#### Consultas básicas

```sql
-- Ver todos los lenguajes disponibles
SELECT * FROM languages;

-- Ver lenguajes en formato bonito
SELECT id, display_name, version, docker_image 
FROM languages 
ORDER BY id;

-- Ver últimas 10 submissions
SELECT id, language_id, status, exit_code, created_at 
FROM submissions 
ORDER BY created_at DESC 
LIMIT 10;

-- Ver una submission específica (reemplaza el ID)
SELECT * FROM submissions 
WHERE id = 'tu-submission-id-aqui';

-- Ver submissions con el nombre del lenguaje
SELECT 
    s.id,
    l.display_name as language,
    s.status,
    s.exit_code,
    LEFT(s.source_code, 50) as code_preview,
    s.created_at
FROM submissions s
JOIN languages l ON s.language_id = l.id
ORDER BY s.created_at DESC
LIMIT 20;
```

#### Estadísticas

```sql
-- Contar submissions por estado
SELECT status, COUNT(*) as total 
FROM submissions 
GROUP BY status;

-- Contar submissions por lenguaje
SELECT 
    l.display_name,
    COUNT(s.id) as total_submissions
FROM languages l
LEFT JOIN submissions s ON l.id = s.language_id
GROUP BY l.display_name
ORDER BY total_submissions DESC;

-- Promedio de tiempo de ejecución por lenguaje
SELECT 
    l.display_name,
    ROUND(AVG(s.time)::numeric, 3) as avg_time_seconds,
    COUNT(s.id) as total
FROM submissions s
JOIN languages l ON s.language_id = l.id
WHERE s.status = 'completed'
GROUP BY l.display_name
ORDER BY avg_time_seconds DESC;

-- Submissions exitosas vs fallidas
SELECT 
    CASE 
        WHEN exit_code = 0 THEN 'Success'
        ELSE 'Failed'
    END as result,
    COUNT(*) as total
FROM submissions
WHERE status = 'completed'
GROUP BY result;
```

#### Ver contenido completo de submissions

```sql
-- Ver código fuente completo
SELECT id, source_code 
FROM submissions 
WHERE id = 'tu-submission-id';

-- Ver stdout completo
SELECT id, stdout 
FROM submissions 
WHERE id = 'tu-submission-id';

-- Ver stderr completo
SELECT id, stderr 
FROM submissions 
WHERE status = 'error' OR exit_code != 0
LIMIT 5;

-- Ver output de compilación
SELECT id, compile_output 
FROM submissions 
WHERE compile_output IS NOT NULL AND compile_output != ''
LIMIT 5;
```

#### Búsquedas avanzadas

```sql
-- Buscar submissions por código fuente
SELECT id, language_id, LEFT(source_code, 100)
FROM submissions
WHERE source_code LIKE '%fibonacci%';

-- Submissions que tardaron más de 1 segundo
SELECT id, language_id, time, status
FROM submissions
WHERE time > 1.0
ORDER BY time DESC;

-- Submissions con errores de compilación
SELECT id, language_id, compile_output
FROM submissions
WHERE compile_output LIKE '%error:%'
LIMIT 10;

-- Submissions de las últimas 24 horas
SELECT id, language_id, status, created_at
FROM submissions
WHERE created_at > NOW() - INTERVAL '24 hours'
ORDER BY created_at DESC;
```

---

## 🔧 Opción 3: API de ROJUDGER

### Ver submissions vía API

```bash
# Listar todas las submissions
curl http://localhost:8080/api/v1/submissions | python3 -m json.tool

# Ver submission específica
curl http://localhost:8080/api/v1/submissions/{submission_id} | python3 -m json.tool

# Filtrar por estado
curl "http://localhost:8080/api/v1/submissions?status=completed" | python3 -m json.tool
curl "http://localhost:8080/api/v1/submissions?status=error" | python3 -m json.tool
curl "http://localhost:8080/api/v1/submissions?status=queued" | python3 -m json.tool

# Ver lenguajes disponibles
curl http://localhost:8080/api/v1/languages | python3 -m json.tool
```

### Usando jq para mejor formato (si lo tienes instalado)

```bash
# Instalar jq (opcional)
sudo apt install jq  # Ubuntu/Debian
brew install jq      # macOS

# Usar jq
curl -s http://localhost:8080/api/v1/submissions | jq '.'
curl -s http://localhost:8080/api/v1/languages | jq '.[] | {id, name, version}'
```

---

## 📝 Opción 4: Script de consulta rápida

Crea un script para consultas frecuentes:

```bash
#!/bin/bash
# Guardar como: scripts/query_db.sh

echo "=== ROJUDGER Database Quick Query ==="
echo ""

docker exec rojudger-postgres psql -U rojudger -d rojudger_db -c "
SELECT 
    l.display_name as Language,
    COUNT(s.id) as Total,
    SUM(CASE WHEN s.exit_code = 0 THEN 1 ELSE 0 END) as Success,
    SUM(CASE WHEN s.exit_code != 0 THEN 1 ELSE 0 END) as Failed,
    ROUND(AVG(s.time)::numeric, 3) as AvgTime
FROM languages l
LEFT JOIN submissions s ON l.id = s.language_id
GROUP BY l.display_name
ORDER BY Total DESC;
"
```

Darle permisos y ejecutar:
```bash
chmod +x scripts/query_db.sh
./scripts/query_db.sh
```

---

## 🗃️ Opción 5: DBeaver (Cliente Universal)

Si prefieres un cliente más robusto:

### Descargar e instalar
```bash
# Ubuntu/Debian
wget https://dbeaver.io/files/dbeaver-ce_latest_amd64.deb
sudo dpkg -i dbeaver-ce_latest_amd64.deb

# O descarga desde: https://dbeaver.io/download/
```

### Configurar conexión

1. New Database Connection
2. PostgreSQL
3. Detalles:
   - **Host:** localhost
   - **Port:** 5432
   - **Database:** rojudger_db
   - **Username:** rojudger
   - **Password:** rojudger_password

---

## 📊 Estructura de las Tablas

### Tabla: `languages`

| Campo        | Tipo    | Descripción                        |
|--------------|---------|-------------------------------------|
| id           | INTEGER | ID único del lenguaje (PK)         |
| name         | VARCHAR | Nombre interno (ej: "python3")     |
| display_name | VARCHAR | Nombre para mostrar                |
| version      | VARCHAR | Versión (ej: "3.11")               |
| extension    | VARCHAR | Extensión de archivo (ej: ".py")   |
| compile_cmd  | TEXT    | Comando de compilación (opcional)  |
| execute_cmd  | TEXT    | Comando de ejecución               |
| docker_image | VARCHAR | Imagen de Docker a usar            |
| is_compiled  | BOOLEAN | Si requiere compilación            |
| is_enabled   | BOOLEAN | Si está habilitado                 |
| created_at   | TIMESTAMP | Fecha de creación                |

### Tabla: `submissions`

| Campo          | Tipo      | Descripción                          |
|----------------|-----------|--------------------------------------|
| id             | VARCHAR   | UUID único de la submission (PK)     |
| language_id    | INTEGER   | ID del lenguaje (FK → languages)     |
| source_code    | TEXT      | Código fuente enviado                |
| stdin          | TEXT      | Entrada estándar                     |
| expected_output| TEXT      | Salida esperada (para tests)         |
| status         | VARCHAR   | Estado: queued/processing/completed  |
| stdout         | TEXT      | Salida estándar del programa         |
| stderr         | TEXT      | Salida de error                      |
| exit_code      | INTEGER   | Código de salida (0 = éxito)         |
| time           | REAL      | Tiempo de ejecución (segundos)       |
| memory         | INTEGER   | Memoria usada (KB)                   |
| compile_output | TEXT      | Output de compilación                |
| message        | TEXT      | Mensaje de error/info                |
| created_at     | TIMESTAMP | Fecha de creación                    |
| finished_at    | TIMESTAMP | Fecha de finalización                |

---

## 🔍 Consultas Útiles Avanzadas

### Top 10 códigos más lentos
```sql
SELECT 
    s.id,
    l.display_name,
    s.time as seconds,
    LEFT(s.source_code, 80) as code_preview
FROM submissions s
JOIN languages l ON s.language_id = l.id
WHERE s.status = 'completed'
ORDER BY s.time DESC
LIMIT 10;
```

### Submissions de hoy
```sql
SELECT 
    COUNT(*) as total_today,
    SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) as completed,
    SUM(CASE WHEN status = 'error' THEN 1 ELSE 0 END) as errors
FROM submissions
WHERE created_at::date = CURRENT_DATE;
```

### Errores más comunes
```sql
SELECT 
    LEFT(stderr, 100) as error_message,
    COUNT(*) as occurrences
FROM submissions
WHERE stderr IS NOT NULL AND stderr != ''
GROUP BY LEFT(stderr, 100)
ORDER BY occurrences DESC
LIMIT 10;
```

### Rendimiento por hora
```sql
SELECT 
    EXTRACT(HOUR FROM created_at) as hour,
    COUNT(*) as submissions,
    ROUND(AVG(time)::numeric, 3) as avg_time
FROM submissions
WHERE created_at > NOW() - INTERVAL '24 hours'
GROUP BY hour
ORDER BY hour;
```

---

## 🧹 Mantenimiento de la Base de Datos

### Limpiar submissions antiguas
```sql
-- Ver cuántas submissions hay
SELECT COUNT(*) FROM submissions;

-- Ver submissions más antiguas
SELECT id, created_at 
FROM submissions 
ORDER BY created_at ASC 
LIMIT 10;

-- Eliminar submissions de más de 30 días
DELETE FROM submissions 
WHERE created_at < NOW() - INTERVAL '30 days';

-- O mantener solo las últimas 1000
DELETE FROM submissions
WHERE id NOT IN (
    SELECT id FROM submissions
    ORDER BY created_at DESC
    LIMIT 1000
);
```

### Backup de la base de datos
```bash
# Crear backup
docker exec rojudger-postgres pg_dump -U rojudger rojudger_db > backup.sql

# O con Makefile
make db-backup

# Restaurar desde backup
docker exec -i rojudger-postgres psql -U rojudger rojudger_db < backup.sql
```

### Ver tamaño de la base de datos
```sql
SELECT 
    pg_size_pretty(pg_database_size('rojudger_db')) as database_size,
    pg_size_pretty(pg_total_relation_size('submissions')) as submissions_table_size,
    pg_size_pretty(pg_total_relation_size('languages')) as languages_table_size;
```

---

## 🚨 Solución de Problemas

### No puedo conectar a la base de datos
```bash
# Verificar que PostgreSQL está corriendo
docker ps | grep postgres

# Ver logs de PostgreSQL
docker logs rojudger-postgres

# Reiniciar PostgreSQL
docker-compose restart postgres
```

### pgAdmin no carga
```bash
# Ver logs de pgAdmin
docker logs rojudger-pgadmin

# Reiniciar pgAdmin
docker-compose restart pgadmin

# Recrear pgAdmin
docker-compose rm -sf pgadmin
docker-compose up -d pgadmin
```

### Resetear la base de datos completamente
```bash
# ⚠️ CUIDADO: Esto elimina todos los datos
make db-reset

# O manualmente:
docker-compose down -v
docker volume rm rojudger_postgres_data
docker-compose up -d postgres
```

---

## 📚 Recursos Adicionales

- **PostgreSQL Docs:** https://www.postgresql.org/docs/
- **pgAdmin Docs:** https://www.pgadmin.org/docs/
- **SQL Tutorial:** https://www.postgresqltutorial.com/

---

## 🎯 Resumen Rápido

| Método      | Comando                               | Uso                     |
|-------------|---------------------------------------|-------------------------|
| **pgAdmin** | `http://localhost:5050`              | GUI, exploración visual |
| **psql**    | `make db-shell`                      | Terminal, queries SQL   |
| **API**     | `curl http://localhost:8080/api/...` | Programático            |
| **Script**  | `./scripts/query_db.sh`              | Queries frecuentes      |

**Recomendación:** Usa **pgAdmin** para exploración general y **psql** para consultas rápidas.

¡Feliz consulta de datos! 📊