# GLAM Scenario Contract

This directory locks the contract between three parties:

| Party | File it reads | What it enforces |
|-------|---------------|------------------|
| **LLM generator** | `scenario.schema.json` + `asset-registry.json` | Must emit valid JSON that conforms to the schema and references **only** registry asset IDs. No executable code. |
| **Go validator** (`server/scenario`) | `scenario.schema.json` + `asset-registry.json` | JSON Schema validation + semantic checks: asset-id existence, position bounds within `world.size`, mission trigger targets exist, at least one stat outcome sanity. |
| **Phaser runtime** (`client/src/engine`) | Same two files (bundled via `client/src/assets/registry.json`) | Assumes validated input; renders world, characters, buildings, objects and dispatches `interaction.type` to the activity handlers. |

`client/src/assets/registry.json` and `schema/asset-registry.json` are **identical copies**. The former is bundled for the browser; the latter is read by the Go server. Keep them in sync (CI can `diff` them).

**Sync check:** `make registry-check` or `npm run registry:check` (from `client/`) diffs the two files; CI workflow `.github/workflows/registry-sync.yml` fails on drift. `make registry-sync` copies canonical `schema/asset-registry.json` to `client/src/assets/registry.json`.

**World bounds canonical source:** `schema/scenario.schema.json` — `WorldColsMin/Max`, `WorldRowsMin/Max`, `WorldPosMin/Max` are centralized in `client/src/world/worldConstants.ts` and `server/world/constants.go` (both comment `canonical: schema/scenario.schema.json`). Do not repeat magic numbers 8,30,20,0-30.

**Forest layout mirrored:** `server/world/forest.go` and `client/src/world/layouts.ts` share identical geometry plus named constants `FOREST_*` / `Forest*`. Header comment `NOTE: Mirrored in …` marks the pair; keep density 13%, factors 0.27/0.25/0.33 and hash 37/71/19/100 in sync.

---

## Files

- **`scenario.schema.json`** — JSON Schema 2020-12 (`$schema: https://json-schema.org/draft/2020-12/schema`), `additionalProperties: false` everywhere, `propertyNames` forbid `code`/`script`/`component`/`bundle`. Go will additionally validate `x,y ∈ [0,30]` against `world.size` and that every `typeAssetId`/`assetId`/`appearance.spriteId` exists in the registry.
- **`asset-registry.json`** — Manifest of the predefined GLAM library. Each entry:

  ```json
  {
    "id": "shop_small",          // kebab_snake, unique
    "type": "building|character|object|tile|prop",
    "bundle": "town|forest|school|common",
    "sprite": "sprites/...png",  // optional, relative to client public root
    "icon": "🏪",                // optional emoji for fallback / editor
    "solid": true,               // whether Phaser treats it as collidable
    "metadata": { "description": "human readable …" }
  }
  ```

  The schema for this file is intentionally loose on `metadata` — it is documentation, not logic. `id` values are the **only** legal asset references inside a scenario.

- **`../scenarios/example.json`** — Canonical example: *Money Management Town* (town, 15×12, spawn 7,9). Uses only registry IDs and demonstrates all five interaction types (dialogue, mcq, shop, math, information). Validate with:

  ```bash
  python3 -m jsonschema -i scenarios/example.json schema/scenario.schema.json
  ```

---

## Scenario shape (summary)

```json
{
  "id": "kebab-case",
  "title": "Human title",
  "version": "1.0",
  "world": {
    "template": "town|forest|desert|school",
    "spawn": { "x": 0, "y": 0 },
    "size": { "cols": 15, "rows": 12 },
    "theme": "optional string",
    "regions": [ { "id": "region_a", "x": 0, "y": 0, "width": 5, "height": 5 } ]
  },
  "characters": [
    { "id":"…", "name":"…", "profession":"…", "appearance":{"spriteId":"character_teacher","color":"#fff"}, "position":{"x":0,"y":0}, "interaction":{ … } }
  ],
  "buildings": [
    { "id":"…", "typeAssetId":"shop_small", "position":{"x":0,"y":0}, "width":1, "height":1, "interaction":{ … } }
  ],
  "objects": [
    { "id":"…", "assetId":"object_chest", "position":{"x":0,"y":0}, "interaction":{ … } }
  ],
  "missions": [
    { "id":"…", "title":"…", "description":"…", "trigger":{"entityId":"…"}, "checkAtEnd":false, "done":false }
  ]
}
```

