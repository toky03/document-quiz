package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Chapter struct {
	ID         int    `json:"id"`
	Title      string `json:"title"`
	SourceName string `json:"source_name"`
	SourceType string `json:"source_type"`
	CreatedAt  string `json:"created_at"`
	QACount    int    `json:"question_count"`
}

type QAPair struct {
	ID             int      `json:"id"`
	ChapterID      int      `json:"chapter_id"`
	Question       string   `json:"question"`
	QuizType       string   `json:"quiz_type"`
	Options        []string `json:"options"`
	CorrectOptions []int    `json:"correct_options"`
	Answer         string   `json:"answer"`
	CreatedAt      string   `json:"created_at"`
}

type QuizQuestion struct {
	Question       string   `json:"question"`
	QuizType       string   `json:"quiz_type"`
	Options        []string `json:"options"`
	CorrectOptions []int    `json:"correct_options"`
	Answer         string   `json:"answer"`
	Explanations   []string `json:"explanations,omitempty"`
}

func getDB() (*sql.DB, error) {
	return sql.Open("sqlite3", SQLiteDBPath)
}

func addColumnIfMissing(db *sql.DB, table, column, columnType string) error {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid       int
			name      string
			ctype     string
			notnull   int
			dfltValue sql.NullString
			pk        int
		)
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
			return err
		}
		if name == column {
			return nil
		}
	}

	_, err = db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, columnType))
	return err
}

func initSQLiteDB() error {
	db, err := getDB()
	if err != nil {
		return err
	}
	defer db.Close()

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return err
	}

	// Create chapters table
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS chapters (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			source_name TEXT NOT NULL,
			source_type TEXT NOT NULL,
			created_at TEXT NOT NULL,
			UNIQUE(source_name, source_type)
		)
	`); err != nil {
		return err
	}

	// Create qa_pairs table
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS qa_pairs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			chapter_id INTEGER NOT NULL,
			question TEXT NOT NULL,
			quiz_type TEXT NOT NULL DEFAULT 'single',
			options_json TEXT NOT NULL DEFAULT '[]',
			correct_options_json TEXT NOT NULL DEFAULT '[]',
			answer TEXT NOT NULL,
			created_at TEXT NOT NULL,
			FOREIGN KEY(chapter_id) REFERENCES chapters(id) ON DELETE CASCADE
		)
	`); err != nil {
		return err
	}

	// Idempotent migration: add explanations_json to qa_pairs if missing.
	// SQLite has no IF NOT EXISTS for ADD COLUMN, so detect via PRAGMA.
	if err := addColumnIfMissing(db, "qa_pairs", "explanations_json", "TEXT"); err != nil {
		return err
	}

	// Create app settings table
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS app_settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)
	`); err != nil {
		return err
	}

	return nil
}

func setSetting(key, value string) error {
	db, err := getDB()
	if err != nil {
		return err
	}
	defer db.Close()

	updatedAt := time.Now().UTC().Format(time.RFC3339)
	_, err = db.Exec(`
		INSERT INTO app_settings (key, value, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(key)
		DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at
	`, key, value, updatedAt)

	return err
}

func deleteSetting(key string) error {
	db, err := getDB()
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec("DELETE FROM app_settings WHERE key = ?", key)
	return err
}

func getSetting(key string) (string, error) {
	db, err := getDB()
	if err != nil {
		return "", err
	}
	defer db.Close()

	var value string
	err = db.QueryRow("SELECT value FROM app_settings WHERE key = ?", key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}

	return value, nil
}

func upsertChapter(title, sourceName, sourceType string) (int, error) {
	db, err := getDB()
	if err != nil {
		return 0, err
	}
	defer db.Close()

	createdAt := time.Now().UTC().Format(time.RFC3339)

	_, err = db.Exec(`
		INSERT INTO chapters (title, source_name, source_type, created_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(source_name, source_type)
		DO UPDATE SET title = excluded.title
	`, title, sourceName, sourceType, createdAt)

	if err != nil {
		return 0, err
	}

	var chapterID int64
	row := db.QueryRow(
		"SELECT id FROM chapters WHERE source_name = ? AND source_type = ?",
		sourceName, sourceType,
	)
	if err := row.Scan(&chapterID); err != nil {
		return 0, err
	}

	return int(chapterID), nil
}

func replaceQAPairs(chapterID int, qaPairs []QuizQuestion) error {
	db, err := getDB()
	if err != nil {
		return err
	}
	defer db.Close()

	createdAt := time.Now().UTC().Format(time.RFC3339)

	// Delete existing QA pairs
	if _, err := db.Exec("DELETE FROM qa_pairs WHERE chapter_id = ?", chapterID); err != nil {
		return err
	}

	// Insert new QA pairs
	for _, pair := range qaPairs {
		optionsJSON, _ := json.Marshal(pair.Options)
		correctJSON, _ := json.Marshal(pair.CorrectOptions)
		var explanationsJSON sql.NullString
		if len(pair.Explanations) > 0 {
			b, _ := json.Marshal(pair.Explanations)
			explanationsJSON = sql.NullString{String: string(b), Valid: true}
		}

		_, err := db.Exec(`
			INSERT INTO qa_pairs (
				chapter_id, question, quiz_type, options_json,
				correct_options_json, answer, created_at, explanations_json
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, chapterID, pair.Question, pair.QuizType, string(optionsJSON),
			string(correctJSON), pair.Answer, createdAt, explanationsJSON)

		if err != nil {
			return err
		}
	}

	return nil
}

