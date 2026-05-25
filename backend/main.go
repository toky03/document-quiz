package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"document-quiz-backend/api"
	"document-quiz-backend/app"
)

var (
	VectorDBDir        = "vector_db"
	CollectionName     = "pdf_chunks"
	SQLiteDBPath       = "quiz_data.db"
	VectorDbDir        = "vector_db"
	ChunkSize          = 900
	ChunkOverlap       = 150
	MaxCharsPerChapter = 12000
	QuizOptionCount    = 4
)

func resolveSQLitePath() string {
	if envPath := os.Getenv("SQLITE_DB_PATH"); envPath != "" {
		return envPath
	}

	// When running from ./backend, prefer the shared DB in the repository root.
	if _, err := os.Stat(filepath.Join("..", "quiz_data.db")); err == nil {
		return filepath.Join("..", "quiz_data.db")
	}

	return SQLiteDBPath
}

func main() {
	SQLiteDBPath = resolveSQLitePath()

	// Initialize SQLite database
	if err := initSQLiteDB(); err != nil {
		log.Fatalf("Fehler beim Initialisieren der SQLite-DB: %v", err)
	}
	if err := initVectorStoreDB(); err != nil {
		log.Fatalf("Fehler beim Initialisieren der Vector-DB: %v", err)
	}

	// Setup standard Go router
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", handleHealth)

	relationalStore := sqliteRelationalStoreAdapter{}
	vectorStore := chromaVectorStoreAdapter{}
	openAILLM := app.NewOpenAILLM(MaxCharsPerChapter, QuizOptionCount)
	claudeCLILLM := app.NewClaudeCLILLM(MaxCharsPerChapter, QuizOptionCount)
	service := app.NewQuizService(
		relationalStore,
		vectorStore,
		openAILLM,
		claudeCLILLM,
		app.ServiceConfig{
			ChunkSize:       ChunkSize,
			ChunkOverlap:    ChunkOverlap,
			MaxCharsPerText: MaxCharsPerChapter,
			QuizOptionCount: QuizOptionCount,
		},
	)
	api.NewHandler(service).RegisterRoutes(mux)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Printf("Server läuft auf Port %s\n", port)
	if err := http.ListenAndServe(":"+port, withCORS(mux)); err != nil {
		log.Fatalf("Fehler beim Starten des Servers: %v", err)
	}
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().
			Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
