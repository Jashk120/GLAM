# GLAM Roadmap

> One engine — any lesson. Data-driven scenario contract (schema + registry) between Go, LLM, and Phaser.

Date: 2026-08-31  
Status: swarm audit item 1 (duplicate position + warnings) **done**; items 2–4 deferred pending user approval per `AGENTS.md` minimal-change policy.

---

## Decisions

**shop.currency: free-form (no enum).** Stays `string 1-32` per `schema/scenario.schema.json` `interaction_shop.currency`. Intentionally free to support bartering (`"barter"`, `"gems"`, `"shells"`, `"coins"`) and alternative economies. Tightening to an enum would break educational flexibility and requires schema + validator + prompt + example changes — blocked pending user approval. No code change in this PR; decision documented in `server/scenario/types.go:InteractionShop.Currency` comment and `server/llm/prompt.go` prose.

**information.image: text-only display (no `<img>` fetch).** Rendered as escaped text in `client/src/activities/InformationActivity.ts:13`:
```ts
${interaction.image ? `<div>🖼️ ${escapeHtml(interaction.image)}</div>` : ""}
```
No network fetch, no `src=` assignment, so external URL risk is display-only and XSS-mitigated via `escapeHtml`. No allowlist/tightening applied now; would require `<img>` rendering + URL validation (allowlist, CSP) — blocked pending user approval. Documented here only.

---

## 2. 250 LOC Splits

`AGENTS.md` imposes 250 LOC/file ceiling. Three existing files exceed it; splitting needs user approval (new files, moved symbols, import rewiring).

| File | Lines | Proposed split | Rationale | Effort |
|------|-------|----------------|-----------|--------|
| `server/api/handler.go` (634) | 634 | `handler_generate.go` (~180: `HandleGenerate`, `hasPlotError`, `hasAdditionalPropertiesError`, `truncateErr`, warnings logic), `handler_validate.go` (~90: `HandleValidate`, `readBody`), `handler_assets.go` (~50: `HandleAssets`), `handler_scenarios.go` (~180: `HandleListScenarios`, `HandleGetScenario`, `scenariosDir`, `sanitizeFilename`, `saveGenerated`) + `handler_common.go` (`writeJSON`, `writeError`, `isBodyTooLarge`, constants) | Single file mixes 4 endpoint families + persistence. Split restores <250, isolates generate auto-repair warnings from scenario file I/O. Keep `Handler` struct in `common`. | ~2h, verify `go vet` + handler tests (36+ cases) |
| `server/scenario/validator.go` (606) | 606 | `validator.go` (core `ValidateScenario` orchestration), `validator_forbidden.go` (`checkForbiddenFields`), `validator_positions.go` (`checkDuplicatePositions`, `PositionInPlot` helpers, footprint/bounds), `validator_layout.go` (layout/plot/solid checks), `validator_interaction.go` (`collectInteractions`, `ValidActivityTypes` wiring) | Validation pipeline is 6 stages in one file. Isolating position/duplicate logic and layout checks reduces blast radius for swarm item 1 and keeps each <150 LOC. | ~1.5h |
| `server/llm/client.go` (489) | 489 | `client_generate.go` (`Generate`, `extractText`, fence stripping), `client_tools.go` (`GenerateWithTools`, `extractToolCalls`, tool payload building), `client_env.go` (`envOr`, `envTimeout`, `envMaxTokens`, constants) | Generate vs tool-calling paths share env/config but diverge in message handling. Separation avoids mixing truncation logic with tool-call JSON. | ~1h |

All splits require `go vet ./...` + `go test ./...` clean and `gofmt -w`. No new dependencies. Needs approval because it touches the validation contract's file layout.

---

## 3. Dependencies

Per `AGENTS.md` — keep `server/go.mod` minimal (currently 2 deps: `santhosh-tekuri/jsonschema/v5`, `joho/godotenv`). Any addition requires user sign-off.

