package handlers

// CreateSubmissionRequest representa una petición para crear una submission
type CreateSubmissionRequest struct {
	LanguageID     int    `json:"language_id" binding:"required"`
	SourceCode     string `json:"source_code" binding:"required"`
	Stdin          string `json:"stdin"`
	ExpectedOutput string `json:"expected_output"`
	Priority       int    `json:"priority"`
	WebhookURL     string `json:"webhook_url,omitempty"`
}
