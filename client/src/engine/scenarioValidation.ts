import type { Position, Size } from "../types/scenario";
import { hasAsset } from "../assets/assetRegistry";
import {
  WORLD_COLS_MAX,
  WORLD_ROWS_MAX,
  WORLD_COLS_MIN,
  WORLD_POS_MAX,
  WORLD_POS_MIN,
  WORLD_ROWS_MIN,
} from "../world/worldConstants";
import { VALID_INTERACTION_TYPES } from "../types/interactionTypes";

const FORBIDDEN_FIELDS = new Set(["code", "script", "component", "bundle"]);

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
  assert(
    (r["x"] as number) >= WORLD_POS_MIN && (r["x"] as number) <= WORLD_POS_MAX,
    `${path}.x out of bounds [${WORLD_POS_MIN},${WORLD_POS_MAX}]`,
  );
  assert(
    (r["y"] as number) >= WORLD_POS_MIN && (r["y"] as number) <= WORLD_POS_MAX,
    `${path}.y out of bounds [${WORLD_POS_MIN},${WORLD_POS_MAX}]`,
  );
}

function validateSize(size: unknown, path: string): void {
  assert(isRecord(size), `${path} must be object`);
  const r = size as Record<string, unknown>;
  assert(typeof r["cols"] === "number" && Number.isInteger(r["cols"]), `${path}.cols must be integer`);
  assert(typeof r["rows"] === "number" && Number.isInteger(r["rows"]), `${path}.rows must be integer`);
  assert(
    (r["cols"] as number) >= WORLD_COLS_MIN && (r["cols"] as number) <= WORLD_COLS_MAX,
    `${path}.cols out of bounds [${WORLD_COLS_MIN},${WORLD_COLS_MAX}]`,
  );
  assert(
    (r["rows"] as number) >= WORLD_ROWS_MIN && (r["rows"] as number) <= WORLD_ROWS_MAX,
    `${path}.rows out of bounds [${WORLD_ROWS_MIN},${WORLD_ROWS_MAX}]`,
  );
}

function inBounds(pos: Position, size: Size): boolean {
  return pos.x >= 0 && pos.y >= 0 && pos.x < size.cols && pos.y < size.rows;
}

function checkForbiddenFields(value: unknown, path: string): void {
  if (isRecord(value)) {
    for (const k of Object.keys(value)) {
      if (FORBIDDEN_FIELDS.has(k)) {
        const label = path ? `${path}.${k}` : k;
        throw new Error(`forbidden field "${label}" not allowed`);
      }
      const child = (value as Record<string, unknown>)[k];
      if (isRecord(child)) checkForbiddenFields(child, path ? `${path}.${k}` : k);
      else if (Array.isArray(child)) {
        for (let i = 0; i < child.length; i++) {
          const elem = child[i];
          if (isRecord(elem)) checkForbiddenFields(elem, `${path ? `${path}.` : ""}${k}[${i}]`);
          else if (Array.isArray(elem)) checkForbiddenFields(elem, `${path ? `${path}.` : ""}${k}[${i}]`);
        }
      }
    }
  } else if (Array.isArray(value)) {
    for (let i = 0; i < value.length; i++) checkForbiddenFields(value[i], `${path}[${i}]`);
  }
}

