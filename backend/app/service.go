package app

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tmc/langchaingo/documentloaders"
)

type ServiceConfig struct {
	ChunkSize       int
	ChunkOverlap    int
	MaxCharsPerText int
	QuizOptionCount int
}

type QuizService struct {
	relationalStore RelationalStorePort
	vectorStore     VectorDBPort
	llm             LLMPort
	chunkSize       int
	chunkOverlap    int
}

func NewQuizService(
	relationalStore RelationalStorePort,
	vectorStore VectorDBPort,
	llm LLMPort,
	cfg ServiceConfig,
) *QuizService {
	return &QuizService{
		relationalStore: relationalStore,
		vectorStore:     vectorStore,
		llm:             llm,
		chunkSize:       cfg.ChunkSize,
		chunkOverlap:    cfg.ChunkOverlap,
	}
}

func (s *QuizService) UploadDocuments(
	ctx context.Context,
	cmd UploadCommand,
) (UploadResult, error) {
	if strings.TrimSpace(cmd.Model) == "" {
		return UploadResult{}, fmt.Errorf("model ist erforderlich")
	}
	if len(cmd.Files) == 0 {
		return UploadResult{}, fmt.Errorf("keine dateien hochgeladen")
	}

	apiKey := strings.TrimSpace(cmd.APIKey)
	if apiKey == "" {
		storedKey, err := s.relationalStore.GetSetting("openai_api_key")
		if err != nil {
			return UploadResult{}, fmt.Errorf("api key konnte nicht geladen werden: %w", err)
		}
		apiKey = strings.TrimSpace(storedKey)
	}
	if apiKey == "" {
		return UploadResult{}, fmt.Errorf("kein openai api key hinterlegt")
	}

	result := UploadResult{
		ProcessedFiles: len(cmd.Files),
		Issues:         make([]UploadIssue, 0),
	}

	addIssue := func(fileName, stage, message string, err error) {
		if err != nil {
			log.Printf(
				"Upload-Verarbeitung fehlgeschlagen (datei=%q, schritt=%s): %v",
				fileName,
				stage,
				err,
			)
		}
		result.Issues = append(result.Issues, UploadIssue{
			File:    fileName,
			Stage:   stage,
			Message: message,
		})
	}

	for _, file := range cmd.Files {
		text, err := extractPDFText(file.Content)
		if err != nil {
			result.FailedFiles++
			addIssue(file.Name, "extract_pdf_text", "PDF-Text konnte nicht extrahiert werden", err)
			continue
		}

		chunks := chunkText(text, s.chunkSize, s.chunkOverlap)
		if len(chunks) == 0 {
			result.FailedFiles++
			addIssue(file.Name, "chunk_text", "Es konnten keine Textabschnitte erzeugt werden", nil)
			continue
		}

		vectorChunks := make([]VectorChunk, 0, len(chunks))
		for chunkIndex, chunkText := range chunks {
			prefix := chunkText
			if len(prefix) > 80 {
				prefix = prefix[:80]
			}
			vectorChunks = append(vectorChunks, VectorChunk{
				ID:         hashDocID(file.Name, chunkIndex, prefix),
				SourceName: file.Name,
				ChunkIndex: chunkIndex,
				ChunkText:  chunkText,
			})
		}

		if err := s.vectorStore.ReplaceChunks(ctx, file.Name, vectorChunks, apiKey); err != nil {
			result.FailedFiles++
			addIssue(
				file.Name,
				"create_embeddings",
				"Embeddings konnten nicht gespeichert werden",
				err,
			)
			continue
		}

		chapterName := strings.TrimSuffix(file.Name, filepath.Ext(file.Name))
		chapterID, err := s.relationalStore.UpsertChapter(chapterName, file.Name, "pdf")
		if err != nil {
			result.FailedFiles++
			addIssue(file.Name, "upsert_chapter", "Kapitel konnte nicht gespeichert werden", err)
			continue
		}

		chapterText := strings.Join(chunks, "\n\n")
		qaPairs, err := s.llm.GenerateQA(ctx, cmd.Model, apiKey, chapterName, chapterText, 20)
		if err != nil {
			result.FailedFiles++
			addIssue(file.Name, "generate_quiz", "Quizfragen konnten nicht erzeugt werden", err)
			result.TotalChunks += len(chunks)
			continue
		}

		if len(qaPairs) == 0 {
			result.FailedFiles++
			addIssue(file.Name, "generate_quiz", "LLM hat keine gültigen Quizfragen geliefert", nil)
			result.TotalChunks += len(chunks)
			continue
		}

		if err := s.relationalStore.ReplaceQAPairs(chapterID, qaPairs); err != nil {
			result.FailedFiles++
			addIssue(file.Name, "save_quiz", "Quizfragen konnten nicht gespeichert werden", err)
			result.TotalChunks += len(chunks)
			continue
		}

		result.SuccessfulFiles++
		result.GeneratedChapters++
		result.GeneratedPairs += len(qaPairs)
		result.TotalChunks += len(chunks)
	}

	result.ErrorCount = len(result.Issues)
	return result, nil
}

