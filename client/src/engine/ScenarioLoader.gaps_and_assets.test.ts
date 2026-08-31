/**
 * Gap + AssetResolver + AssetStreamer + world bounds negatives.
 * Split from ScenarioLoader.negative.test.ts to keep each file <250 LOC.
 *
 * Run: npx --yes tsx src/engine/ScenarioLoader.gaps_and_assets.test.ts
 */
import "./__tests__/polyfill.js";
import { validateScenarioSync } from "./ScenarioLoader.js";
import { hasAsset, getAsset } from "../assets/assetRegistry.js";
import { AssetStreamer } from "../assets/AssetStreamer.js";
import { resolveAsset, resolveSpritePath, clearCache } from "./AssetResolver.js";
import {
  WORLD_COLS_MAX,
  WORLD_COLS_MIN,
  WORLD_POS_MAX,
  WORLD_POS_MIN,
  WORLD_ROWS_MAX,
  WORLD_ROWS_MIN,
} from "../world/worldConstants.js";
import { getLayout } from "../world/layouts.js";

const assert = {
  equal(a: unknown, b: unknown, msg?: string): void {
    if (a !== b) throw new Error(msg ?? `assert equal failed: ${String(a)} !== ${String(b)}`);
  },
};

let passed = 0;
let failed = 0;
function ok(name: string, fn: () => void): void {
  try {
    fn();
    passed++;
    console.log(`✅ ${name}`);
  } catch (e) {
    failed++;
    console.error(`❌ ${name}: ${e instanceof Error ? e.message : String(e)}`);
  }
}
function expectNoThrow(fn: () => unknown): void {
  fn();
}
function expectThrow(fn: () => unknown, substr: string): void {
  let threw = false;
  let msg = "";
  try {
    fn();
  } catch (e) {
    threw = true;
    msg = e instanceof Error ? e.message : String(e);
  }
  if (!threw) throw new Error(`expected throw containing "${substr}" but did not throw`);
  if (!msg.includes(substr)) throw new Error(`throw message "${msg}" missing expected "${substr}"`);
}
function baseScenario(): Record<string, unknown> {
  return {
    id: "test-town",
    title: "Test Town",
    version: "1.0",
    world: { template: "town", spawn: { x: 4, y: 4 }, size: { cols: 15, rows: 12 } },
    characters: [{ id: "char1", name: "A", position: { x: 1, y: 1 }, appearance: { spriteId: "character_teacher" } }],
    buildings: [{ id: "b1", typeAssetId: "shop_small", position: { x: 2, y: 2 } }],
    objects: [{ id: "o1", assetId: "object_chest", position: { x: 3, y: 3 } }],
    missions: [{ id: "m1", title: "M", description: "D" }],
  };
}
function clone(obj: Record<string, unknown>): Record<string, unknown> {
  return JSON.parse(JSON.stringify(obj)) as Record<string, unknown>;
}
function sizeOf(s: Record<string, unknown>): Record<string, unknown> {
  return (s["world"] as Record<string, unknown>)["size"] as Record<string, unknown>;
}

// GAPs — client loader does NOT yet enforce these (server does); they document drift
ok("GAP: duplicate entity ID not caught by client", () => {
  const sc = clone(baseScenario());
  (sc["buildings"] as unknown[]).push({ id: "char1", typeAssetId: "shop_small", position: { x: 5, y: 5 } });
  expectNoThrow(() => validateScenarioSync(sc));
});
ok("FIXED: forbidden field 'code' now rejected", () => {
  const sc = clone(baseScenario()) as Record<string, unknown> & { code?: string };
  sc["code"] = "evil()";
  expectThrow(() => validateScenarioSync(sc), "forbidden field");
});
ok("FIXED: forbidden 'script' nested now rejected", () => {
  const sc = clone(baseScenario());
  const c = (sc["characters"] as Record<string, unknown>[])[0] as Record<string, unknown>;
  c["script"] = "alert(1)";
  expectThrow(() => validateScenarioSync(sc), "forbidden field");
});
ok("FIXED: building width overflow now rejected", () => {
  const sc = clone(baseScenario());
  const b = (sc["buildings"] as Record<string, unknown>[])[0] as Record<string, unknown>;
  b["position"] = { x: 14, y: 2 };
  b["width"] = 2;
  sizeOf(sc)["cols"] = 15;
  expectThrow(() => validateScenarioSync(sc), "exceeds world cols");
});
ok("GAP: mission trigger invalid entityId not caught", () => {
  const sc = clone(baseScenario());
  (sc["missions"] as Record<string, unknown>[])[0] = { id: "m1", title: "M", description: "D", trigger: { entityId: "ghost_entity" } };
  expectNoThrow(() => validateScenarioSync(sc));
});
ok("GAP: initialStats out of bounds not caught", () => {
  const sc = clone(baseScenario());
  sc["initialStats"] = { coins: 9999999, lives: 999 } as unknown as Record<string, unknown>;
  expectNoThrow(() => validateScenarioSync(sc));
});
ok("FIXED: dialogue without text now rejected", () => {
  const sc = clone(baseScenario());
  const c = (sc["characters"] as Record<string, unknown>[])[0] as Record<string, unknown>;
  c["interaction"] = { type: "dialogue" } as unknown as Record<string, unknown>;
  expectThrow(() => validateScenarioSync(sc), "text");
});
ok("FIXED: mcq single option now rejected", () => {
  const sc = clone(baseScenario());
  const c = (sc["characters"] as Record<string, unknown>[])[0] as Record<string, unknown>;
  c["interaction"] = { type: "mcq", question: "Q", options: [{ text: "only", correct: true }] } as unknown as Record<string, unknown>;
  expectThrow(() => validateScenarioSync(sc), "2-5");
});
ok("FIXED: shop empty items now rejected", () => {
  const sc = clone(baseScenario());
  const c = (sc["characters"] as Record<string, unknown>[])[0] as Record<string, unknown>;
  c["interaction"] = { type: "shop", items: [] } as unknown as Record<string, unknown>;
  expectThrow(() => validateScenarioSync(sc), "1-10");
});
ok("FIXED: shop negative price now rejected", () => {
  const sc = clone(baseScenario());
  const c = (sc["characters"] as Record<string, unknown>[])[0] as Record<string, unknown>;
  c["interaction"] = { type: "shop", items: [{ name: "Bad", price: -5 }] } as unknown as Record<string, unknown>;
  expectThrow(() => validateScenarioSync(sc), "price");
});