function validateInteraction(inter: unknown, path: string): void {
  assert(isRecord(inter), `${path} must be object`);
  const r = inter as Record<string, unknown>;
  assert(typeof r["type"] === "string", `${path}.type is required`);
  const t = r["type"] as string;
  assert((VALID_INTERACTION_TYPES as Set<string>).has(t), `${path}.type "${t}" is not valid`);
  // Common optional checks (lightweight)
  if (r["cooldown"] !== undefined) {
    assert(typeof r["cooldown"] === "number" && Number.isInteger(r["cooldown"]) && (r["cooldown"] as number) >= 0, `${path}.cooldown must be integer >=0`);
  }
  if (r["auto"] !== undefined) assert(typeof r["auto"] === "boolean", `${path}.auto must be boolean`);
  // Per-type required fields — minimal strict checks (full schema is larger; remaining rules via server schema)
  if (t === "dialogue") {
    assert(typeof r["text"] === "string" && (r["text"] as string).length >= 1 && (r["text"] as string).length <= 1000, `${path}.text is required (1-1000)`);
  } else if (t === "mcq") {
    assert(typeof r["question"] === "string" && (r["question"] as string).length >= 1, `${path}.question is required`);
    assert(Array.isArray(r["options"]), `${path}.options must be array`);
    const opts = r["options"] as unknown[];
    assert(opts.length >= 2 && opts.length <= 5, `${path}.options must have 2-5 items`);
    for (let i = 0; i < opts.length; i++) {
      assert(isRecord(opts[i]), `${path}.options[${i}] must be object`);
      const o = opts[i] as Record<string, unknown>;
      assert(typeof o["text"] === "string" && (o["text"] as string).length >= 1, `${path}.options[${i}].text is required`);
      assert(typeof o["correct"] === "boolean", `${path}.options[${i}].correct is required (boolean)`);
    }
  } else if (t === "math") {
    assert(typeof r["question"] === "string" && (r["question"] as string).length >= 1, `${path}.question is required`);
    assert(r["answer"] !== undefined, `${path}.answer is required`);
    assert(typeof r["answer"] === "number" || typeof r["answer"] === "string", `${path}.answer must be number|string`);
    if (typeof r["answer"] === "string") assert((r["answer"] as string).length >= 1, `${path}.answer must be non-empty`);
    if (r["tolerance"] !== undefined) assert(typeof r["tolerance"] === "number" && (r["tolerance"] as number) >= 0, `${path}.tolerance must be number >=0`);
  } else if (t === "shop") {
    assert(Array.isArray(r["items"]), `${path}.items must be array`);
    const items = r["items"] as unknown[];
    assert(items.length >= 1 && items.length <= 10, `${path}.items must have 1-10 items`);
    for (let i = 0; i < items.length; i++) {
      assert(isRecord(items[i]), `${path}.items[${i}] must be object`);
      const it = items[i] as Record<string, unknown>;
      assert(typeof it["name"] === "string" && (it["name"] as string).length >= 1, `${path}.items[${i}].name is required`);
      assert(typeof it["price"] === "number" && Number.isInteger(it["price"]) && (it["price"] as number) >= 0, `${path}.items[${i}].price must be integer >=0`);
    }
  } else if (t === "information") {
    assert(typeof r["content"] === "string" && (r["content"] as string).length >= 1 && (r["content"] as string).length <= 3000, `${path}.content is required (1-3000)`);
  }
}

function validateBuildingFootprint(b: Record<string, unknown>, size: Size, idx: number): void {
  const pos = b["position"] as unknown as Position;
  // width / height optional but if present must be int >=1 <= max and footprint inside world
  if (b["width"] !== undefined) {
    assert(typeof b["width"] === "number" && Number.isInteger(b["width"]), `buildings[${idx}].width must be integer`);
    const w = b["width"] as number;
    assert(w >= 1 && w <= WORLD_COLS_MAX, `buildings[${idx}].width out of range [1,${WORLD_COLS_MAX}]`);
    assert(pos.x + w <= size.cols, `buildings[${idx}].position x+width ${pos.x + w} exceeds world cols ${size.cols}`);
  }
  if (b["height"] !== undefined) {
    assert(typeof b["height"] === "number" && Number.isInteger(b["height"]), `buildings[${idx}].height must be integer`);
    const h = b["height"] as number;
    assert(h >= 1 && h <= WORLD_ROWS_MAX, `buildings[${idx}].height out of range [1,${WORLD_ROWS_MAX}]`);
    assert(pos.y + h <= size.rows, `buildings[${idx}].position y+height ${pos.y + h} exceeds world rows ${size.rows}`);
  }
}

export function validateScenarioObject(obj: unknown): asserts obj is import("../types/scenario").Scenario {
  assert(isRecord(obj), "Scenario must be an object");
  checkForbiddenFields(obj, "");
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
    assert(inBounds(c["position"] as unknown as Position, size), `characters[${i}].position out of bounds`);
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
    if (c["interaction"] !== undefined) validateInteraction(c["interaction"], `characters[${i}].interaction`);
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
    validateBuildingFootprint(b, size, i);
    if (b["plot"] !== undefined) {
      assert(typeof b["plot"] === "string" && /^[a-z0-9][a-z0-9_-]*$/.test(b["plot"] as string), `buildings[${i}].plot must match ^[a-z0-9][a-z0-9_-]*$`);
    }
    if (b["interaction"] !== undefined) validateInteraction(b["interaction"], `buildings[${i}].interaction`);
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
    if (o["interaction"] !== undefined) validateInteraction(o["interaction"], `objects[${i}].interaction`);
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
