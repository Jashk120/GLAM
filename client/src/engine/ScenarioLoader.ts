import type { Scenario, Position, Size } from "../types/scenario";
import { hasAsset } from "../assets/assetRegistry";
import { assetStreamer } from "../assets/AssetStreamer";

function isRecord(v: unknown): v is Record<string, unknown> {
  return typeof v === "object" && v !== null && !Array.isArray(v);
}

function assert(cond: boolean, msg: string): void {
  if (!cond) throw new Error(msg);
}

function validatePosition(pos: unknown, path: string): void {
  assert(isRecord(pos), `${path} must be object`);
  const r = pos as Record<string, unknown>;
  assert(typeof r["x"] === "number" && Number.isInteger(r["x"]), `${path}.x must be integer`);
  assert(typeof r["y"] === "number" && Number.isInteger(r["y"]), `${path}.y must be integer`);
  assert((r["x"] as number) >= 0 && (r["x"] as number) <= 30, `${path}.x out of bounds [0,30]`);
  assert((r["y"] as number) >= 0 && (r["y"] as number) <= 30, `${path}.y out of bounds [0,30]`);
}

function validateSize(size: unknown, path: string): void {
  assert(isRecord(size), `${path} must be object`);
  const r = size as Record<string, unknown>;
  assert(typeof r["cols"] === "number" && Number.isInteger(r["cols"]), `${path}.cols must be integer`);
  assert(typeof r["rows"] === "number" && Number.isInteger(r["rows"]), `${path}.rows must be integer`);
  assert((r["cols"] as number) >= 8 && (r["cols"] as number) <= 30, `${path}.cols out of bounds [8,30]`);
  assert((r["rows"] as number) >= 8 && (r["rows"] as number) <= 20, `${path}.rows out of bounds [8,20]`);
}

function inBounds(pos: Position, size: Size): boolean {
  return pos.x >= 0 && pos.y >= 0 && pos.x < size.cols && pos.y < size.rows;
}