// AssetResolver
ok("hasAsset false for fake id", () => {
  assert.equal(hasAsset("not_an_asset_xyz"), false);
  assert.equal(hasAsset("fake_building_zzz"), false);
});
ok("hasAsset true for known ids", () => {
  assert.equal(hasAsset("shop_small"), true);
  assert.equal(hasAsset("character_teacher"), true);
  assert.equal(hasAsset("object_chest"), true);
});
ok("getAsset undefined for fake", () => {
  assert.equal(getAsset("zzz_fake"), undefined);
});
ok("resolveAsset fallback for fake", () => {
  clearCache();
  const r = resolveAsset("zzz_fake_2");
  assert.equal(r.entry, null);
  assert.equal(r.icon, "❓");
  assert.equal(r.solid, false);
  assert.equal(r.bundle, null);
  assert.equal(resolveSpritePath("zzz_fake_2"), null);
});
ok("resolveAsset cached same icon", () => {
  clearCache();
  const a = resolveAsset("shop_small");
  const b = resolveAsset("shop_small");
  assert.equal(a.icon, b.icon);
  assert.equal(a.solid, true);
});

// AssetStreamer edges
ok("getRequiredAssetIds empty", () => {
  const s = new AssetStreamer();
  const ids = s.getRequiredAssetIds({ buildings: [], objects: [], characters: [] } as unknown as import("../types/scenario.js").Scenario);
  assert.equal(ids.length, 0);
});
ok("getRequiredAssetIds dedup", () => {
  const s = new AssetStreamer();
  const sc = {
    buildings: [{ typeAssetId: "shop_small" }, { typeAssetId: "shop_small" }],
    objects: [{ assetId: "object_chest" }],
    characters: [{ appearance: { spriteId: "character_teacher" } }, { appearance: {} }, {}],
  } as unknown as import("../types/scenario.js").Scenario;
  const ids = s.getRequiredAssetIds(sc);
  assert.equal(ids.length, 3);
  assert.equal(ids.includes("shop_small"), true);
});
ok("checkCache missing only", () => {
  const s = new AssetStreamer();
  s.clearCache();
  const missing = s.checkCache(["shop_small", "object_chest"]);
  assert.equal(missing.length, 2);
  assert.equal(s.cachedCount(), 0);
});
ok("totalCount is 21", () => {
  assert.equal(new AssetStreamer().totalCount(), 21);
});

// world constants + layouts
ok("world constants match schema", () => {
  assert.equal(WORLD_COLS_MIN, 8);
  assert.equal(WORLD_COLS_MAX, 30);
  assert.equal(WORLD_ROWS_MIN, 8);
  assert.equal(WORLD_ROWS_MAX, 20);
  assert.equal(WORLD_POS_MIN, 0);
  assert.equal(WORLD_POS_MAX, 30);
});
ok("getLayout null for OOB size", () => {
  assert.equal(getLayout("town", { cols: 7, rows: 12 }), null);
  assert.equal(getLayout("town", { cols: 8, rows: 7 }), null);
  assert.equal(getLayout("town", { cols: 31, rows: 12 }), null);
  assert.equal(getLayout("town", { cols: 15, rows: 21 }), null);
});
ok("getLayout valid produces plots", () => {
  const l1 = getLayout("town", { cols: 8, rows: 8 });
  const l2 = getLayout("town", { cols: 15, rows: 12 });
  assert.equal(l1 !== null, true);
  assert.equal(l2 !== null, true);
  if (l1) assert.equal(l1.plots.length > 0, true);
  if (l2) assert.equal(l2.tilemap.length, 12);
});

console.log(`\n${passed} passed, ${failed} failed of ${passed + failed}`);
if (failed > 0) throw new Error(`${failed} test(s) failed`);
