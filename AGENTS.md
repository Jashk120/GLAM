# GLAM — Agents Guide

> One engine — any lesson. Scenarios are just JSON.

## General Principles

- Make the smallest correct change.
- Understand the existing architecture before modifying it.
- Match the existing coding style.
- Do not introduce unnecessary dependencies.
- Do not rewrite unrelated code.
- Leave the repository in a buildable state.

## Before Implementing Any Plan

1. Read the plan document fully.
2. Read the relevant source files.
3. Cross-check the plan's assumptions against the actual code.
4. List any mismatches, ambiguities, or unresolved questions and raise them
   to the user before writing a single line of implementation.
5. Do not resolve open questions by reasoning through them independently.
   Ask instead.

## During Implementation

- Treat decisions stated in the plan as closed. Do not re-examine them.
- If you encounter something the plan did not anticipate, stop and ask.
- Do not think through alternatives silently — surface them immediately.

---

## Overview

GLAM generates playable learning worlds from teacher prompts. Teacher → Go → Muse Spark (OpenCode) → validated Scenario JSON → Phaser runtime. Local JSON/files only — no DB/Redis/multiplayer in MVP.

Stack: **TypeScript + Phaser 3.80 + Vite 5** (client) + **Next.js 16 + React 19** (teacher-interface) + **Go 1.22** (server) + **OpenCode/OpenRouter** LLM backend + JSON Schema (2020-12) + asset registry.

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
│   └── public/scenarios/example.json / README.md
├── teacher-interface/       # Next.js 16 + React 19 app for teacher-facing UI
│   ├── app/ / components/ / lib/ / public/
│   ├── eslint.config.mjs / components.json / next.config.ts
│   └── README.md
├── server/                 # Go — key stays server-side only
│   ├── main.go             # :8080, CORS 5173, dotenv, schema/registry resolve, air
│   ├── api/handler.go      # POST /api/scenario/generate, GET /api/assets, /health, /validate
│   ├── llm/client.go       # OpenRouter chat completions endpoint, bearer auth, fence stripping
│   ├── llm/prompt.go       # system prompt: schema + registry + 5 types
│   ├── scenario/validator.go # JSON parse → Schema → asset-ID → activity → bounds/duplicate
│   ├── scenario/types.go / registry.go / normalize.go / interaction.go
│   ├── .air.toml           # air hot-reload (watch go+json)
│   └── .env.example        # never commit .env
├── schema/
│   ├── scenario.schema.json  # strict, additionalProperties:false, forbids code/script/component/bundle
│   ├── asset-registry.json   # 21 entries (identical copy in client/src/assets/registry.json)
│   └── README.md
├── scenarios/example.json  # Money Management Town (15×12, 5 interaction types)
├── dev.sh                  # one-command dev (see below)
├── Makefile                 # registry-check / registry-sync / vet / build
└── .gitignore               # ignores .env, server/tmp, client/dist, node_modules, .omo, demo htmls
```

Demo HTMLs at root (`pokemon-style-demo.html`, `scenario-engine-demo.html`) are **prototypes, gitignored** — not part of MVP.

## Dev — One Command

```bash
cp server/.env.example server/.env  # set OPENROUTER_API_KEY=sk-...
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
cd client && npm install && npm run dev -- --port 5173 --host
npm run build # tsc && vite build → dist/
# teacher-interface
cd teacher-interface && npm install && npm run dev
npm run lint && npm run build
```

## Scenario Contract

- **Schema** `schema/scenario.schema.json`: `id/title/version/world{template:town|forest|desert|school, spawn{x,y 0-30}, size{cols 8-30, rows 8-20}, regions?} + characters[] + buildings[]{typeAssetId} + objects[]{assetId} + missions[]{trigger?, checkAtEnd?}`. Interactions `oneOf` 5:
  - `dialogue{ text 1-1000, speaker? }`
  - `mcq{ question, options 2-5 {text,correct,explanation?}, allowRetry? }`
  - `math{ question, answer number|string, tolerance?, hint? }`
  - `shop{ currency=coins, items 1-10 {name,price≥0,icon?} }`
  - `information{ content, title?, image? }`
  Common: `cooldown?`, `auto?`, `onCorrect?/onWrong?{stat?,delta?,toast?}`. All levels `propertyNames` forbid `code/script/component/bundle`.
- **Registry** `schema/asset-registry.json` ≡ `client/src/assets/registry.json` — 21 IDs: `shop_small`, `bank`, `character_teacher`, `tile_grass` etc (`type: building|character|object|tile|prop`, `bundle: town|forest|school|common`, `icon`, `solid`). Keep copies in sync (`make registry-check` / `make registry-sync`). Scenario may only reference these IDs.
- **Example** `scenarios/example.json` demonstrates all 5 interactions; validate via Go validator or `python3 -m jsonschema -i scenarios/example.json schema/scenario.schema.json`.

## Validation Pipeline (Go)

`JSON parse → JSON Schema (santhosh-tekuri/jsonschema/v5 Draft2020) → asset-ID existence → activity-ID (5 types) → reference/position (inside world.size, no duplicates, mission triggers) → forbid code fields`. Reject → `400 {error, details[]}`; missing key → `500`; LLM failure → `502`. Client also does light validation in `ScenarioLoader` before `Game.create`.

## LLM Flow

`TeacherUI POST {prompt} → server/api/handler.go → llm/client.go Generate(systemPrompt+schema+registry) → POST OpenRouter chat/completions {model, messages:[system,user], response_format:{type:json_object}} Bearer $OPENROUTER_API_KEY (falls back to $OPENCODE_API_KEY) → strip \`\`\`json fences → validate → {scenario} → client loadScenario → Phaser`.

