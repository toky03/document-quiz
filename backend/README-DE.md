# Document Quiz - Backend (Go)

## Features

- 🚀 RESTful API mit Gin Framework
- 🗂️ SQLite Datenbank für Kapitel und Q&A
- 🔍 Chroma Vector DB für Vektorsuche
- 🧠 LangChain Go Integration für LLM (OpenAI)
- 📄 PDF-Verarbeitung und Text-Chunking

## Voraussetzungen

- Go 1.21+
- SQLite3 (meist vorinstalliert)
- OpenAI API Key

## Installation

```bash
cd backend
go mod download
go mod tidy
```

## Umgebungsvariablen

Erstelle eine `.env` Datei basierend auf `.env.example`:

```env
PORT=8080
OPENAI_API_KEY=your-api-key-here
CHROMA_URL=http://localhost:8000
SQLITE_DB_PATH=quiz_data.db
```

Starte Chroma lokal (separater Prozess), z. B. mit Docker:

```bash
docker run --rm -p 8000:8000 \
  -v $(pwd)/../vector_db:/chroma/chroma \
  chromadb/chroma:latest
```

Wichtig: Durch das Volume wird die bestehende Datenbank im Ordner `vector_db` wiederverwendet.

## Development

```bash
go run .
```

Server läuft auf http://localhost:8080

## Build

```bash
go build -o document-quiz-backend
./document-quiz-backend
```

## Docker

```bash
docker build -t document-quiz-backend .
docker run -p 8080:8080 \
  -e OPENAI_API_KEY=your-api-key \
  -e CHROMA_URL=http://<dein-chroma-host>:8000 \
  document-quiz-backend
```

## API Endpunkte

### Health Check
```
GET /api/health
```

### Kapitel abrufen
```
GET /api/chapters
```

Response:
```json
[
  {
    "id": 1,
    "title": "Chapter 1",
    "source_name": "file.pdf",
    "question_count": 20
  }
]
```

### Fragen eines Kapitels abrufen
```
GET /api/chapters/:id/questions
```

### Quiz absenden
```
POST /api/quiz/submit
Content-Type: application/json

{
  "chapter_id": 1,
  "answers": [[0], [1, 2], [0]]
}
```

### Dateien hochladen
```
POST /api/upload
Content-Type: multipart/form-data

Files: [pdf1, pdf2, ...]
model: gpt-4-mini
api_key: sk-...
```

## Datenbank

### Struktur

**chapters** Tabelle:
- id (PRIMARY KEY)
- title
- source_name (UNIQUE)
- source_type (pdf, image)
- created_at

**qa_pairs** Tabelle:
- id (PRIMARY KEY)
- chapter_id (FOREIGN KEY)
- question
- quiz_type (single, multiple)
- options_json
- correct_options_json
- answer
- created_at

## TODO: Zu implementieren

- [ ] PDF-Textextraktion (verwende pdfcpu)
- [ ] LangChain Go LLM-Integration
- [ ] Chroma Vector DB Client
- [ ] Error Handling verbessern
- [ ] Logging mit logrus
- [ ] Unit Tests
- [ ] API Dokumentation (Swagger)

## CORS

CORS ist für alle Origins aktiviert. In Production sollte das eingeschränkt werden.
