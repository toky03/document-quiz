package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os/exec"
	"strings"
	"time"
)

type ClaudeCLILLM struct {
	maxCharsPerChapter int
	quizOptionCount    int
	timeout            time.Duration
	binaryPath         string
}

func NewClaudeCLILLM(maxCharsPerChapter, quizOptionCount int) *ClaudeCLILLM {
	return &ClaudeCLILLM{
		maxCharsPerChapter: maxCharsPerChapter,
		quizOptionCount:    quizOptionCount,
		timeout:            5 * time.Minute,
		binaryPath:         "claude",
	}
}

type claudeCLIResult struct {
	Type    string `json:"type"`
	Subtype string `json:"subtype"`
	IsError bool   `json:"is_error"`
	Result  string `json:"result"`
}

// GenerateQA shells out to the local `claude` CLI. The `apiKey` argument is
// ignored: authentication is whatever the CLI already has (OAuth/Max
// subscription via `claude login`). The `model` argument is forwarded as
// --model when non-empty (e.g. "sonnet", "opus", "haiku").
func (l *ClaudeCLILLM) GenerateQA(
	ctx context.Context,
	model, _, chapterTitle, contextText string,
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

	prompt := fmt.Sprintf(`Erzeuge %d Quizfragen auf Deutsch für das Kapitel "%s".
Mische Single-Choice- und Multiple-Choice-Fragen.
Jede Frage muss genau %d Antwortoptionen haben.
Nutze ausschließlich den Kontext.

Jede Frage muss eigenständig verständlich sein und darf nicht auf das
Ausgangsdokument verweisen. Vermeide Formulierungen wie „laut Text",
„im Dokument", „im Kapitel oben", „in der Abbildung", „auf Seite X" oder
ähnliche Bezüge auf die Quelle. Die Frage soll auch ohne Kenntnis des
Kontexts gestellt werden können.

Zu jeder Antwortoption gehört eine kurze Erklärung (1–2 Sätze, auf Deutsch),
warum diese Option richtig oder falsch ist. Auch die Erklärungen müssen
eigenständig verständlich sein und dürfen nicht auf das Ausgangsdokument
verweisen (keine Formulierungen wie „laut Text", „im Dokument", „wie oben
erwähnt", „siehe Abbildung"); benenne den Sachverhalt direkt statt darauf
zu verweisen. Die Erklärungen müssen dieselbe Reihenfolge wie die Optionen
haben.

Kontext:
%s

Antworte nur als gültiges JSON im Format:
[
	{
		"question": "...",
		"quiz_type": "single" | "multiple",
		"options": ["...", "...", "...", "..."],
		"correct_options": [0],
		"explanations": ["...", "...", "...", "..."]
	}
]
Die Indizes in correct_options sind 0-basiert. Das explanations-Array muss
genauso viele Einträge haben wie das options-Array.`, qaCount, chapterTitle, l.quizOptionCount, trimmedContext)

	cmdCtx, cancel := context.WithTimeout(ctx, l.timeout)
	defer cancel()

	args := []string{"-p", "--output-format", "json", "--tools", ""}
	if trimmed := strings.TrimSpace(model); trimmed != "" {
		args = append(args, "--model", trimmed)
	}

	cmd := exec.CommandContext(cmdCtx, l.binaryPath, args...)
	cmd.Stdin = strings.NewReader(prompt)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		var execErr *exec.Error
		if errors.As(err, &execErr) && errors.Is(execErr.Err, exec.ErrNotFound) {
			return nil, fmt.Errorf(
				"claude CLI nicht gefunden. Bitte Claude Code installieren und 'claude login' ausführen",
			)
		}
		if errors.Is(cmdCtx.Err(), context.DeadlineExceeded) {
			return nil, fmt.Errorf("claude CLI Timeout nach %s", l.timeout)
		}
		return nil, fmt.Errorf(
			"claude CLI fehlgeschlagen: %w (stderr: %s)",
			err,
			strings.TrimSpace(stderr.String()),
		)
	}

	var envelope claudeCLIResult
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		return nil, fmt.Errorf("claude CLI Antwort ist kein gültiges JSON: %w", err)
	}
	if envelope.IsError || envelope.Subtype != "success" {
		return nil, fmt.Errorf("claude CLI meldete Fehler: %s", envelope.Result)
	}

	cleaned := cleanJSONResponse(envelope.Result)
	var rawItems []llmQuestionPayload
	if err := json.Unmarshal([]byte(cleaned), &rawItems); err != nil {
		return nil, fmt.Errorf("claude Antwort ist kein valides Quiz-JSON: %w", err)
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

		var explanations []string
		if len(item.Explanations) == l.quizOptionCount {
			explanations = make([]string, 0, l.quizOptionCount)
			for _, exp := range item.Explanations {
				explanations = append(explanations, strings.TrimSpace(exp))
			}
		}

		shuffledOptions, remappedCorrect, shuffledExplanations := shuffleOptionsAndRemapCorrect(
			options, correct, explanations, rng,
		)

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
			Explanations:   shuffledExplanations,
		})
	}

	if len(normalized) == 0 {
		return nil, fmt.Errorf("claude CLI hat keine gültigen Fragen geliefert")
	}

	return normalized, nil
}
