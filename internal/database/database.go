package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/RobertoRochaT/rojudger/internal/config"
	"github.com/RobertoRochaT/rojudger/internal/models"
	_ "github.com/lib/pq"
)

// DB es el wrapper de la base de datos
type DB struct {
	conn *sql.DB
}

// NewDB crea una nueva conexión a la base de datos
func NewDB(cfg *config.Config) (*DB, error) {
	dsn := cfg.GetDatabaseDSN()

	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configurar pool de conexiones
	conn.SetMaxOpenConns(25)
	conn.SetMaxIdleConns(5)
	conn.SetConnMaxLifetime(5 * time.Minute)

	// Verificar conexión
	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Database connected successfully")

	return &DB{conn: conn}, nil
}

// InitSchema crea las tablas necesarias si no existen
func (db *DB) InitSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS languages (
		id SERIAL PRIMARY KEY,
		name VARCHAR(50) NOT NULL UNIQUE,
		display_name VARCHAR(100) NOT NULL,
		version VARCHAR(50) NOT NULL,
		extension VARCHAR(10) NOT NULL,
		compile_cmd TEXT,
		execute_cmd TEXT NOT NULL,
		docker_image VARCHAR(200) NOT NULL,
		is_compiled BOOLEAN DEFAULT FALSE,
		is_enabled BOOLEAN DEFAULT TRUE,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS submissions (
		id VARCHAR(36) PRIMARY KEY,
		language_id INTEGER NOT NULL REFERENCES languages(id),
		source_code TEXT NOT NULL,
		stdin TEXT,
		expected_output TEXT,
		status VARCHAR(20) NOT NULL DEFAULT 'queued',
		stdout TEXT,
		stderr TEXT,
		exit_code INTEGER DEFAULT -1,
		time REAL DEFAULT 0,
		memory INTEGER DEFAULT 0,
		compile_output TEXT,
		message TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		finished_at TIMESTAMP,
		CONSTRAINT fk_language FOREIGN KEY (language_id) REFERENCES languages(id)
	);

	CREATE INDEX IF NOT EXISTS idx_submissions_status ON submissions(status);
	CREATE INDEX IF NOT EXISTS idx_submissions_created_at ON submissions(created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_submissions_language ON submissions(language_id);
	`

	_, err := db.conn.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	log.Println("Database schema initialized")
	return nil
}

// SeedLanguages inserta los lenguajes iniciales si no existen
func (db *DB) SeedLanguages() error {
	languages := []models.Language{
		{
			ID:          models.LanguagePython3,
			Name:        "python3",
			DisplayName: "Python 3",
			Version:     "3.11",
			Extension:   ".py",
			ExecuteCmd:  "python3 {file}",
			DockerImage: "python:3.11-slim",
			IsCompiled:  false,
			IsEnabled:   true,
		},
		{
			ID:          models.LanguageJavaScript,
			Name:        "javascript",
			DisplayName: "JavaScript (Node.js)",
			Version:     "20",
			Extension:   ".js",
			ExecuteCmd:  "node {file}",
			DockerImage: "node:20-slim",
			IsCompiled:  false,
			IsEnabled:   true,
		},
		{
			ID:          models.LanguageGo,
			Name:        "go",
			DisplayName: "Go",
			Version:     "1.21",
			Extension:   ".go",
			ExecuteCmd:  "go run {file}",
			DockerImage: "golang:1.21-alpine",
			IsCompiled:  false, // go run compila y ejecuta
			IsEnabled:   true,
		},
		{
			ID:          models.LanguageC,
			Name:        "c",
			DisplayName: "C (GCC)",
			Version:     "11",
			Extension:   ".c",
			CompileCmd:  "gcc {file} -o main",
			ExecuteCmd:  "./main",
			DockerImage: "gcc:11",
			IsCompiled:  true,
			IsEnabled:   true,
		},
		{
			ID:          models.LanguageCPP,
			Name:        "cpp",
			DisplayName: "C++ (G++)",
			Version:     "11",
			Extension:   ".cpp",
			CompileCmd:  "g++ {file} -o main",
			ExecuteCmd:  "./main",
			DockerImage: "gcc:11",
			IsCompiled:  true,
			IsEnabled:   true,
		},
	}

	for _, lang := range languages {
		// Insertar solo si no existe
		query := `
		INSERT INTO languages (id, name, display_name, version, extension, compile_cmd, execute_cmd, docker_image, is_compiled, is_enabled)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (name) DO NOTHING
		`
		_, err := db.conn.Exec(query,
			lang.ID, lang.Name, lang.DisplayName, lang.Version,
			lang.Extension, lang.CompileCmd, lang.ExecuteCmd,
			lang.DockerImage, lang.IsCompiled, lang.IsEnabled,
		)
		if err != nil {
			return fmt.Errorf("failed to seed language %s: %w", lang.Name, err)
		}
	}

	log.Println("Languages seeded successfully")
	return nil
}

// CreateSubmission inserta una nueva submission en la base de datos
func (db *DB) CreateSubmission(sub *models.Submission) error {
	query := `
	INSERT INTO submissions (id, language_id, source_code, stdin, expected_output, status, created_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := db.conn.Exec(query,
		sub.ID, sub.LanguageID, sub.SourceCode, sub.Stdin,
		sub.ExpectedOut, sub.Status, sub.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create submission: %w", err)
	}
	return nil
}

// GetSubmission obtiene una submission por ID
func (db *DB) GetSubmission(id string) (*models.Submission, error) {
	query := `
	SELECT id, language_id, source_code, stdin, expected_output, status,
	       stdout, stderr, exit_code, time, memory, compile_output, message,
	       created_at, finished_at
	FROM submissions
	WHERE id = $1
	`
	var sub models.Submission
	var finishedAt sql.NullTime
	var stdout, stderr, compileOut, message sql.NullString

	err := db.conn.QueryRow(query, id).Scan(
		&sub.ID, &sub.LanguageID, &sub.SourceCode, &sub.Stdin, &sub.ExpectedOut,
		&sub.Status, &stdout, &stderr, &sub.ExitCode, &sub.Time,
		&sub.Memory, &compileOut, &message, &sub.CreatedAt, &finishedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("submission not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get submission: %w", err)
	}

	// Handle nullable fields
	if stdout.Valid {
		sub.Stdout = stdout.String
	}
	if stderr.Valid {
		sub.Stderr = stderr.String
	}
	if compileOut.Valid {
		sub.CompileOut = compileOut.String
	}
	if message.Valid {
		sub.Message = message.String
	}
	if finishedAt.Valid {
		sub.FinishedAt = &finishedAt.Time
	}

	return &sub, nil
}

// UpdateSubmission actualiza una submission existente
func (db *DB) UpdateSubmission(sub *models.Submission) error {
	query := `
	UPDATE submissions
	SET status = $1, stdout = $2, stderr = $3, exit_code = $4,
	    time = $5, memory = $6, compile_output = $7, message = $8, finished_at = $9
	WHERE id = $10
	`
	_, err := db.conn.Exec(query,
		sub.Status, sub.Stdout, sub.Stderr, sub.ExitCode,
		sub.Time, sub.Memory, sub.CompileOut, sub.Message, sub.FinishedAt, sub.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update submission: %w", err)
	}
	return nil
}

// GetLanguage obtiene un lenguaje por ID
func (db *DB) GetLanguage(id int) (*models.Language, error) {
	query := `
	SELECT id, name, display_name, version, extension, compile_cmd,
	       execute_cmd, docker_image, is_compiled, is_enabled
	FROM languages
	WHERE id = $1 AND is_enabled = true
	`
	var lang models.Language
	var compileCmd sql.NullString

	err := db.conn.QueryRow(query, id).Scan(
		&lang.ID, &lang.Name, &lang.DisplayName, &lang.Version,
		&lang.Extension, &compileCmd, &lang.ExecuteCmd,
		&lang.DockerImage, &lang.IsCompiled, &lang.IsEnabled,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("language not found or disabled")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get language: %w", err)
	}

	if compileCmd.Valid {
		lang.CompileCmd = compileCmd.String
	}

	return &lang, nil
}

// GetAllLanguages obtiene todos los lenguajes habilitados
func (db *DB) GetAllLanguages() ([]models.Language, error) {
	query := `
	SELECT id, name, display_name, version, extension, compile_cmd,
	       execute_cmd, docker_image, is_compiled, is_enabled
	FROM languages
	WHERE is_enabled = true
	ORDER BY id
	`
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get languages: %w", err)
	}
	defer rows.Close()

	var languages []models.Language
	for rows.Next() {
		var lang models.Language
		var compileCmd sql.NullString

		err := rows.Scan(
			&lang.ID, &lang.Name, &lang.DisplayName, &lang.Version,
			&lang.Extension, &compileCmd, &lang.ExecuteCmd,
			&lang.DockerImage, &lang.IsCompiled, &lang.IsEnabled,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan language: %w", err)
		}

		if compileCmd.Valid {
			lang.CompileCmd = compileCmd.String
		}

		languages = append(languages, lang)
	}

	return languages, nil
}

// GetSubmissionsByStatus obtiene submissions por estado
func (db *DB) GetSubmissionsByStatus(status string, limit int) ([]models.Submission, error) {
	query := `
	SELECT id, language_id, source_code, stdin, expected_output, status,
	       stdout, stderr, exit_code, time, memory, compile_output, message,
	       created_at, finished_at
	FROM submissions
	WHERE status = $1
	ORDER BY created_at ASC
	LIMIT $2
	`
	rows, err := db.conn.Query(query, status, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get submissions: %w", err)
	}
	defer rows.Close()

	var submissions []models.Submission
	for rows.Next() {
		var sub models.Submission
		var finishedAt sql.NullTime
		var stdout, stderr, compileOut, message sql.NullString

		err := rows.Scan(
			&sub.ID, &sub.LanguageID, &sub.SourceCode, &sub.Stdin, &sub.ExpectedOut,
			&sub.Status, &stdout, &stderr, &sub.ExitCode, &sub.Time,
			&sub.Memory, &compileOut, &message, &sub.CreatedAt, &finishedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan submission: %w", err)
		}

		// Handle nullable fields
		if stdout.Valid {
			sub.Stdout = stdout.String
		}
		if stderr.Valid {
			sub.Stderr = stderr.String
		}
		if compileOut.Valid {
			sub.CompileOut = compileOut.String
		}
		if message.Valid {
			sub.Message = message.String
		}
		if finishedAt.Valid {
			sub.FinishedAt = &finishedAt.Time
		}

		submissions = append(submissions, sub)
	}

	return submissions, nil
}

// Close cierra la conexión a la base de datos
func (db *DB) Close() error {
	return db.conn.Close()
}

// Health verifica el estado de la base de datos
func (db *DB) Health() error {
	return db.conn.Ping()
}
