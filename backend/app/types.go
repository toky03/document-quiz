package app

type UploadedFile struct {
	Name    string
	Content []byte
}

type UploadCommand struct {
	Model  string
	APIKey string
	Files  []UploadedFile
}

type UploadIssue struct {
	File    string `json:"file"`
	Stage   string `json:"stage"`
	Message string `json:"message"`
}

type UploadResult struct {
	ProcessedFiles    int           `json:"processed_files"`
	SuccessfulFiles   int           `json:"successful_files"`
	FailedFiles       int           `json:"failed_files"`
	ErrorCount        int           `json:"error_count"`
	Issues            []UploadIssue `json:"issues"`
	GeneratedChapters int           `json:"generated_chapters"`
	GeneratedPairs    int           `json:"generated_pairs"`
	TotalChunks       int           `json:"total_chunks"`
}

type QuizEvaluation struct {
	Index         int      `json:"index"`
	Question      string   `json:"question"`
	UserAnswer    []int    `json:"user_answer"`
	CorrectAnswer []int    `json:"correct_answer"`
	IsCorrect     bool     `json:"is_correct"`
	Options       []string `json:"options"`
	QuizType      string   `json:"quiz_type"`
}

type QuizEvaluationResult struct {
	CorrectCount int              `json:"correct_count"`
	TotalCount   int              `json:"total_count"`
	Results      []QuizEvaluation `json:"results"`
}

type Chapter struct {
	ID         int
	Title      string
	SourceName string
	SourceType string
	CreatedAt  string
	QACount    int
}

type QuizQuestion struct {
	Question       string   `json:"question"`
	QuizType       string   `json:"quiz_type"`
	Options        []string `json:"options"`
	CorrectOptions []int    `json:"correct_options"`
	Answer         string   `json:"answer"`
}

type VectorChunk struct {
	ID         string
	SourceName string
	ChunkIndex int
	ChunkText  string
}
