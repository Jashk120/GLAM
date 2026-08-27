# GLAM — Agents Guide

> One engine — any lesson. Scenarios are just JSON.

## Overview

GLAM generates playable learning worlds from teacher prompts. Teacher → Go → Muse Spark (OpenCode) → validated Scenario JSON → Phaser runtime. Local JSON/files only — no DB/Redis/multiplayer in MVP.

Stack: **TypeScript + Phaser 3.80 + Vite 5** (client) + **Go 1.22** (server) + **OpenCode Go → muse-spark-1.2-contributor** via `https://opencode.ai/zen/go/v1/responses` + JSON Schema (2020-12) + asset registry.

## Repo Structure (mono-repo root = `GLAM/`)

```
GLAM/
├── client/                 # Vite + Phaser runtime (data-driven, no LLM calls)
│   ├── src/engine/         # ScenarioLoader, AssetResolver, Game (Phaser.Scene), MissionManager
│   ├── src/world/          # WorldRenderer (TILE=32, theme colors)
│   ├── src/entities/       # Character/Building/ObjectRenderer
│   ├── src/interactions/   # InteractionManager (proximity ≤1, auto vs E, cooldowns)
│   ├── src/activities/     # Dialogue, MCQ, Math, Shop, Information (+ index.ts)
│   ├── src/assets/         # registry.json (21) + assetRegistry.ts + AssetStreamer.ts
│   ├── src/teacher/        # TeacherUI.ts (POST /api/scenario/generate, assetStatus)
│   ├── src/types/scenario.ts
│   ├── index.html / vite.config.ts (proxy /api → :8080) / tsconfig.json
│   └── public/scenarios/example.json
├── server/                 # Go — key stays server-side only
│   ├── main.go             # :8080, CORS 5173, dotenv, schema/registry resolve, air
│   ├── api/handler.go      # POST /api/scenario/generate, GET /api/assets, /health, /validate
│   ├── llm/client.go       # OpenCode Responses endpoint, bearer auth, fence stripping
│   ├── llm/prompt.go       # system prompt: schema + registry + 5 types
│   ├── scenario/validator.go # JSON parse → Schema → asset-ID → activity → bounds/duplicate
│   ├── scenario/types.go / registry.go
│   ├── .air.toml           # air hot-reload (watch go+json)
│   └── .env.example        # never commit .env
├── schema/
│   ├── scenario.schema.json  # strict, additionalProperties:false, forbids code/script/component/bundle
│   ├── asset-registry.json   # 21 entries (identical copy in client/src/assets/registry.json)
│   └── README.md
├── scenarios/example.json  # Money Management Town (15×12, 5 interaction types)
├── dev.sh                  # one-command dev (see below)
└── .gitignore              # ignores .env, server/tmp, client/dist, node_modules, .omo, demo htmls
```

Demo HTMLs at root (`pokemon-style-demo.html`, `scenario-engine-demo.html`) are **prototypes, gitignored** — not part of MVP.

## Dev — One Command

```bash
cp server/.env.example server/.env  # set OPENCODE_API_KEY=sk-...
./dev.sh
# → Go air hot-reload on :8080 + Vite HMR on :5173
# → http://localhost:5173 (client) http://localhost:8080/health
# Ctrl+C kills both; logs: tail -f /tmp/glam-server.log /tmp/glam-client.log
```

- **Client HMR:** Vite watches `client/src/**` → instant browser update.
- **Server HMR:** `air` watches `server/**/*.go` + `**/*.json` (800ms delay, 400ms kill_delay) → rebuilds `server/tmp/main` → restart. Config in `server/.air.toml` (`env_files=[".env"]`). Fallback to `go run .` if `air` missing (`go install github.com/air-verse/air@latest`).
- Vite proxy: `vite.config.ts` → `/api` → `http://localhost:8080`.

Manual:
```bash
# server
go vet ./... && go build -o /tmp/glam-server . && PORT=8080 ./tmp/glam-server
# client
npm install && npm run dev -- --port 5173 --host
npm run build # tsc && vite build → dist/
```

## Scenario Contract

