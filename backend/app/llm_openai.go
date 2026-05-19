package app

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

type OpenAILLM struct {
	maxCharsPerChapter int
	quizOptionCount    int
}

func NewOpenAILLM(maxCharsPerChapter, quizOptionCount int) *OpenAILLM {
	return &OpenAILLM{
		maxCharsPerChapter: maxCharsPerChapter,
		quizOptionCount:    quizOptionCount,
	}
}

type llmQuestionPayload struct {
	Question       string   `json:"question"`
	QuizType       string   `json:"quiz_type"`
	Options        []string `json:"options"`
	CorrectOptions []int    `json:"correct_options"`
}

func (l *OpenAILLM) GenerateQA(
	ctx context.Context,
	model, apiKey, chapterTitle, contextText string,
	qaCount int,
) ([]QuizQuestion, error) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	trimmedContext := strings.TrimSpace(contextText)
	if trimmedContext == "" {
		return nil, fmt.Errorf("kein Kontext für LLM-Generierung vorhanden")
	}

	if len(trimmedContext) > l.maxCharsPerChapter {
		trimmedContext = trimmedContext[:l.maxCharsPerChapter]
	}

	client, err := openai.New(
		openai.WithToken(apiKey),
		openai.WithModel(model),
	)
	if err != nil {
		return nil, fmt.Errorf("LLM-Client konnte nicht initialisiert werden: %w", err)
	}

	prompt := fmt.Sprintf(`Erzeuge %d Quizfragen auf Deutsch für das Kapitel "%s".
Mische Single-Choice- und Multiple-Choice-Fragen.
Jede Frage muss genau %d Antwortoptionen haben.
Nutze ausschließlich den Kontext.

Kontext:
%s

Antworte nur als gültiges JSON im Format:
[
	{
		"question": "...",
		"quiz_type": "single" | "multiple",
		"options": ["...", "...", "...", "..."],
		"correct_options": [0]
	}
]
Die Indizes in correct_options sind 0-basiert.`, qaCount, chapterTitle, l.quizOptionCount, trimmedContext)

	response, err := llms.GenerateFromSinglePrompt(
		ctx,
		client,
		prompt,
		llms.WithTemperature(0.2),
	)
	if err != nil {
		return nil, fmt.Errorf("LLM-Anfrage fehlgeschlagen: %w", err)
	}

	cleaned := cleanJSONResponse(response)
	var rawItems []llmQuestionPayload
	if err := json.Unmarshal([]byte(cleaned), &rawItems); err != nil {
		return nil, fmt.Errorf("LLM-Antwort ist kein valides JSON: %w", err)
	}

	normalized := make([]QuizQuestion, 0, len(rawItems))
	for _, item := range rawItems {
		question := strings.TrimSpace(item.Question)
		quizType := strings.ToLower(strings.TrimSpace(item.QuizType))
		if quizType != "single" && quizType != "multiple" {
			quizType = "single"
		}

		if question == "" || len(item.Options) != l.quizOptionCount {
			continue
		}

		options := make([]string, 0, l.quizOptionCount)
		validOptions := true
		for _, opt := range item.Options {
			trimmed := strings.TrimSpace(opt)
			if trimmed == "" {
				validOptions = false
				break
			}
			options = append(options, trimmed)
		}
		if !validOptions {
			continue
		}

		correct := normalizeSelection(item.CorrectOptions, l.quizOptionCount)
		if len(correct) == 0 {
			continue
		}
		if quizType == "single" && len(correct) != 1 {
			continue
		}
		if quizType == "multiple" && len(correct) < 2 {
			continue
		}

		shuffledOptions, remappedCorrect := shuffleOptionsAndRemapCorrect(options, correct, rng)

		answerParts := make([]string, 0, len(remappedCorrect))
		for _, idx := range remappedCorrect {
			answerParts = append(answerParts, shuffledOptions[idx])
		}

		normalized = append(normalized, QuizQuestion{
			Question:       question,
			QuizType:       quizType,
			Options:        shuffledOptions,
			CorrectOptions: remappedCorrect,
			Answer:         strings.Join(answerParts, ", "),
		})
	}

	if len(normalized) == 0 {
		return nil, fmt.Errorf("LLM hat keine gültigen Fragen geliefert")
	}

	return normalized, nil
}

func shuffleOptionsAndRemapCorrect(
	options []string,
	correct []int,
	rng *rand.Rand,
) ([]string, []int) {
	if len(options) <= 1 {
		return append([]string(nil), options...), append([]int(nil), correct...)
	}

	permutation := rng.Perm(len(options))
	shuffledOptions := make([]string, len(options))
	oldToNewIndex := make(map[int]int, len(options))

	for newIdx, oldIdx := range permutation {
		shuffledOptions[newIdx] = options[oldIdx]
		oldToNewIndex[oldIdx] = newIdx
	}

	remappedCorrect := make([]int, 0, len(correct))
	for _, oldIdx := range correct {
		newIdx, exists := oldToNewIndex[oldIdx]
		if !exists {
			continue
		}
		remappedCorrect = append(remappedCorrect, newIdx)
	}
	sort.Ints(remappedCorrect)

	return shuffledOptions, remappedCorrect
}

func cleanJSONResponse(raw string) string {
	text := strings.TrimSpace(raw)
	re := regexp.MustCompile("(?s)^```(?:json)?\\s*(.*?)\\s*```$")
	matches := re.FindStringSubmatch(text)
	if len(matches) == 2 {
		return strings.TrimSpace(matches[1])
	}
	return text
}

func normalizeSelection(indices []int, optionCount int) []int {
	seen := map[int]struct{}{}
	normalized := make([]int, 0, len(indices))
	for _, idx := range indices {
		if idx < 0 || idx >= optionCount {
			continue
		}
		if _, exists := seen[idx]; exists {
			continue
		}
		seen[idx] = struct{}{}
		normalized = append(normalized, idx)
	}
	sort.Ints(normalized)
	return normalized
}
