# GLAM Client — Scenario Engine

One engine — any lesson. Scenarios are just JSON. Teacher UI → Go → Phaser flow + asset streaming cache.

## Quick Start

```bash
npm install
npm run dev      # http://localhost:5173
```

The Go server must be running on **http://localhost:8080** for generation.

```bash
# from repo root
go run ./server/cmd/server   # listens on :8080, requires OPENCODE_API_KEY
# or
./server/bin/server
```

Vite proxies `/api` → `http://localhost:8080` (see `vite.config.ts`).

## Teacher Flow (Phase 7 MVP)

1. Open `http://localhost:5173`.
2. Top card: enter a lesson prompt (default placeholder: *Create a small town where students learn basic money management.*).
3. Click **✨ Generate Scenario** (or Ctrl/Cmd+Enter). Status shows *Generating scenario...*.
4. Client POSTs `http://localhost:8080/api/scenario/generate` via Vite proxy `/api/scenario/generate` with `{ prompt }`.
5. Go server forwards to Muse Spark with schema + registry, validates, returns `{ scenario }`.
6. Client validates via `ScenarioLoader` light checks, calls `window.Glam.loadScenario(scenario)` to render instantly in Phaser without reload.
7. Topbar selector adds **Generated: <title>** option. Toast shows *Generated!*. Scenario saved to `localStorage` as `glam_lastGenerated` and restored on next load.
8. Errors: validator `details` array rendered nicely in `#genError`; existing scenario stays loaded (game not blanked). Network failures show *is the Go server running on :8080?*.
9. Keep **Restart** button to replay current scenario. Selector keeps **Example** (`/scenarios/example.json`) and any generated entries.

Logs: `Load Example` on boot, `Play Generated` when switching to generated, `Teacher -> Go -> Phaser flow` on generate.

API: `window.Glam.loadScenario(src)` / `getScenario()` / `getGame()` still available.

## Asset Streaming (Phase 8)

`src/assets/AssetStreamer.ts` — cache via `Map` + `localStorage` JSON (`glam_asset_cache_v1`), designed for IndexedDB fallback later.

Methods:
- `getRequiredAssetIds(scenario)` – collects `building.typeAssetId` + `objects.assetId` + `characters.appearance.spriteId`
- `checkCache(ids)` – returns missing ids
- `fetchMissing(ids)` – simulates download (50 ms, emoji-based assets) then marks cached; ready for future bundle URLs
- `preloadScenarioAssets(scenario)` – called before `Game.create` via `ScenarioLoader.loadScenario` after parsing

UI: `#assetStatus` bar shows `Assets cached: 21/21` or `Downloading 2 assets...` live. Cache persists in localStorage.

Registry: `src/assets/registry.json` (21 entries, town/school/forest/common bundles).

## Build & Check

```bash
npm run build        # tsc + vite build
npx tsc --noEmit     # type clean, no as any
npm run preview      # serve dist
```

## Project Layout

- `src/main.ts` – boots Phaser `GameScene`, wires `TeacherUI` + `AssetStreamer`, exposes `window.Glam`
- `src/teacher/TeacherUI.ts` – textarea + generate logic + asset UI, separated from Game
- `src/engine/Game.ts` – Phaser scene (stable API)
- `src/engine/ScenarioLoader.ts` – fetch/validate + `AssetStreamer.preload` integration
- `src/assets/AssetStreamer.ts` + `assetRegistry.ts` + `registry.json`
- `src/arena/` – declarative Arena v1 flow validation, presentation model, and DOM runtime
- `index.html` – teacher panel + topbar + gameWrap + missionPanel
- `vite.config.ts` – `/api` proxy to 8080

No backend DB; Go server is read-only from client perspective; no hardcoded keys.

## Arena v1

Choose **🍎 Adding Apples — Arena** in the player selector to run the reference arena. An arena scenario carries an optional `arena` field. The tile-map UI is hidden while `ArenaRuntime` renders a teaching stage above a persistent learning console that handles dialogue, replayable visuals, multiple-choice answers, feedback, and completion.

The runtime accepts a deliberately small, safe component vocabulary: `dialogue`, `teaching` (counting objects plus `add` motion), `multipleChoice`, and `complete`. The schema and client validator reject arbitrary executable fields and invalid transitions. The full authoring contract is in [`../schema/README.md`](../schema/README.md).
