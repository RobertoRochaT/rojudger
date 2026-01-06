package models

import (
	"time"
)

// Submission representa una solicitud de ejecución de código
type Submission struct {
	ID          string     `json:"id" db:"id"`
	LanguageID  int        `json:"language_id" db:"language_id"`
	SourceCode  string     `json:"source_code" db:"source_code"`
	Stdin       string     `json:"stdin,omitempty" db:"stdin"`
	ExpectedOut string     `json:"expected_output,omitempty" db:"expected_output"`
	Status      string     `json:"status" db:"status"` // queued, processing, completed, error
	Stdout      string     `json:"stdout,omitempty" db:"stdout"`
	Stderr      string     `json:"stderr,omitempty" db:"stderr"`
	ExitCode    int        `json:"exit_code" db:"exit_code"`
	Time        float64    `json:"time" db:"time"`     // tiempo de ejecución en segundos
	Memory      int        `json:"memory" db:"memory"` // memoria usada en KB
	CompileOut  string     `json:"compile_output,omitempty" db:"compile_output"`
	Message     string     `json:"message,omitempty" db:"message"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	FinishedAt  *time.Time `json:"finished_at,omitempty" db:"finished_at"`
	WebhookURL  string     `json:"webhook_url,omitempty"`
}

// SubmissionRequest es lo que recibe la API
type SubmissionRequest struct {
	LanguageID  int    `json:"language_id" binding:"required"`
	SourceCode  string `json:"source_code" binding:"required"`
	Stdin       string `json:"stdin"`
	ExpectedOut string `json:"expected_output"`
	WebhookURL  string `json:"webhook_url,omitempty"`
	Priority    int    `json:"priority,omitempty"`
}

// SubmissionResponse es lo que devuelve la API
type SubmissionResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Token  string `json:"token,omitempty"` // para consultar el resultado después
}

// Language representa un lenguaje de programación soportado
type Language struct {
	ID          int    `json:"id" db:"id"`
	Name        string `json:"name" db:"name"`
	DisplayName string `json:"display_name" db:"display_name"`
	Version     string `json:"version" db:"version"`
	Extension   string `json:"extension" db:"extension"` // .py, .js, .go, etc.
	CompileCmd  string `json:"compile_cmd,omitempty" db:"compile_cmd"`
	ExecuteCmd  string `json:"execute_cmd" db:"execute_cmd"`
	DockerImage string `json:"docker_image" db:"docker_image"`
	IsCompiled  bool   `json:"is_compiled" db:"is_compiled"` // true para C, C++, Go, etc.
	IsEnabled   bool   `json:"is_enabled" db:"is_enabled"`
}

// Status constants
const (
	StatusQueued     = "queued"
	StatusProcessing = "processing"
	StatusCompleted  = "completed"
	StatusError      = "error"
	StatusTimeout    = "timeout"
)

// Language IDs (como Judge0)
const (
	LanguagePython3    = 71
	LanguageJavaScript = 63
	LanguageJava       = 62
	LanguageCPP        = 54
	LanguageC          = 50
	LanguageGo         = 60
	LanguageRust       = 73
)

// ExecutionResult contiene los resultados de la ejecución
type ExecutionResult struct {
	Stdout     string
	Stderr     string
	ExitCode   int
	Time       float64 // en segundos
	Memory     int     // en KB
	CompileOut string
	Error      string
	TimedOut   bool
}

// NewSubmission crea una nueva submission con valores por defecto
func NewSubmission(req SubmissionRequest, id string) *Submission {
	return &Submission{
		ID:          id,
		LanguageID:  req.LanguageID,
		SourceCode:  req.SourceCode,
		Stdin:       req.Stdin,
		ExpectedOut: req.ExpectedOut,
		WebhookURL:  req.WebhookURL,
		Status:      StatusQueued,
		CreatedAt:   time.Now(),
		ExitCode:    -1,
	}
}

// IsFinished verifica si la submission ya terminó de procesarse
func (s *Submission) IsFinished() bool {
	return s.Status == StatusCompleted ||
		s.Status == StatusError ||
		s.Status == StatusTimeout
}

// MarkAsProcessing marca la submission como en procesamiento
func (s *Submission) MarkAsProcessing() {
	s.Status = StatusProcessing
}

// MarkAsCompleted marca la submission como completada
func (s *Submission) MarkAsCompleted(result ExecutionResult) {
	now := time.Now()
	s.Status = StatusCompleted
	s.Stdout = result.Stdout
	s.Stderr = result.Stderr
	s.ExitCode = result.ExitCode
	s.Time = result.Time
	s.Memory = result.Memory
	s.CompileOut = result.CompileOut
	s.FinishedAt = &now

	if result.TimedOut {
		s.Status = StatusTimeout
		s.Message = "Execution timed out"
	}
}

// MarkAsError marca la submission como error
func (s *Submission) MarkAsError(errMsg string) {
	now := time.Now()
	s.Status = StatusError
	s.Message = errMsg
	s.FinishedAt = &now
}
