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

Server auto-loads `.env` via `github.com/joho/godotenv` — checks `GLAM_ROOT/.env` and `GLAM_ROOT/server/.env` (if `GLAM_ROOT` set), plus `server/.env` and `./.env` on startup. No `export` needed. System env still overrides file. Path resolution for `schema/` also follows `GLAM_ROOT` → executable location → cwd (no hardcoded home paths).

Alternatively: `export OPENCODE_API_KEY=sk-...` before launch.

Server listens on `:8080` (override with `PORT` env).

## Env

- `OPENCODE_API_KEY` (required for `/api/scenario/generate`)
- `OPENCODE_ENDPOINT` (default `https://opencode.ai/zen/go/v1/responses`)
- `OPENCODE_MODEL` (default `muse-spark-1.2-contributor`)
- `PORT` (default `8080`)
- `GLAM_ROOT` (optional — absolute or relative path to repo root for schema/registry and `.env` lookup)
- `CORS_ALLOWED_ORIGINS` (optional — comma-separated allowlist; defaults to `http://localhost:5173,http://127.0.0.1:5173,http://localhost:3000,http://127.0.0.1:3000`; header set only when `Origin` matches)

Do not commit `.env` — `.gitignore` handles it.

## Endpoints

- `GET /health` → `{"status":"ok"}`
- `GET /api/assets` → registry JSON array
- `POST /api/scenario/generate` body `{"prompt":"..."}` → `{"scenario":{...}}` or `{"error":"...", "details":[...]}` (400 validation, 502 LLM failure, 500 missing key)
- `GET /api/scenario/validate` → validates `scenarios/example.json`
- `POST /api/scenario/validate` body scenario or `{"scenario":{...}}` → `{"valid":true}` or 400 with details
- CORS enabled only for origins in `CORS_ALLOWED_ORIGINS` (defaults to `http://localhost:5173,http://127.0.0.1:5173,http://localhost:3000,http://127.0.0.1:3000`; Vite `5173` + Next `3000` via `/api` → `:8080`)

## Validation pipeline

JSON parse → JSON Schema (santhosh-tekuri/jsonschema) → asset-ID → activity-ID → reference/position/bounds/duplicate/forbidden fields
