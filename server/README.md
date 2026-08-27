# GLAM Server

Go backend for GLAM scenario generation.

## Run with dotenv

```bash
cp server/.env.example server/.env
# edit server/.env: OPENCODE_API_KEY=sk-...
go run ./...
# or
go build -o /tmp/glam-server ./...
/tmp/glam-server
```

Server auto-loads `.env` via `github.com/joho/godotenv` — checks `server/.env`, `GLAM/.env`, and `./.env` on startup. No `export` needed. System env still overrides file.

Alternatively: `export OPENCODE_API_KEY=sk-...` before launch.

Server listens on `:8080` (override with `PORT` env).

## Env

- `OPENCODE_API_KEY` (required for `/api/scenario/generate`)
- `OPENCODE_ENDPOINT` (default `https://opencode.ai/zen/go/v1/responses`)
- `OPENCODE_MODEL` (default `muse-spark-1.2-contributor`)
- `PORT` (default `8080`)

Do not commit `.env` — `.gitignore` handles it.

## Endpoints

- `GET /health` → `{"status":"ok"}`
- `GET /api/assets` → registry JSON array
- `POST /api/scenario/generate` body `{"prompt":"..."}` → `{"scenario":{...}}` or `{"error":"...", "details":[...]}` (400 validation, 502 LLM failure, 500 missing key)
- `GET /api/scenario/validate` → validates `scenarios/example.json`
- `POST /api/scenario/validate` body scenario or `{"scenario":{...}}` → `{"valid":true}` or 400 with details
- CORS enabled for `http://localhost:5173` (Vite proxy `/api` → `:8080`)

## Validation pipeline

JSON parse → JSON Schema (santhosh-tekuri/jsonschema) → asset-ID → activity-ID → reference/position/bounds/duplicate/forbidden fields