| Dep | Where | Impact | Why approval needed |
|-----|-------|--------|---------------------|
| `golang.org/x/time/rate` | server rate-limiting for `/api/scenario/generate`, `/api/chat` | Std-lib lacks token-bucket; needed for abuse protection but adds transitive dep and hot-path locking | Minimal-deps policy; `x/time` is semi-stdlib but still external |
| `vitest` | `client` | Enables `*.test.ts` without asking user per AGENTS; no runner currently wired (`client/package.json` has no test script) | Adding test runner is a dependency decision — raise first |
| `teacher-interface` cleanup: `lenis` (smooth scroll), `shadcn` (CLI), `react-icons`, `lucide-react`, `tw-animate-css` | `teacher-interface/package.json` | `lenis` unused after design iteration; `shadcn` is CLI tool, not runtime dep; `react-icons` vs `lucide-react` duplicate icon sets; `tw-animate-css` heavy for single animation | Bloat: should audit `npm ls` and remove unused, but removal changes `node_modules` shape — needs explicit user OK |
| `go vet`/`gofmt` remain only server linters | — | No addition | — |

No dep added in this PR. Next PR should present `go get golang.org/x/time/rate` diff + rate-limit middleware + test for 429.

---

## 4. New Packages

Discovered packages lack `README.md` per `AGENTS.md` (every top-level dir / substantial Go package needs one). Endpoints also undocumented in `server/README.md`.

| Package | Role | Files | Missing README? | Verdict |
|---------|------|-------|-----------------|---------|
| `server/pipeline` | 4-stage deterministic assembly: `stage1_structure` (world+entities), `stage2_flavor` (names/professions), `stage3_interactions` (5 types), `stage4_assemble` (final marshalling) + `types.go:Deps` | 5 files (~570 LOC) | yes | **Keep + document**: provides explainable LLM pipeline; needs `server/pipeline/README.md` describing stage inputs/outputs and schema coupling |
| `server/agent` | Tool-calling loop: `loop.go:RunTurn`, `executor.go`, `tools.go` (MCP-style tool defs → JSON Schema), `prompt.go:SystemPrompt` | 4 files (~156 LOC) | yes | **Keep + document**: powers `/api/chat` multi-turn; needs `server/agent/README.md` + loop diagram |
| `server/session` | In-memory `sync.Mutex` session store: `Store{Get,Set,Append,Exists}` keyed by hex session ID | 1 file (46 LOC) | yes | **Keep + document** (or evaluate persistent store later); needs `server/session/README.md`; currently no TTL/cleanup |
| `server/llm/message.go` | Shared `Message{Role,Content,ToolCallID,ToolCalls}` + helpers `NewUserMessage` etc., used by agent + session | 1 file (32 LOC) | covered by `server/README.md`? but still deserves section | **Keep**; document in `server/llm/README.md` or top-level server README |
| `server/world` (extended) | `forest.go`, `layout.go`, `constants.go` — already documented in root `AGENTS.md` but no package README | 3 files | partial | Add `server/world/README.md` noting mirror with `client/src/world/layouts.ts` |

**Undocumented endpoints (need `server/README.md` update if kept):**
- `POST /api/chat {session_id?, message}` → `{session_id, reply, scenario?}` via `ChatHandler` + `session.Store` + `agent.RunTurn`
- `GET /api/scenarios` + `GET /api/scenarios/{id}` / `/api/scenario/{id}` — file-backed listing; scans `scenarios/` dir, skips bad JSON/txt

**Action:** do not revert; add READMEs and document endpoints in next PR. If any package is experimental, gate behind env flag rather than deleting — deletion would break `main.go` wiring.

---

## Next Steps (1 at a time)

1. **✅ Done (this PR): duplicate position + warnings** — `validator.go:checkDuplicatePositions` (origin-only, footprint comment) + `handler.go` warnings slice (`auto-fixed plot refs`, `stripped additionalProperties`) + tests (3 duplicate-position negatives, 3 warnings scenarios) + roadmap. Verified `go vet ./...`, `go test ./... -v`, `go build -o /tmp/glam-server .`.

2. **Next: 250 LOC splits** — propose PR with file boundaries above; owner approves file names + symbol moves; keep each <250, `gofmt -w`, re-run `make vet`.

3. **Then: dependencies** — separate PRs: (a) `golang.org/x/time/rate` middleware + 429 test, (b) `client` `vitest` wiring, (c) `teacher-interface` dep prune. Each needs minimal-dep justification in PR description.

4. **Finally: new packages READMEs + endpoint docs** — add `server/pipeline/README.md`, `server/agent/README.md`, `server/session/README.md`, update `server/README.md` endpoints table (`/api/chat`, `/api/scenarios`) and top-level `AGENTS.md` repo structure diagram.

Order preserves contract stability: validation behavior (1) before structural churn (2) before external deps (3) before docs (4).