Key **never leaves server** — `main.go:init()` loads `.env` via `joho/godotenv` from `server/.env`, `GLAM/.env`, `./.env` (system env overrides). **Agents: never read `.env` files** — use `.env.example` for shape.

## Asset Streaming (Phase 8)

`AssetStreamer.ts`: `getRequiredAssetIds(scenario) → checkCache(Map+localStorage glam_asset_cache_v1) → fetchMissing(sim 50ms, ready for bundle URLs) → preloadScenarioAssets()` called in `ScenarioLoader`. UI `#assetStatus` shows `Assets cached: 21/21`.

---

# Formatting

**Go (`server/`):**
```bash
cd server && gofmt -l .   # list files needing formatting
cd server && gofmt -w .   # apply
```
Run before considering Go work complete. `go vet` (see Linting) does not format for you.

**TypeScript — client:** no formatter is configured (no Prettier). Rely on `tsc` (strict mode) for correctness; match surrounding style by hand.

**TypeScript/React — teacher-interface:** no Prettier configured either; ESLint (`eslint.config.mjs`, Next.js flat config) covers style-adjacent issues — see Linting.

Do not add Prettier, a different Go formatter, or reformat unrelated files as a side effect of an unrelated change.

---

# Linting

```bash
# server (Go)
cd server && go vet ./...

# client (type-check; no ESLint configured here)
cd client && npx tsc --noEmit

# teacher-interface
cd teacher-interface && npm run lint
```

Also: `make vet` from repo root runs `go vet ./...` + `cd client && npx tsc --noEmit` in one step.

Fix warnings whenever practical. Do not add lint-suppression comments (`//nolint`, `eslint-disable`) to silence a real issue — fix it or ask.

---

# Testing

**There is currently no automated test suite in this repo** — no `*_test.go` files under `server/`, no test script in `client/package.json` or `teacher-interface/package.json`. Do not claim "tests pass" — there is nothing to run yet.

If you add tests:
```bash
# server
cd server && go test ./...          # whole package
cd server && go test ./scenario/... -run TestName   # smallest relevant test first
```
For `client`/`teacher-interface`, no test runner is wired up — adding one (e.g. Vitest) is a dependency decision, raise it with the user first per the Dependencies section below rather than adding it unilaterally.

Before finishing larger changes, run whatever test suite exists for the packages you touched; state plainly if none exists for that area.

---

# Git

Every commit must be signed.

```bash
git commit -s -S -m "commit message"
```

- `-s` → Developer Certificate of Origin (Signed-off-by)
- `-S` → GPG signing

Never create unsigned commits unless explicitly requested. Write commit messages in the imperative mood ("Fix plot-bounds check for forest clearings", "Add tool-calling support to llm client").

Commits: atomic, no `commit --no-verify` without asking first, never commit secrets (`.env`, API keys).

---

# Code Style

Prefer:

- Small functions, **250 LOC ceiling per file** — split logically (existing convention; `server/api/handler.go` at 600 lines is already over this and should not grow further without splitting).
- Descriptive names.
- Early returns.
- Immutable variables where possible.

Avoid:

- Large nested blocks.
- Unnecessary cloning/copying.
- Magic numbers (e.g. registry counts, world bounds — use the named constants already in `world`/`scenario` packages).
- Panics in Go library code unless truly unrecoverable and justified in a comment.

