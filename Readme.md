# Document Quiz

Diese Anleitung beschreibt den **erstmaligen Start** des Projekts auf Linux/macOS (analog auf Windows mit angepassten Befehlen).

## 1. Voraussetzungen

Bitte stelle sicher, dass folgende Tools installiert sind:

- Docker
- Go 1.24+
- Node.js 20.19+ oder 22.12+ (Angular 21)
- npm
- Optional: Claude Code CLI (`claude`) — nur wenn der Anbieter `claude_cli` verwendet werden soll (siehe Abschnitt 5)

Statt manuell zu prüfen kannst du `./doctor.sh` ausführen — das Skript meldet
fehlende oder zu alte Versionen.

```bash
./doctor.sh
```

## 2. Projekt klonen und ins Verzeichnis wechseln

```bash
git clone <REPO-URL>
cd document-quiz
```

## 3. Abhängigkeiten einmalig installieren

Der schnellste Weg ist das Setup-Skript im Projektroot:

```bash
./install.sh
```

Es ruft zuerst `./doctor.sh` auf, lädt anschließend Backend- und
Frontend-Abhängigkeiten, zieht das Chroma-Docker-Image vor und baut das
Backend-Binary.

Manuell (Alternative):

### Backend

```bash
cd backend
go mod download
cd ..
```

### Frontend

```bash
cd frontend
npm ci
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

## 5. LLM-Anbieter wählen

Im Upload-Dialog gibt es eine Auswahl zwischen zwei Anbietern:

### OpenAI (Standard)

Benötigt einen OpenAI API Key. Der Key kann direkt in der UI hinterlegt
werden, alternativ per Backend-Endpunkt:

```http
POST /api/settings/openai-key
```

In diesem Modus werden zusätzlich Embeddings in Chroma gespeichert
(ebenfalls über die OpenAI-API).

### Claude CLI (lokal, ohne API Key)

Verwendet die lokal installierte `claude` CLI. Voraussetzungen:

- `claude` ist auf `PATH` (`brew install claude` o. ä.)
- `claude login` wurde einmalig ausgeführt (z. B. mit einem Max-Abo)

In diesem Modus wird **kein API Key** benötigt und Chroma wird **nicht**
beschrieben — das Backend ruft `claude -p --output-format json` auf und
gibt die generierten Fragen direkt in SQLite.

Der aktive Anbieter wird im Setting `llm_provider` (Werte: `openai`,
`claude_cli`) gespeichert. Endpunkte:

```http
GET  /api/settings/provider
POST /api/settings/provider     {"provider":"openai"|"claude_cli"}
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

Bei Problemen zuerst `./doctor.sh` ausführen — meldet fehlende oder
inkompatible Versionen von Node, Go und Docker.

- `permission denied` bei `./start-all.sh` / `./install.sh` / `./doctor.sh`:

	```bash
	chmod +x start-all.sh install.sh doctor.sh
	```

- Port belegt (`4200`, `8080` oder `8000`):
	Beende den Prozess auf dem Port oder passe Ports in den Startbefehlen an.

- Docker läuft nicht:
	Docker Daemon starten und erneut versuchen.

- Frontend kann Backend nicht erreichen:
	Prüfen, ob Backend unter `http://localhost:8080/api/health` antwortet.

- `claude CLI nicht gefunden` beim Upload mit Anbieter `claude_cli`:
	`claude` installieren und `claude login` ausführen, dann das Backend neu starten.

- `node`/`npm` nicht gefunden, obwohl nvm installiert ist:
	`doctor.sh` und `install.sh` sourcen nvm automatisch aus den üblichen Pfaden
	(`~/.nvm/nvm.sh`, `/opt/homebrew/opt/nvm/nvm.sh`,
	`/usr/local/opt/nvm/nvm.sh`). Wenn nvm an einem anderen Ort liegt,
	`nvm alias default <version>` setzen oder `NVM_DIR` exportieren.

## 8. Verzeichnisüberblick

- `backend/` Go API
- `frontend/` Angular UI
- `vector_db/` Persistente Chroma-Daten (nur Anbieter `openai`)
- `doctor.sh` Read-only Umgebungs-Check
- `install.sh` Setup-Skript (Dependencies, Docker-Image, Backend-Binary)
- `start-all.sh` Startet Chroma + Backend + Frontend gemeinsam
