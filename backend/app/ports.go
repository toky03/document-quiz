package app

import "context"

// APIPort is the inbound port used by HTTP handlers.
type APIPort interface {
	UploadDocuments(
		ctx context.Context,
		cmd UploadCommand,
		progress ProgressReporter,
	) (UploadResult, error)
	SaveAPIKey(ctx context.Context, apiKey string) error
	IsAPIKeySaved(ctx context.Context) (bool, error)
	ListChapters(ctx context.Context) ([]Chapter, error)
	GetChapterQuestions(ctx context.Context, chapterID int) ([]QuizQuestion, error)
	SubmitQuiz(ctx context.Context, chapterID int, answers [][]int) (QuizEvaluationResult, error)
	DeleteChapter(ctx context.Context, chapterID int) error
	ClearAPIKey(ctx context.Context) error
	GetProvider(ctx context.Context) (string, error)
	SetProvider(ctx context.Context, provider string) error
}

// LLM provider identifiers stored in app_settings under key "llm_provider".
const (
	ProviderOpenAI    = "openai"
	ProviderClaudeCLI = "claude_cli"
)

// ProgressEvent is emitted at each stage of an upload. Consumers receive a
// stream of these so they can render fine-grained UI feedback.
type ProgressEvent struct {
	Event          string        `json:"event"`
	File           string        `json:"file,omitempty"`
	Index          int           `json:"index,omitempty"`
	Total          int           `json:"total,omitempty"`
	Stage          string        `json:"stage,omitempty"`
	Message        string        `json:"message,omitempty"`
	ChunkCount     int           `json:"chunk_count,omitempty"`
	GeneratedPairs int           `json:"generated_pairs,omitempty"`
	Result         *UploadResult `json:"result,omitempty"`
}

// ProgressReporter receives streaming events. May be nil; callers must
// tolerate a no-op reporter.
type ProgressReporter func(ProgressEvent)

// RelationalStorePort is the outbound port for relational persistence and app settings.
type RelationalStorePort interface {
	SetSetting(key, value string) error
	GetSetting(key string) (string, error)
	UpsertChapter(title, sourceName, sourceType string) (int, error)
	ReplaceQAPairs(chapterID int, qaPairs []QuizQuestion) error
	ListQuizChapters() ([]Chapter, error)
	GetChapterQuestions(chapterID int) ([]QuizQuestion, error)
	DeleteChapter(chapterID int) error
	DeleteSetting(key string) error
}

// VectorDBPort is the outbound port for vector storage.
type VectorDBPort interface {
	ReplaceChunks(
		ctx context.Context,
		sourceName string,
		chunks []VectorChunk,
		openAIAPIKey string,
	) error
}

// LLMPort is the outbound port for question generation.
type LLMPort interface {
	GenerateQA(
		ctx context.Context,
		model, apiKey, chapterTitle, contextText string,
		qaCount int,
	) ([]QuizQuestion, error)
}