Language-specific, already enforced in this repo:
- **No `as any` / `@ts-ignore` / `@ts-expect-error`** in TypeScript — strict `tsconfig.json` (`strict: true`) in both `client` and `teacher-interface`.
- **No empty `catch`** blocks (TS) and **no silently ignored errors** in Go (`_ = err` only when the error is genuinely inconsequential and that's stated in a comment).
- `interface{}` in Go only where justified (e.g. dynamic JSON handling in `scenario/validator.go`) — not as a generic escape hatch.
- Engine stays **data-driven** — no hardcoded scenario content in `Game`/`WorldRenderer`.

---

# Documentation

When changing a public API (Go exported functions/types, HTTP endpoints, the scenario schema, the asset registry shape):

- Update the relevant `README.md`.
- Update `scenarios/example.json` if the contract itself changed.
- Keep comments synchronized with implementation.

Every top-level project directory (`client/`, `server/`, `schema/`, `teacher-interface/`) already has a `README.md` describing its role — keep these current. If a new top-level directory or a substantial new Go package (e.g. a future `server/pipeline/`, `server/agent/`) is added, give it a `README.md` describing its role in the workspace, same pattern as the existing four.

---

# Dependencies

Before adding a new dependency:

- Check whether the functionality already exists in the project (`server/go.mod` currently has exactly two: `santhosh-tekuri/jsonschema/v5`, `joho/godotenv` — this repo intentionally stays minimal).
- Prefer the standard library (Go) or what's already a dependency of `client`/`teacher-interface`.
- Keep dependency count minimal — raise new additions with the user before adding, don't add speculatively.

---

# Contracts / Wire Formats

The external contract in this repo is **not protobuf** — it's `schema/scenario.schema.json` (JSON Schema 2020-12), enforced identically at generation time (`scenario.ValidateScenario`) and consumed by the Phaser client (`ScenarioLoader`). Any new client-facing surface (a new HTTP endpoint, a new field, a new interaction type) is exactly as load-bearing as a wire format change would be in a protobuf-based system.

Before changing `schema/scenario.schema.json`, `schema/asset-registry.json`, or adding/changing an HTTP endpoint's request/response shape (e.g. `/api/scenario/generate`, `/api/assets`, or any new endpoint): **confirm the schema and its scope with the user first** — do not decide the contract independently. The `additionalProperties:false` + `code/script/component/bundle` blocklist at every schema level is a deliberate security boundary, not incidental strictness — don't loosen it without asking.

---

# Performance

Avoid unnecessary:

- Allocations and copying, especially of the schema/registry JSON (`server/api/handler.go` already loads and holds `SchemaJSON`/`RegistryJSON` once at startup — don't re-read files per request).
- Re-compiling the JSON Schema per request — compile once, reuse the compiled instance (as `scenario/validator.go` already does).
- Cloning large scenario objects in Go where a pointer/reference suffices.
- Locking and heap churn on hot paths (validation, LLM request building).

Consider algorithmic complexity (e.g. entity/plot bounds checks) before micro-optimizing.

---

# Safety

Never disable:

- `go vet` findings, TypeScript strict-mode errors, or ESLint rules — fix or ask, don't suppress.
- The schema's `additionalProperties:false` / forbidden-field checks (see Contracts above).

Do not use:

```go
panic(...)          // in library code, unless failure is truly unrecoverable
```
without explicit justification in a comment. Handle and propagate errors (`error` return values) instead. In TypeScript, no `as any`/`@ts-ignore` — see Code Style.

---

# Before Finishing — CI Parity (MUST NOT FAIL IN CI)

CI is `.github/workflows/registry-sync.yml` and runs exactly:
`make registry-check` → `make vet` → `make build` (plus `go test ./...` when tests exist).
`make vet` = `gofmt -l` (fmt-check) + `go vet ./...` (Go's clippy) + `tsc --noEmit` + `eslint`.
Never finish work that would fail these — always run them locally first.

```bash
# EXACT CI COMMANDS — run every time before you consider work done:
make registry-check
make vet          # gofmt + go vet (clippy equivalent) + tsc + lint — must be 0
make build        # go build + vite build + next build — must be 0
go test ./...     # from server/ — run even if you think you didn't touch Go; 0 tests = ok
# or, verbosely per-package (same as CI):
cd server && gofmt -l . && go vet ./... && go test ./... && go build -o /tmp/glam-server .
cd client && npx tsc --noEmit && npm run build
cd teacher-interface && npm run lint && npm run build
```

No test suite currently exists (see Testing) — `go test ./...` will report `no test files` which is not a failure; do not claim "tests pass" when nothing ran, just report the output. If one of the above cannot be run in your environment, explicitly state why rather than skipping silently. If you add a test, it becomes part of this gate.

---

# Pull Requests

Before opening a PR:

- Ensure `gofmt`/`tsc`/ESLint (whichever applies) pass.
- Ensure `go vet` / `make vet` passes.
- Ensure `go build` / `npm run build` (client and/or teacher-interface, as touched) passes.
- Run `make registry-check` if the registry was touched.
- Keep commits focused, signed (`-s -S`).
- Avoid unrelated changes.

---

# Communication

Never claim:

- "fixed"
- "works"
- "passes"

unless verified by actually running the relevant command in this repo.

Instead say:

- Verified with `go vet ./...`
- Verified with `npm run build`
- Unable to verify because ...

Always distinguish between assumptions and verified facts. This applies to schema/registry claims too — e.g. don't assert an asset ID or interaction shape is valid without checking it against `schema/scenario.schema.json` / `schema/asset-registry.json`.

---

## Security / Secrets

- Never read `.env` or `server/.env` — treat as secret. Agents ask the user to set `OPENROUTER_API_KEY` (or legacy `OPENCODE_API_KEY`) locally.
- Validator rejects arbitrary component names / executable code fields — do not weaken this (see Contracts).
- CORS allow only `http://localhost:5173` (+ `127.0.0.1:5173`).

## Useful Paths

- Health: `GET /health → {"status":"ok"}`
- Assets: `GET /api/assets`
- Generate: `POST /api/scenario/generate` + Validate: `GET|POST /api/scenario/validate`
- Dist not committed; `client/dist/` built via `npm run build`.