### Interactions

Every `interaction` has `type` and common optional fields `cooldown` (ms, int ≥0), `auto` (bool — proximity trigger without E), `onCorrect`/`onWrong` (`{ stat?, delta?, toast? }`). Type-specific payload:

| `type` | Required payload | Notes |
|--------|-----------------|-------|
| `dialogue` | `text` 1–1000, `speaker?` | Simple popup |
| `mcq` | `question`, `options` 2–5 × `{ text, correct:bool, explanation? }`, `allowRetry?` | Quiz; maps to demo `quiz` |
| `math` | `question`, `answer` (number\|string), `tolerance?` number ≥0, `hint?` | Numeric/string exact or tolerant match |
| `shop` | `currency?` default `coins`, `items` 1–10 × `{ id?, name, price:int≥0, icon?, description? }` | Purchase list |
| `information` | `content` 1–3000, `title?`, `image?` | Read-only panel |

No other `type` values are allowed. Unknown interaction types will be rejected by both JSON Schema and Go.

---

## Rules for LLM output

1. **Only references, never definitions.** Emit `typeAssetId`/`assetId`/`spriteId` that exist in `asset-registry.json`. Do not invent assets, sprite paths, or bundles.
2. **No code.** Properties named `code`, `script`, `component`, `bundle` are forbidden at **every** level (enforced via `propertyNames.not`). Do not emit functions, JS, or template strings. The Phaser runtime maps `type` to pre-built handlers.
3. **Strict IDs.** All `id` fields: `^[a-z0-9][a-z0-9_-]*$`, max 64. No spaces, no uppercase.
4. **Bounds.** `x,y` integers 0–30. Go additionally checks `x < cols`, `y < rows`. Stay inside `world.size`.
5. **Keep `additionalProperties: false` in mind.** Extra keys (e.g. `missionId` on entities — present in the HTML demo — is now `missions[].trigger.entityId`) will fail validation. Follow this schema exactly.
6. **Missions.** Use `trigger.entityId` to link a mission to the entity whose interaction completes it. `checkAtEnd: true` missions are evaluated after the others (e.g. savings threshold).
7. **Version.** Omit `version` → defaults to `1.0`.

---

## Go validator responsibilities (beyond JSON Schema)

JSON Schema cannot check cross-file references or map bounds exhaustively, so Go also validates:

- Every `typeAssetId`, `assetId`, `appearance.spriteId` ∈ registry `id` set.
- `sprite`/`icon` existence is not required, but IDs must match.
- `position` inside `world.size` (`0 ≤ x < cols`, `0 ≤ y < rows`).
- `missions[].trigger.entityId` (if present) references an existing character/building/object `id`.
- No duplicate `id` across characters+buildings+objects+missions (recommends namespacing, e.g. `ms_rao` vs `grocery_shop`).
- `shop.items[].price` affordability is runtime concern, not validation error.
- Forbidden property names double-checked at Go level for defence in depth.

---

## Phaser runtime expectations

- Loads `client/src/assets/registry.json` to resolve `typeAssetId`/`assetId` → sprite + solidity + icon.
- `world.template` selects base tileset (`town`/`forest`/`desert`/`school`); `world.theme` overrides if present.
- Interaction dispatch is a `switch (interaction.type)` — adding a new type requires engine + schema + Go changes together.
- `auto: true` interactions fire on proximity with `cooldown` debounce; others require `E` press.
- `onCorrect`/`onWrong` optionally update a stat and show a toast; they do not execute code.

---

## Adding a new asset

1. Add entry to `client/src/assets/registry.json`.
2. Copy to `schema/asset-registry.json` (or `cp` it).
3. Add sprite file under `client/public/sprites/...` (not covered here).
4. No schema change needed — registry is data.

Adding a new **interaction type** or **world template** requires coordinated changes to `scenario.schema.json`, Go validator, and Phaser handlers — treat as a breaking contract change.

---

## Validation

```bash
# Python (requires jsonschema)
python3 -m jsonschema -i scenarios/example.json schema/scenario.schema.json
python3 -c "import json, pathlib; data=json.loads(pathlib.Path('client/src/assets/registry.json').read_text()); print(f'registry entries: {len(data)}')"

# Node (requires ajv-cli)
npx ajv-cli validate -s schema/scenario.schema.json -d scenarios/example.json --spec=draft2020

# Go (once validator exists)
go run ./server/scenario --validate scenarios/example.json
```
