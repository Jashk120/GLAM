# GLAM Server

Go backend for GLAM scenario generation.

## Run with dotenv

```bash
cp server/.env.example server/.env
# edit server/.env: OPENROUTER_API_KEY=sk-or-...
go run ./...
# or
go build -o /tmp/glam-server ./...
/tmp/glam-server
```

Server auto-loads `.env` via `github.com/joho/godotenv` — checks `GLAM_ROOT/.env` and `GLAM_ROOT/server/.env` (if `GLAM_ROOT` set), plus `server/.env` and `./.env` on startup. No `export` needed. System env still overrides file. Path resolution for `schema/` also follows `GLAM_ROOT` → executable location → cwd (no hardcoded home paths).

Alternatively: `export OPENROUTER_API_KEY=sk-or-...` before launch.

Server listens on `:8080` (override with `PORT` env).

## Env

- `LLM_PROVIDER` (optional — `openrouter`, the default, or `ollama` for a local Ollama model)
- `OPENROUTER_API_KEY` (required for `/api/scenario/generate`) — also accepts `OPENCODE_API_KEY` for backward compat
- `OPENROUTER_ENDPOINT` (default `https://openrouter.ai/api/v1/chat/completions`)
- `OPENROUTER_MODEL` (default `google/gemma-4-31b-it:free`)
- `OPENROUTER_TIMEOUT` (default `60s` — also accepts `OPENCODE_TIMEOUT`)
- `OPENROUTER_MAX_TOKENS` (default `6000`)
- `OPENROUTER_ERROR_PREVIEW_LIMIT` (default `2000`)
- `OPENROUTER_REFERER` (optional — sent as `HTTP-Referer` header for OpenRouter rankings)
- `OPENROUTER_TITLE` (optional — sent as `X-Title` header, default `GLAM`)
- `PORT` (default `8080`)
- `GLAM_ROOT` (optional — absolute or relative path to repo root for schema/registry and `.env` lookup)
- `CORS_ALLOWED_ORIGINS` (optional — comma-separated allowlist; defaults to `http://localhost:5173,http://127.0.0.1:5173,http://localhost:3000,http://127.0.0.1:3000`; header set only when `Origin` matches)

### Local Ollama

Ollama runs locally and does not need an API key. Start Ollama (`ollama serve`, if it is not already running), then set:

```bash
LLM_PROVIDER=ollama
OLLAMA_MODEL=qwen2.5-coder:14b
```

Optional settings: `OLLAMA_ENDPOINT` (default `http://127.0.0.1:11434/api/chat`) and `OLLAMA_TIMEOUT` (default `180s`; accepts a Go duration such as `5m` or seconds). The Teacher interface uses this provider for `POST /api/scenario/generate`; the separate tool-calling `/api/chat` endpoint remains OpenRouter-only.

If a provider's first draft fails the strict scenario validator, GLAM performs one low-temperature repair request containing the rejected JSON and the validation errors. It returns a game only if that repaired JSON also validates. For OpenRouter, this repair request is sent to OpenRouter; for Ollama, it stays on the local machine.

> Legacy `OPENCODE_*` vars still work as fallback but are deprecated — use `OPENROUTER_*`.

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

## LLM Provider

The default provider is **OpenRouter** (`https://openrouter.ai/api/v1/chat/completions`) with model `google/gemma-4-31b-it:free`. Set `LLM_PROVIDER=ollama` to use a locally installed Ollama model instead. OpenRouter requests use OpenAI-compatible `chat/completions` with `response_format: {type:"json_object"}`; Ollama requests use `/api/chat` with `format:"json"`.