function validateScenarioObject(obj: unknown): asserts obj is Scenario {
  assert(isRecord(obj), "Scenario must be an object");
  const r = obj as Record<string, unknown>;

  assert(typeof r["id"] === "string" && (r["id"] as string).length > 0, "Scenario.id is required (non-empty string)");
  assert(/^[a-z0-9][a-z0-9_-]*$/.test(r["id"] as string), "Scenario.id must match ^[a-z0-9][a-z0-9_-]*$");
  assert(typeof r["title"] === "string" && (r["title"] as string).length > 0, "Scenario.title is required");

  assert(isRecord(r["world"]), "Scenario.world is required");
  const world = r["world"] as Record<string, unknown>;
  assert(
    typeof world["template"] === "string" && ["town", "forest", "desert", "school"].includes(world["template"] as string),
    "world.template must be one of town|forest|desert|school",
  );
  validatePosition(world["spawn"], "world.spawn");
  validateSize(world["size"], "world.size");

  const size = world["size"] as unknown as Size;
  const spawn = world["spawn"] as unknown as Position;
  assert(inBounds(spawn, size), `world.spawn (${spawn.x},${spawn.y}) out of world size ${size.cols}x${size.rows}`);

  assert(Array.isArray(r["characters"]), "Scenario.characters must be array");
  assert(Array.isArray(r["buildings"]), "Scenario.buildings must be array");
  assert(Array.isArray(r["objects"]), "Scenario.objects must be array");
  assert(Array.isArray(r["missions"]), "Scenario.missions must be array");

  const chars = r["characters"] as unknown[];
  for (let i = 0; i < chars.length; i++) {
    assert(isRecord(chars[i]), `characters[${i}] must be object`);
    const c = chars[i] as Record<string, unknown>;
    assert(typeof c["id"] === "string", `characters[${i}].id required`);
    assert(typeof c["name"] === "string", `characters[${i}].name required`);
    validatePosition(c["position"], `characters[${i}].position`);
    assert(
      inBounds(c["position"] as unknown as Position, size),
      `characters[${i}].position out of bounds`,
    );
    if (c["plot"] !== undefined) {
      assert(typeof c["plot"] === "string" && /^[a-z0-9][a-z0-9_-]*$/.test(c["plot"] as string), `characters[${i}].plot must match ^[a-z0-9][a-z0-9_-]*$`);
    }
    if (c["appearance"] && isRecord(c["appearance"])) {
      const ap = c["appearance"] as Record<string, unknown>;
      if (ap["spriteId"]) {
        assert(typeof ap["spriteId"] === "string", `characters[${i}].appearance.spriteId must be string`);
        assert(hasAsset(ap["spriteId"] as string), `characters[${i}].appearance.spriteId "${ap["spriteId"] as string}" not in registry`);
      }
    }
  }

  const buildings = r["buildings"] as unknown[];
  for (let i = 0; i < buildings.length; i++) {
    assert(isRecord(buildings[i]), `buildings[${i}] must be object`);
    const b = buildings[i] as Record<string, unknown>;
    assert(typeof b["id"] === "string", `buildings[${i}].id required`);
    assert(typeof b["typeAssetId"] === "string", `buildings[${i}].typeAssetId required`);
    assert(hasAsset(b["typeAssetId"] as string), `buildings[${i}].typeAssetId "${b["typeAssetId"] as string}" not in registry`);
    validatePosition(b["position"], `buildings[${i}].position`);
    assert(inBounds(b["position"] as unknown as Position, size), `buildings[${i}].position out of bounds`);
    if (b["plot"] !== undefined) {
      assert(typeof b["plot"] === "string" && /^[a-z0-9][a-z0-9_-]*$/.test(b["plot"] as string), `buildings[${i}].plot must match ^[a-z0-9][a-z0-9_-]*$`);
    }
  }

  const objects = r["objects"] as unknown[];
  for (let i = 0; i < objects.length; i++) {
    assert(isRecord(objects[i]), `objects[${i}] must be object`);
    const o = objects[i] as Record<string, unknown>;
    assert(typeof o["id"] === "string", `objects[${i}].id required`);
    assert(typeof o["assetId"] === "string", `objects[${i}].assetId required`);
    assert(hasAsset(o["assetId"] as string), `objects[${i}].assetId "${o["assetId"] as string}" not in registry`);
    validatePosition(o["position"], `objects[${i}].position`);
    assert(inBounds(o["position"] as unknown as Position, size), `objects[${i}].position out of bounds`);
    if (o["plot"] !== undefined) {
      assert(typeof o["plot"] === "string" && /^[a-z0-9][a-z0-9_-]*$/.test(o["plot"] as string), `objects[${i}].plot must match ^[a-z0-9][a-z0-9_-]*$`);
    }
  }

  const missions = r["missions"] as unknown[];
  for (let i = 0; i < missions.length; i++) {
    assert(isRecord(missions[i]), `missions[${i}] must be object`);
    const m = missions[i] as Record<string, unknown>;
    assert(typeof m["id"] === "string", `missions[${i}].id required`);
    assert(typeof m["title"] === "string", `missions[${i}].title required`);
    assert(typeof m["description"] === "string", `missions[${i}].description required`);
  }
}

export async function loadScenario(source: string | object): Promise<Scenario> {
  let raw: unknown;
  if (typeof source === "string") {
    const res = await fetch(source);
    if (!res.ok) throw new Error(`Failed to fetch scenario: ${source} — ${res.status} ${res.statusText}`);
    raw = (await res.json()) as unknown;
  } else {
    raw = source;
  }
  validateScenarioObject(raw);
  const scenario = raw as Scenario;
  await assetStreamer.preloadScenarioAssets(scenario);
  return scenario;
}

export function validateScenarioSync(obj: unknown): Scenario {
  validateScenarioObject(obj);
  return obj as Scenario;
}
