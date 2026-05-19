# Document Quiz

Diese Anleitung beschreibt den **erstmaligen Start** des Projekts auf Linux/macOS (analog auf Windows mit angepassten Befehlen).

## 1. Voraussetzungen

Bitte stelle sicher, dass folgende Tools installiert sind:

- Docker
- Go (empfohlen: 1.23+)
- Node.js (empfohlen: 20+)
- npm

Optional zur Prüfung:

```bash
docker --version
go version
node -v
npm -v
```

## 2. Projekt klonen und ins Verzeichnis wechseln

```bash
git clone <REPO-URL>
cd document-quiz
```

## 3. Abhängigkeiten einmalig installieren

### Backend

```bash
cd backend
go mod download
go mod tidy
cd ..
```

### Frontend

```bash
cd frontend
npm install
cd ..
```

## 4. Projekt starten (empfohlen)

Der schnellste Weg ist das Startskript im Projektroot:

```bash
./start-all.sh
```

Was dabei automatisch passiert:

1. Chroma Vector DB wird als Docker-Container auf Port `8000` gestartet.
2. Backend wird auf Port `8080` gestartet.
3. Frontend wird per `ng serve` auf Port `4200` gestartet.

Danach ist die Anwendung erreichbar unter:

- Frontend: http://localhost:4200
- Backend Health: http://localhost:8080/api/health
- Chroma: http://localhost:8000

Beenden mit `Strg + C` im Terminal, in dem `./start-all.sh` läuft.

## 5. OpenAI API Key setzen

Für Upload/Quiz-Generierung wird ein OpenAI API Key benötigt.

Der Key kann in der Anwendung hinterlegt werden (Settings), alternativ über den Backend-Endpunkt:

```http
POST /api/settings/openai-key
```

## 6. Manueller Start (Alternative)

Falls du die Komponenten einzeln starten möchtest:

### 6.1 Chroma starten

```bash
docker run -d --rm \
	--name document-quiz-chroma \
	-p 8000:8000 \
	-v "$(pwd)/vector_db:/chroma/chroma" \
	chromadb/chroma:latest
```

### 6.2 Backend starten

```bash
cd backend
go run .
```

### 6.3 Frontend starten

```bash
cd frontend
npm run start
```

## 7. Häufige Probleme

- `permission denied` bei `./start-all.sh`:

	```bash
	chmod +x start-all.sh
	```

- Port belegt (`4200`, `8080` oder `8000`):
	Beende den Prozess auf dem Port oder passe Ports in den Startbefehlen an.

- Docker läuft nicht:
	Docker Daemon starten und erneut versuchen.

- Frontend kann Backend nicht erreichen:
	Prüfen, ob Backend unter `http://localhost:8080/api/health` antwortet.

## 8. Verzeichnisüberblick

- `backend/` Go API
- `frontend/` Angular UI
- `vector_db/` Persistente Chroma-Daten
- `start-all.sh` Startet Chroma + Backend + Frontend gemeinsam