func (s *QuizService) SaveAPIKey(_ context.Context, apiKey string) error {
	trimmed := strings.TrimSpace(apiKey)
	if trimmed == "" {
		return fmt.Errorf("api key darf nicht leer sein")
	}
	return s.relationalStore.SetSetting("openai_api_key", trimmed)
}

func (s *QuizService) IsAPIKeySaved(_ context.Context) (bool, error) {
	key, err := s.relationalStore.GetSetting("openai_api_key")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(key) != "", nil
}

func (s *QuizService) ListChapters(_ context.Context) ([]Chapter, error) {
	return s.relationalStore.ListQuizChapters()
}

func (s *QuizService) GetChapterQuestions(
	_ context.Context,
	chapterID int,
) ([]QuizQuestion, error) {
	return s.relationalStore.GetChapterQuestions(chapterID)
}

func (s *QuizService) SubmitQuiz(
	_ context.Context,
	chapterID int,
	answers [][]int,
) (QuizEvaluationResult, error) {
	questions, err := s.relationalStore.GetChapterQuestions(chapterID)
	if err != nil {
		return QuizEvaluationResult{}, err
	}

	correctCount := 0
	results := make([]QuizEvaluation, 0, len(questions))

	for idx, question := range questions {
		var userAnswer []int
		if idx < len(answers) {
			userAnswer = answers[idx]
		}

		isCorrect := isAnswerCorrect(userAnswer, question.CorrectOptions)
		if isCorrect {
			correctCount++
		}

		results = append(results, QuizEvaluation{
			Index:         idx + 1,
			Question:      question.Question,
			UserAnswer:    userAnswer,
			CorrectAnswer: question.CorrectOptions,
			IsCorrect:     isCorrect,
			Options:       question.Options,
			QuizType:      question.QuizType,
		})
	}

	return QuizEvaluationResult{
		CorrectCount: correctCount,
		TotalCount:   len(questions),
		Results:      results,
	}, nil
}

func extractPDFText(fileContent []byte) (string, error) {
	loader := documentloaders.NewPDF(bytes.NewReader(fileContent), int64(len(fileContent)))
	docs, err := loader.Load(context.Background())
	if err != nil {
		return "", fmt.Errorf("PDF konnte nicht gelesen werden: %w", err)
	}

	var b strings.Builder
	for _, doc := range docs {
		pageText := strings.TrimSpace(doc.PageContent)
		if pageText == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(pageText)
	}

	out := strings.TrimSpace(b.String())
	if out == "" {
		return "", fmt.Errorf("kein extrahierbarer Text im PDF gefunden")
	}

	return out, nil
}

func chunkText(text string, chunkSize, overlap int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return []string{}
	}

	var chunks []string
	start := 0
	textLen := len(text)

	for start < textLen {
		end := start + chunkSize
		if end > textLen {
			end = textLen
		}

		chunk := strings.TrimSpace(text[start:end])
		if chunk != "" {
			chunks = append(chunks, chunk)
		}

		if end == textLen {
			break
		}

		start = end - overlap
		if start < 0 {
			start = 0
		}
	}

	return chunks
}

func isAnswerCorrect(userAnswer, correctAnswer []int) bool {
	if len(userAnswer) != len(correctAnswer) {
		return false
	}

	userCopy := append([]int(nil), userAnswer...)
	correctCopy := append([]int(nil), correctAnswer...)
	sort.Ints(userCopy)
	sort.Ints(correctCopy)

	for i, ua := range userCopy {
		if ua != correctCopy[i] {
			return false
		}
	}

	return true
}

func hashDocID(pdfName string, chunkIndex int, chunkPrefix string) string {
	h := sha1.New()
	_, _ = io.WriteString(h, fmt.Sprintf("%s-%d-%s", pdfName, chunkIndex, chunkPrefix))
	return hex.EncodeToString(h.Sum(nil))
}
