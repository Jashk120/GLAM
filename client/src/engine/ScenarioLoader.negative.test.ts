/**
 * Negative / edge tests for ScenarioLoader.
 * Data-driven — each case clones minimal scenario and mutates one field.
 * No Phaser canvas; pure validation.
 *
 * Run: npx --yes tsx src/engine/ScenarioLoader.negative.test.ts
 * Typecheck: npx tsc --noEmit
 * Strict TS, no `as any`.
 */
import "./__tests__/polyfill.js";
import { validateScenarioSync } from "./ScenarioLoader.js";

// ------------------------------------------------------------------ harness
let passed = 0;
let failed = 0;
function ok(name: string, fn: () => void): void {
  try {
    fn();
    passed++;
    console.log(`✅ ${name}`);
  } catch (e) {
    failed++;
    const msg = e instanceof Error ? e.message : String(e);
    console.error(`❌ ${name}: ${msg}`);
  }
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
function expectNoThrow(fn: () => unknown): void {
  fn();
}

// ------------------------------------------------------------------ base
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
function world(s: Record<string, unknown>): Record<string, unknown> {
  return s["world"] as Record<string, unknown>;
}
function sizeOf(s: Record<string, unknown>): Record<string, unknown> {
  return world(s)["size"] as Record<string, unknown>;
}
function spawnOf(s: Record<string, unknown>): Record<string, unknown> {
  return world(s)["spawn"] as Record<string, unknown>;
}

// ------------------------------------------------------------------ ScenarioLoader negatives (15+)
ok("missing id throws", () => {
  const sc = clone(baseScenario());
  delete sc["id"];
  expectThrow(() => validateScenarioSync(sc), "Scenario.id");
});
ok("missing title throws", () => {
  const sc = clone(baseScenario());
  delete sc["title"];
  expectThrow(() => validateScenarioSync(sc), "Scenario.title");
});
ok("missing world throws", () => {
  const sc = clone(baseScenario());
  delete sc["world"];
  expectThrow(() => validateScenarioSync(sc), "Scenario.world");
});
ok("empty id throws", () => {
  const sc = clone(baseScenario());
  sc["id"] = "";
  expectThrow(() => validateScenarioSync(sc), "Scenario.id");
});
ok("uppercase id throws", () => {
  const sc = clone(baseScenario());
  sc["id"] = "Bad_ID";
  expectThrow(() => validateScenarioSync(sc), "Scenario.id");
});
ok("slash in id throws", () => {
  const sc = clone(baseScenario());
  sc["id"] = "bad/id";
  expectThrow(() => validateScenarioSync(sc), "Scenario.id");
});
ok("invalid world.template throws", () => {
  const sc = clone(baseScenario());
  world(sc)["template"] = "volcano";
  expectThrow(() => validateScenarioSync(sc), "world.template");
});
ok("spawn out of 0-30 throws (x=31)", () => {
  const sc = clone(baseScenario());
  spawnOf(sc)["x"] = 31;
  expectThrow(() => validateScenarioSync(sc), "world.spawn");
});
ok("spawn negative throws", () => {
  const sc = clone(baseScenario());
  spawnOf(sc)["y"] = -1;
  expectThrow(() => validateScenarioSync(sc), "world.spawn");
});
ok("size cols too small (7) throws", () => {
  const sc = clone(baseScenario());
  sizeOf(sc)["cols"] = 7;
  expectThrow(() => validateScenarioSync(sc), "world.size.cols");
});
ok("size cols too large (31) throws", () => {
  const sc = clone(baseScenario());
  sizeOf(sc)["cols"] = 31;
  expectThrow(() => validateScenarioSync(sc), "world.size.cols");
});
ok("size rows too small (7) throws", () => {
  const sc = clone(baseScenario());
  sizeOf(sc)["rows"] = 7;
  expectThrow(() => validateScenarioSync(sc), "world.size.rows");
});
ok("size rows too large (21) throws", () => {
  const sc = clone(baseScenario());
  sizeOf(sc)["rows"] = 21;
  expectThrow(() => validateScenarioSync(sc), "world.size.rows");
});
ok("spawn >= cols throws (spawn 10 in 8 cols)", () => {
  const sc = clone(baseScenario());
  sizeOf(sc)["cols"] = 8;
  sizeOf(sc)["rows"] = 8;
  spawnOf(sc)["x"] = 10;
  spawnOf(sc)["y"] = 0;
  expectThrow(() => validateScenarioSync(sc), "world.spawn");
});
ok("building typeAssetId fake throws", () => {
  const sc = clone(baseScenario());
  const b = (sc["buildings"] as Record<string, unknown>[])[0] as Record<string, unknown>;
  if (b) b["typeAssetId"] = "fake_building_zzz";
  expectThrow(() => validateScenarioSync(sc), "not in registry");
});
ok("character appearance.spriteId fake throws", () => {
  const sc = clone(baseScenario());
  const c = (sc["characters"] as Record<string, unknown>[])[0] as Record<string, unknown>;
  const ap = c["appearance"] as Record<string, unknown>;
  ap["spriteId"] = "fake_sprite_zzz";
  expectThrow(() => validateScenarioSync(sc), "not in registry");
});
ok("object assetId fake throws", () => {
  const sc = clone(baseScenario());
  const o = (sc["objects"] as Record<string, unknown>[])[0] as Record<string, unknown>;
  if (o) o["assetId"] = "nope_asset";
  expectThrow(() => validateScenarioSync(sc), "not in registry");
});
ok("character position out of world bounds throws", () => {
  const sc = clone(baseScenario());
  const c = (sc["characters"] as Record<string, unknown>[])[0] as Record<string, unknown>;
  (c["position"] as Record<string, unknown>)["x"] = 20;
  expectThrow(() => validateScenarioSync(sc), "out of bounds");
});
ok("building position out of bounds throws", () => {
  const sc = clone(baseScenario());
  const b = (sc["buildings"] as Record<string, unknown>[])[0] as Record<string, unknown>;
  (b["position"] as Record<string, unknown>)["x"] = 99;
  expectThrow(() => validateScenarioSync(sc), "out of bounds");
});

// edge: 8x8 minimal passes, 30x20 max passes (positive edges)
ok("world size 8x8 minimal passes", () => {
  const sc = clone(baseScenario());
  sizeOf(sc)["cols"] = 8;
  sizeOf(sc)["rows"] = 8;
  spawnOf(sc)["x"] = 0;
  spawnOf(sc)["y"] = 0;
  (sc["characters"] as Record<string, unknown>[])[0] = { id: "c1", name: "N", position: { x: 0, y: 0 } };
  (sc["buildings"] as Record<string, unknown>[])[0] = { id: "b1", typeAssetId: "shop_small", position: { x: 1, y: 1 } };
  (sc["objects"] as Record<string, unknown>[])[0] = { id: "o1", assetId: "object_chest", position: { x: 2, y: 2 } };
  expectNoThrow(() => validateScenarioSync(sc));
});
ok("world size 30x20 max passes", () => {
  const sc = clone(baseScenario());
  sizeOf(sc)["cols"] = 30;
  sizeOf(sc)["rows"] = 20;
  spawnOf(sc)["x"] = 29;
  spawnOf(sc)["y"] = 19;
  expectNoThrow(() => validateScenarioSync(sc));
});

// ------------------------------------------------------------------ summary
console.log(`\n${passed} passed, ${failed} failed of ${passed + failed}`);
if (failed > 0) throw new Error(`${failed} test(s) failed`);
