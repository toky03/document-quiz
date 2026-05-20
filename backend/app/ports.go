package app

import "context"

// APIPort is the inbound port used by HTTP handlers.
type APIPort interface {
	UploadDocuments(ctx context.Context, cmd UploadCommand) (UploadResult, error)
	SaveAPIKey(ctx context.Context, apiKey string) error
	IsAPIKeySaved(ctx context.Context) (bool, error)
	ListChapters(ctx context.Context) ([]Chapter, error)
	GetChapterQuestions(ctx context.Context, chapterID int) ([]QuizQuestion, error)
	SubmitQuiz(ctx context.Context, chapterID int, answers [][]int) (QuizEvaluationResult, error)
	DeleteChapter(ctx context.Context, chapterID int) error
	ClearAPIKey(ctx context.Context) error
}

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
