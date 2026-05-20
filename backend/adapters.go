package main

import (
	"context"

	"document-quiz-backend/app"
)

type sqliteRelationalStoreAdapter struct{}

func (sqliteRelationalStoreAdapter) SetSetting(key, value string) error {
	return setSetting(key, value)
}

func (sqliteRelationalStoreAdapter) GetSetting(key string) (string, error) {
	return getSetting(key)
}

func (sqliteRelationalStoreAdapter) DeleteSetting(key string) error {
	return deleteSetting(key)
}

func (sqliteRelationalStoreAdapter) UpsertChapter(
	title, sourceName, sourceType string,
) (int, error) {
	return upsertChapter(title, sourceName, sourceType)
}

func (sqliteRelationalStoreAdapter) ReplaceQAPairs(
	chapterID int,
	qaPairs []app.QuizQuestion,
) error {
	converted := make([]QuizQuestion, 0, len(qaPairs))
	for _, q := range qaPairs {
		converted = append(converted, QuizQuestion{
			Question:       q.Question,
			QuizType:       q.QuizType,
			Options:        append([]string(nil), q.Options...),
			CorrectOptions: append([]int(nil), q.CorrectOptions...),
			Answer:         q.Answer,
		})
	}
	return replaceQAPairs(chapterID, converted)
}

func (sqliteRelationalStoreAdapter) ListQuizChapters() ([]app.Chapter, error) {
	chapters, err := listQuizChapters()
	if err != nil {
		return nil, err
	}
	converted := make([]app.Chapter, 0, len(chapters))
	for _, ch := range chapters {
		converted = append(converted, app.Chapter{
			ID:         ch.ID,
			Title:      ch.Title,
			SourceName: ch.SourceName,
			SourceType: ch.SourceType,
			CreatedAt:  ch.CreatedAt,
			QACount:    ch.QACount,
		})
	}
	return converted, nil
}

func (sqliteRelationalStoreAdapter) GetChapterQuestions(chapterID int) ([]app.QuizQuestion, error) {
	questions, err := getChapterQuestions(chapterID)
	if err != nil {
		return nil, err
	}
	converted := make([]app.QuizQuestion, 0, len(questions))
	for _, q := range questions {
		converted = append(converted, app.QuizQuestion{
			Question:       q.Question,
			QuizType:       q.QuizType,
			Options:        append([]string(nil), q.Options...),
			CorrectOptions: append([]int(nil), q.CorrectOptions...),
			Answer:         q.Answer,
		})
	}
	return converted, nil
}

func (sqliteRelationalStoreAdapter) DeleteChapter(chapterID int) error {
	return deleteChapter(chapterID)
}

type chromaVectorStoreAdapter struct{}

func (chromaVectorStoreAdapter) ReplaceChunks(
	_ context.Context,
	sourceName string,
	chunks []app.VectorChunk,
	openAIAPIKey string,
) error {
	converted := make([]VectorChunk, 0, len(chunks))
	for _, c := range chunks {
		converted = append(converted, VectorChunk{
			ID:         c.ID,
			SourceName: c.SourceName,
			ChunkIndex: c.ChunkIndex,
			ChunkText:  c.ChunkText,
		})
	}
	return replaceVectorChunks(sourceName, converted, openAIAPIKey)
}
