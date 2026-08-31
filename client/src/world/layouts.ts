// NOTE: Mirrored in server/world/forest.go — keep in sync. Registry CI will fail if diff changes. Canonical layout: server/world/forest.go & client/src/world/layouts.ts
import type { TemplateType } from "../types/scenario";
import {
  FOREST_CLEARING_H_FACTOR,
  FOREST_CLEARING_MIN_SIZE,
  FOREST_CLEARING_W_FACTOR_A,
  FOREST_CLEARING_W_FACTOR_B,
  FOREST_HASH_A,
  FOREST_HASH_B,
  FOREST_HASH_MOD,
  FOREST_HASH_RANGE,
  FOREST_TREE_DENSITY_THRESHOLD,
  WORLD_COLS_MAX,
  WORLD_COLS_MIN,
  WORLD_ROWS_MAX,
  WORLD_ROWS_MIN,
} from "./worldConstants";
export type TileKind = "grass" | "path" | "tree" | "water";
export interface Plot { id: string; name: string; x: number; y: number; width: number; height: number; type: "plot" | "clearing"; }
export interface WorldLayout { tilemap: TileKind[][]; plots: Plot[]; spawn: { x: number; y: number }; }
export function isSolidTile(kind: TileKind): boolean { return kind === "tree" || kind === "water"; }
export function isWalkable(kind: TileKind): boolean { return !isSolidTile(kind); }
export const isSolid = isSolidTile;
export function isInPlot(x: number, y: number, plots: Plot[]): boolean {
  for (const p of plots) if (x >= p.x && x < p.x + p.width && y >= p.y && y < p.y + p.height) return true;
  return false;
}
function createGrassMap(cols: number, rows: number): TileKind[][] {
  const m: TileKind[][] = [];
  for (let y = 0; y < rows; y++) { const r: TileKind[] = []; for (let x = 0; x < cols; x++) r.push("grass"); m.push(r); }
  return m;
}
function cloneMap(m: TileKind[][]): TileKind[][] { return m.map((r) => [...r]); }
function ensureWalkableSpawn(tilemap: TileKind[][], spawn: { x: number; y: number }): { x: number; y: number } {
  const rows = tilemap.length; const cols = rows > 0 ? (tilemap[0]?.length ?? 0) : 0;
  if (rows === 0 || cols === 0) return spawn;
  const k = tilemap[spawn.y]?.[spawn.x]; if (k && isWalkable(k)) return spawn;
  const max = Math.max(cols, rows);
  for (let d = 1; d <= max; d++) for (let dy = -d; dy <= d; dy++) for (let dx = -d; dx <= d; dx++) {
    if (Math.abs(dx) !== d && Math.abs(dy) !== d) continue;
    const nx = spawn.x + dx, ny = spawn.y + dy;
    if (nx < 0 || ny < 0 || nx >= cols || ny >= rows) continue;
    const kk = tilemap[ny]?.[nx]; if (kk && isWalkable(kk)) return { x: nx, y: ny };
  }
  return spawn;
}
export function getTownLayout(cols: number, rows: number): { tilemap: TileKind[][]; plots: Plot[] } {
  const tilemap = createGrassMap(cols, rows);
  const rx1 = Math.floor(cols / 3), rx2 = Math.floor((2 * cols) / 3), ry1 = Math.floor(rows / 3), ry2 = Math.floor((2 * rows) / 3);
  for (let x = 0; x < cols; x++) { if (ry1 >= 0 && ry1 < rows) tilemap[ry1][x] = "path"; if (ry2 >= 0 && ry2 < rows) tilemap[ry2][x] = "path"; }
  for (let y = 0; y < rows; y++) { if (rx1 >= 0 && rx1 < cols) tilemap[y][rx1] = "path"; if (rx2 >= 0 && rx2 < cols) tilemap[y][rx2] = "path"; }
  const plots: Plot[] = [];
  const xSegs: Array<[number, number]> = [[0, rx1 - 1], [rx1 + 1, rx2 - 1], [rx2 + 1, cols - 1]];
  const ySegs: Array<[number, number, string]> = [[0, ry1 - 1, "North"], [ry2 + 1, rows - 1, "South"]];
  let id = 1;
  for (const [y0, y1, yL] of ySegs) { if (y0 > y1) continue;
    for (let xi = 0; xi < xSegs.length; xi++) { const s = xSegs[xi]; if (!s) continue;
      const [x0, x1] = s; if (x0 > x1) continue; const w = x1 - x0 + 1, h = y1 - y0 + 1; if (w <= 0 || h <= 0) continue;
      const xL = xi === 0 ? "West" : xi === 1 ? "Central" : "East";
      plots.push({ id: `plot_${id}`, name: `${yL} ${xL} Plot`, x: x0, y: y0, width: w, height: h, type: "plot" }); id++;
    } }
  return { tilemap, plots };
}
export function getForestLayout(cols: number, rows: number): { tilemap: TileKind[][]; plots: Plot[] } {
  const tilemap = createGrassMap(cols, rows);
  for (let x = 0; x < cols; x++) { tilemap[0][x] = "tree"; if (rows > 1) tilemap[rows - 1][x] = "tree"; }
  for (let y = 0; y < rows; y++) { tilemap[y][0] = "tree"; tilemap[y][cols - 1] = "tree"; }
  let clearings: Plot[];
  if (cols === 15 && rows === 12) {
    clearings = [
      { id: "clearing_1", name: "Northwest Clearing", x: 2, y: 2, width: 4, height: 3, type: "clearing" },
      { id: "clearing_2", name: "Northeast Clearing", x: 10, y: 2, width: 4, height: 3, type: "clearing" },
      { id: "clearing_3", name: "Southwest Clearing", x: 2, y: 7, width: 5, height: 3, type: "clearing" },
      { id: "clearing_4", name: "Southeast Clearing", x: 9, y: 8, width: 4, height: 3, type: "clearing" },
    ];
  } else {
    const wA = Math.max(FOREST_CLEARING_MIN_SIZE, Math.floor(cols * FOREST_CLEARING_W_FACTOR_A)), hA = Math.max(FOREST_CLEARING_MIN_SIZE, Math.floor(rows * FOREST_CLEARING_H_FACTOR)), wB = Math.max(FOREST_CLEARING_MIN_SIZE, Math.floor(cols * FOREST_CLEARING_W_FACTOR_B));
    clearings = [
      { id: "clearing_1", name: "Northwest Clearing", x: 1, y: 1, width: wA, height: hA, type: "clearing" },
      { id: "clearing_2", name: "Northeast Clearing", x: Math.max(1, cols - wA - 1), y: 1, width: wA, height: hA, type: "clearing" },
      { id: "clearing_3", name: "Southwest Clearing", x: 1, y: Math.max(1, rows - hA - 1), width: wB, height: hA, type: "clearing" },
      { id: "clearing_4", name: "Southeast Clearing", x: Math.max(1, cols - wA - 1), y: Math.max(1, rows - hA), width: wA, height: hA, type: "clearing" },
    ];
    for (const c of clearings) { if (c.x + c.width >= cols) c.width = cols - c.x - 1; if (c.y + c.height >= rows) c.height = rows - c.y - 1; if (c.x < 1) c.x = 1; if (c.y < 1) c.y = 1; }
  }
  for (const c of clearings) for (let dy = 0; dy < c.height; dy++) for (let dx = 0; dx < c.width; dx++) {
    const xx = c.x + dx, yy = c.y + dy; if (yy >= 0 && yy < rows && xx >= 0 && xx < cols) tilemap[yy][xx] = "grass";
  }
  if (cols === 15 && rows === 12) {
    const fixed: Array<[number, number]> = [[5,5],[6,5],[8,3],[5,3],[7,5],[12,6],[11,6],[4,6],[6,9],[13,5],[3,5],[7,10]];
    for (const [x, y] of fixed) { if (isInPlot(x, y, clearings)) continue; if (y > 0 && y < rows - 1 && x > 0 && x < cols - 1) tilemap[y][x] = "tree"; }
  } else {
    for (let y = 1; y < rows - 1; y++) for (let x = 1; x < cols - 1; x++) {
      if (isInPlot(x, y, clearings)) continue; const v = (x * FOREST_HASH_A + y * FOREST_HASH_B + (x * y) % FOREST_HASH_MOD) % FOREST_HASH_RANGE; if (v < FOREST_TREE_DENSITY_THRESHOLD) tilemap[y][x] = "tree";
    }
  }
  for (const c of clearings) for (let dy = 0; dy < c.height; dy++) for (let dx = 0; dx < c.width; dx++) {
    const xx = c.x + dx, yy = c.y + dy; if (yy > 0 && yy < rows - 1 && xx > 0 && xx < cols - 1) tilemap[yy][xx] = "grass";
  }
  return { tilemap, plots: clearings };
}
function getFallbackLayout(cols: number, rows: number): { tilemap: TileKind[][]; plots: Plot[] } {
  return { tilemap: createGrassMap(cols, rows), plots: [] };
}
export function getLayout(template: TemplateType, size: { cols: number; rows: number }): WorldLayout | null {
  const { cols, rows } = size; if (cols < WORLD_COLS_MIN || cols > WORLD_COLS_MAX || rows < WORLD_ROWS_MIN || rows > WORLD_ROWS_MAX) return null;
  let tilemap: TileKind[][]; let plots: Plot[];
  if (template === "town") { const r = getTownLayout(cols, rows); tilemap = r.tilemap; plots = r.plots; }
  else if (template === "forest") { const r = getForestLayout(cols, rows); tilemap = r.tilemap; plots = r.plots; }
  else if (template === "desert" || template === "school") { const r = getFallbackLayout(cols, rows); tilemap = r.tilemap; plots = r.plots; }
  else return null;
  const spawn = ensureWalkableSpawn(tilemap, { x: Math.floor(cols / 2), y: Math.floor(rows / 2) });
  return { tilemap: cloneMap(tilemap), plots, spawn };
}
export function getPlotsForTemplate(template: TemplateType, size: { cols: number; rows: number }): Plot[] {
  const l = getLayout(template, size); return l ? l.plots : [];
}
