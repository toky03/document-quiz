package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"document-quiz-backend/app"
)

type Handler struct {
	service app.APIPort
}

func NewHandler(service app.APIPort) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/settings/openai-key-status", h.handleAPIKeyStatus)
	mux.HandleFunc("POST /api/settings/openai-key", h.handleSaveAPIKey)
	mux.HandleFunc("DELETE /api/settings/openai-key", h.handleClearAPIKey)
	mux.HandleFunc("GET /api/settings/provider", h.handleGetProvider)
	mux.HandleFunc("POST /api/settings/provider", h.handleSetProvider)
	mux.HandleFunc("POST /api/upload", h.handleFileUpload)
	mux.HandleFunc("GET /api/chapters", h.handleGetChapters)
	mux.HandleFunc("DELETE /api/chapters/{id}", h.handleDeleteChapter)
	mux.HandleFunc("GET /api/chapters/{id}/questions", h.handleGetChapterQuestions)
	mux.HandleFunc("POST /api/quiz/submit", h.handleQuizSubmit)
}

type SaveAPIKeyRequest struct {
	APIKey string `json:"api_key"`
}

type SetProviderRequest struct {
	Provider string `json:"provider"`
}

type QuizSubmitRequest struct {
	ChapterID int     `json:"chapter_id"`
	Answers   [][]int `json:"answers"`
}

func (h *Handler) handleFileUpload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{"error": "Fehler beim Lesen der Dateiinformationen"},
		)
		return
	}

	model := r.FormValue("model")
	apiKey := strings.TrimSpace(r.FormValue("api_key"))
	files := r.MultipartForm.File["files"]

	uploadedFiles := make([]app.UploadedFile, 0, len(files))
	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			writeJSON(
				w,
				http.StatusBadRequest,
				map[string]string{"error": "Datei konnte nicht geöffnet werden"},
			)
			return
		}
		content, err := io.ReadAll(file)
		_ = file.Close()
		if err != nil {
			writeJSON(
				w,
				http.StatusBadRequest,
				map[string]string{"error": "Dateiinhalt konnte nicht gelesen werden"},
			)
			return
		}
		uploadedFiles = append(
			uploadedFiles,
			app.UploadedFile{Name: fileHeader.Filename, Content: content},
		)
	}

	result, err := h.service.UploadDocuments(r.Context(), app.UploadCommand{
		Model:  model,
		APIKey: apiKey,
		Files:  uploadedFiles,
	})
	if err != nil {
		if isBadRequestError(err) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": toUserError(err)})
			return
		}
		writeJSON(
			w,
			http.StatusInternalServerError,
			map[string]string{"error": "Verarbeitung fehlgeschlagen"},
		)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message":            "Verarbeitung abgeschlossen",
		"processed_files":    result.ProcessedFiles,
		"successful_files":   result.SuccessfulFiles,
		"failed_files":       result.FailedFiles,
		"error_count":        result.ErrorCount,
		"issues":             result.Issues,
		"generated_chapters": result.GeneratedChapters,
		"generated_pairs":    result.GeneratedPairs,
		"total_chunks":       result.TotalChunks,
	})
}

func (h *Handler) handleClearAPIKey(w http.ResponseWriter, r *http.Request) {
	if err := h.service.ClearAPIKey(r.Context()); err != nil {
		writeJSON(
			w,
			http.StatusInternalServerError,
			map[string]string{"error": "API Key konnte nicht gelöscht werden"},
		)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleSaveAPIKey(w http.ResponseWriter, r *http.Request) {
	var req SaveAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Ungültige Anfrage"})
		return
	}

	if err := h.service.SaveAPIKey(r.Context(), req.APIKey); err != nil {
		if isBadRequestError(err) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": toUserError(err)})
			return
		}
		writeJSON(
			w,
			http.StatusInternalServerError,
			map[string]string{"error": "API Key konnte nicht gespeichert werden"},
		)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"saved": true})
}

func (h *Handler) handleGetProvider(w http.ResponseWriter, r *http.Request) {
	provider, err := h.service.GetProvider(r.Context())
	if err != nil {
		writeJSON(
			w,
			http.StatusInternalServerError,
			map[string]string{"error": "Provider konnte nicht geladen werden"},
		)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"provider": provider})
}

func (h *Handler) handleSetProvider(w http.ResponseWriter, r *http.Request) {
	var req SetProviderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Ungültige Anfrage"})
		return
	}

	if err := h.service.SetProvider(r.Context(), req.Provider); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": toUserError(err)})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"provider": req.Provider})
}

func (h *Handler) handleAPIKeyStatus(w http.ResponseWriter, r *http.Request) {
	isSaved, err := h.service.IsAPIKeySaved(r.Context())
	if err != nil {
		writeJSON(
			w,
			http.StatusInternalServerError,
			map[string]string{"error": "Status konnte nicht geladen werden"},
		)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"is_saved": isSaved})
}

func (h *Handler) handleDeleteChapter(w http.ResponseWriter, r *http.Request) {
	chapterID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Ungültige Kapitel-ID"})
		return
	}

	if err := h.service.DeleteChapter(r.Context(), chapterID); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "nicht gefunden") {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Kapitel nicht gefunden"})
			return
		}
		writeJSON(
			w,
			http.StatusInternalServerError,
			map[string]string{"error": "Kapitel konnte nicht gelöscht werden"},
		)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleGetChapters(w http.ResponseWriter, r *http.Request) {
	chapters, err := h.service.ListChapters(r.Context())
	if err != nil {
		writeJSON(
			w,
			http.StatusInternalServerError,
			map[string]string{"error": "Fehler beim Abrufen der Kapitel"},
		)
		return
	}

	response := make([]map[string]any, 0, len(chapters))
	for _, ch := range chapters {
		response = append(response, map[string]any{
			"id":             ch.ID,
			"title":          ch.Title,
			"source_name":    ch.SourceName,
			"question_count": ch.QACount,
		})
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) handleGetChapterQuestions(w http.ResponseWriter, r *http.Request) {
	chapterID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Ungültige Kapitel-ID"})
		return
	}

	questions, err := h.service.GetChapterQuestions(r.Context(), chapterID)
	if err != nil {
		writeJSON(
			w,
			http.StatusInternalServerError,
			map[string]string{"error": "Fehler beim Abrufen der Fragen"},
		)
		return
	}

	writeJSON(w, http.StatusOK, questions)
}

func (h *Handler) handleQuizSubmit(w http.ResponseWriter, r *http.Request) {
	var req QuizSubmitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Ungültige Anfrage"})
		return
	}

	result, err := h.service.SubmitQuiz(r.Context(), req.ChapterID, req.Answers)
	if err != nil {
		writeJSON(
			w,
			http.StatusInternalServerError,
			map[string]string{"error": "Fehler beim Abrufen der Fragen"},
		)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func isBadRequestError(err error) bool {
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(msg, "erforderlich") ||
		strings.Contains(msg, "keine dateien") ||
		strings.Contains(msg, "api key darf nicht leer") ||
		strings.Contains(msg, "kein openai api key")
}

func toUserError(err error) string {
	msg := strings.TrimSpace(err.Error())
	if msg == "" {
		return "Ungültige Anfrage"
	}
	return strings.ToUpper(msg[:1]) + msg[1:]
}