- **Schema** `schema/scenario.schema.json`: `id/title/version/world{template:town|forest|desert|school, spawn{x,y 0-30}, size{cols 8-30, rows 8-20}, regions?} + characters[] + buildings[]{typeAssetId} + objects[]{assetId} + missions[]{trigger?, checkAtEnd?}`. Interactions `oneOf` 5:
  - `dialogue{ text 1-1000, speaker? }`
  - `mcq{ question, options 2-5 {text,correct,explanation?}, allowRetry? }`
  - `math{ question, answer number|string, tolerance?, hint? }`
  - `shop{ currency=coins, items 1-10 {name,price≥0,icon?} }`
  - `information{ content, title?, image? }`
  Common: `cooldown?`, `auto?`, `onCorrect?/onWrong?{stat?,delta?,toast?}`. All levels `propertyNames` forbid `code/script/component/bundle`.
- **Registry** `schema/asset-registry.json` ≡ `client/src/assets/registry.json` — 21 IDs: `shop_small`, `bank`, `character_teacher`, `tile_grass` etc (`type: building|character|object|tile|prop`, `bundle: town|forest|school|common`, `icon`, `solid`). Keep copies in sync (`diff` them). Scenario may only reference these IDs.
- **Example** `scenarios/example.json` demonstrates all 5 interactions; validate via Go validator or `python3 -m jsonschema -i scenarios/example.json schema/scenario.schema.json`.

## Validation Pipeline (Go)

`JSON parse → JSON Schema (santhosh-tekuri/jsonschema/v5 Draft2020) → asset-ID existence → activity-ID (5 types) → reference/position (inside world.size, no duplicates, mission triggers) → forbid code fields`. Reject → `400 {error, details[]}`; missing key → `500`; LLM failure → `502`. Client also does light validation in `ScenarioLoader` before `Game.create`.

## LLM Flow

`TeacherUI POST {prompt} → server/api/handler.go → llm/client.go Generate(systemPrompt+schema+registry) → POST https://opencode.ai/zen/go/v1/responses {model:muse-spark-1.2-contributor, input:[system,user], stream:false, text:{format:{type:json_object}}} Bearer $OPENCODE_API_KEY → strip ```json fences → validate → {scenario} → client loadScenario → Phaser`.

Key **never leaves server** — `main.go:init()` loads `.env` via `joho/godotenv` from `server/.env`, `GLAM/.env`, `./.env` (system env overrides). **Agents: never read `.env` files** — use `.env.example` for shape.

## Asset Streaming (Phase 8)

`AssetStreamer.ts`: `getRequiredAssetIds(scenario) → checkCache(Map+localStorage glam_asset_cache_v1) → fetchMissing(sim 50ms, ready for bundle URLs) → preloadScenarioAssets()` called in `ScenarioLoader`. UI `#assetStatus` shows `Assets cached: 21/21`.

## Conventions

- **250 LOC ceiling** per file — split logically.
- **No `as any` / `@ts-ignore` / `@ts-expect-error`** — strict `tsconfig.json` (`strict:true`).
- **No empty catch**, no `as any` in Go (`interface{}` only where justified).
- Engine **data-driven** — no hardcoded scenario content in `Game`/`WorldRenderer`.
- **No DB/Redis/multiplayer** in MVP — local JSON/files.
- Commits: atomic, no `commit --no-verify` without ask, never commit secrets.
- Vite `build` + `go vet` + `go build` must be clean before marking todo done.
- Expose `window.Glam.loadScenario` for dev; keep `Game.ts` API stable.
- Styles: dark `#181828`, gold `#ffd700`, emoji fallback sprites until bundles.

## Security / Secrets

- Never read `.env` or `server/.env` — treat as secret. Agents ask user to set `OPENCODE_API_KEY` locally.
- Validator rejects arbitrary component names / executable code fields.
- CORS allow only `http://localhost:5173` (+ `127.0.0.1:5173`).

## Useful Paths

- Health: `GET /health → {"status":"ok"}`
- Assets: `GET /api/assets`
- Generate: `POST /api/scenario/generate` + Validate: `GET|POST /api/scenario/validate`
- Dist not committed; `client/dist/` built via `npm run build`.
