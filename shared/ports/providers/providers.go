package providers

import "context"

// LLMProvider generates grounded text from retrieved context.
type LLMProvider interface {
	ProviderName() string
	ModelName() string
	Summarize(ctx context.Context, chunks []string, focus string) (string, error)
	AnswerQuestion(ctx context.Context, question string, chunks []string) (string, error)
}

// EmbeddingProvider turns text into vectors for retrieval/indexing.
type EmbeddingProvider interface {
	ProviderName() string
	ModelName() string
	Dimension() int
	Embed(ctx context.Context, text string) ([]float32, error)
}

type TranscriptionOptions struct {
	Language     string
	ProviderName string
	ModelName    string
}

type TranscriptionResult struct {
	Text         string
	ProviderName string
	ModelName    string
	Confidence   float64
}

type SpeechToTextProvider interface {
	ProviderName() string
	ModelName() string
	Transcribe(ctx context.Context, audio []byte, opts TranscriptionOptions) (TranscriptionResult, error)
}

type SpeechSynthesisOptions struct {
	VoiceID      string
	Language     string
	ProviderName string
	ModelName    string
}

type TextToSpeechProvider interface {
	ProviderName() string
	ModelName() string
	Synthesize(ctx context.Context, text string, opts SpeechSynthesisOptions) ([]byte, error)
}

type EmailAttachment struct {
	Filename    string
	ContentType string
	Content     []byte
}

type EmailMessage struct {
	ToEmail     string
	ToName      string
	Subject     string
	PlainText   string
	HTML        string
	Attachments []EmailAttachment
}

type EmailProvider interface {
	ProviderName() string
	SendEmail(ctx context.Context, msg EmailMessage) error
}

type SMSMessage struct {
	ToPhoneNumber string
	Body          string
}

type SMSProvider interface {
	ProviderName() string
	SendText(ctx context.Context, msg SMSMessage) error
}

type TelephonyCallRequest struct {
	ToE164       string
	FromE164     string
	CallbackURL  string
	ProviderName string
}

type TelephonyCallResult struct {
	CallID       string
	ProviderName string
}

type TelephonyProvider interface {
	ProviderName() string
	PlaceCall(ctx context.Context, req TelephonyCallRequest) (TelephonyCallResult, error)
}

type TranslationRequest struct {
	Text           string
	SourceLanguage string
	TargetLanguage string
	ProviderName   string
}

type TranslationResult struct {
	Text                        string
	ProviderName                string
	ConfidenceScore             float64
	MedicalTermPreservationPass bool
}

type TranslationProvider interface {
	ProviderName() string
	Translate(ctx context.Context, req TranslationRequest) (TranslationResult, error)
}

type VectorDocument struct {
	ID          string
	Text        string
	Embedding   []float32
	Metadata    map[string]string
	AccessLevel string
}

type VectorSearchRequest struct {
	QueryEmbedding []float32
	TopK           int
	Filters        map[string]string
}

type VectorSearchResult struct {
	DocumentID string
	Score      float32
	Text       string
	Metadata   map[string]string
}

type VectorSearchProvider interface {
	ProviderName() string
	IndexDocuments(ctx context.Context, docs []VectorDocument) error
	Search(ctx context.Context, req VectorSearchRequest) ([]VectorSearchResult, error)
}

type FHIRBundleIngestRequest struct {
	PatientID    string
	BundleJSON   string
	SourceSystem string
}

type FHIRBundleIngestResult struct {
	ResourcesStored int32
	ChunksStored    int32
}

type FHIRResourceQuery struct {
	PatientID    string
	ResourceType string
	Status       string
	Limit        int32
	Offset       int32
}

type FHIRProvider interface {
	ProviderName() string
	IngestBundle(ctx context.Context, req FHIRBundleIngestRequest) (FHIRBundleIngestResult, error)
	QueryResources(ctx context.Context, req FHIRResourceQuery) ([]string, error)
}