func listQuizChapters() ([]Chapter, error) {
	db, err := getDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT
			c.id, c.title, c.source_name, c.source_type, c.created_at,
			COUNT(q.id) AS question_count
		FROM chapters c
		JOIN qa_pairs q ON q.chapter_id = c.id
		WHERE q.options_json IS NOT NULL AND q.options_json != '[]'
		GROUP BY c.id, c.title, c.source_name
		ORDER BY c.title ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chapters []Chapter
	for rows.Next() {
		var c Chapter
		if err := rows.Scan(&c.ID, &c.Title, &c.SourceName, &c.SourceType, &c.CreatedAt, &c.QACount); err != nil {
			return nil, err
		}
		chapters = append(chapters, c)
	}

	return chapters, nil
}

func getChapterQuestions(chapterID int) ([]QuizQuestion, error) {
	db, err := getDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT question, quiz_type, options_json, correct_options_json, answer, explanations_json
		FROM qa_pairs
		WHERE chapter_id = ?
		  AND options_json IS NOT NULL
		  AND options_json != '[]'
		ORDER BY id ASC
	`, chapterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var questions []QuizQuestion
	for rows.Next() {
		var q QuizQuestion
		var optionsJSON, correctJSON string
		var explanationsJSON sql.NullString

		if err := rows.Scan(&q.Question, &q.QuizType, &optionsJSON, &correctJSON, &q.Answer, &explanationsJSON); err != nil {
			return nil, err
		}

		if err := json.Unmarshal([]byte(optionsJSON), &q.Options); err != nil {
			q.Options = []string{}
		}

		if err := json.Unmarshal([]byte(correctJSON), &q.CorrectOptions); err != nil {
			q.CorrectOptions = []int{}
		}

		if explanationsJSON.Valid && explanationsJSON.String != "" {
			if err := json.Unmarshal([]byte(explanationsJSON.String), &q.Explanations); err != nil {
				q.Explanations = nil
			}
		}

		questions = append(questions, q)
	}

	return questions, nil
}

func deleteChapter(chapterID int) error {
	db, err := getDB()
	if err != nil {
		return err
	}
	defer db.Close()

	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return err
	}

	result, err := db.Exec("DELETE FROM chapters WHERE id = ?", chapterID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("kapitel nicht gefunden")
	}

	return nil
}
