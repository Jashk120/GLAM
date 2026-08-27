import type { World } from "../types/scenario";
import { getLayout, isSolidTile } from "./layouts";
import type { TileKind, WorldLayout } from "./layouts";

export const TILE = 32;

type ThemeFallback = { base: number; accent: number; path: number };

const TILE_COLORS: Record<TileKind, number> = {
  grass: 0x8fd35a,
  path: 0xd9c98a,
  tree: 0x2d5a1e,
  water: 0x3b7dd8,
};

const THEME_FALLBACK: Record<string, ThemeFallback> = {
  town: { base: 0x8fd35a, accent: 0x7ec850, path: 0xd9c98a },
  forest: { base: 0x4e9e2e, accent: 0x3a7d22, path: 0xc9b978 },
  desert: { base: 0xe8d49a, accent: 0xd4b97a, path: 0xc9b07a },
  school: { base: 0xa8d8ea, accent: 0x8ecae6, path: 0xddd8c4 },
};

function themeColors(world: World): ThemeFallback {
  const key = world.theme ?? world.template;
  return THEME_FALLBACK[key] ?? THEME_FALLBACK["town"] ?? { base: 0x8fd35a, accent: 0x7ec850, path: 0xd9c98a };
}

export class WorldRenderer {
  private world: World;
  private layout: WorldLayout | null;
  private tilemap: TileKind[][];
  private fallback: ThemeFallback;

  constructor(world: World, layout?: WorldLayout | null) {
    this.world = world;
    this.fallback = themeColors(world);
    if (layout) {
      this.layout = layout;
      this.tilemap = layout.tilemap;
    } else {
      const computed = getLayout(world.template, world.size);
      this.layout = computed;
      this.tilemap = computed ? computed.tilemap : this.buildFallbackMap();
    }
  }

  private buildFallbackMap(): TileKind[][] {
    const { cols, rows } = this.world.size;
    const m: TileKind[][] = [];
    for (let y = 0; y < rows; y++) {
      const row: TileKind[] = [];
      for (let x = 0; x < cols; x++) row.push("grass");
      m.push(row);
    }
    return m;
  }

  getTileAt(x: number, y: number): TileKind | null {
    const row = this.tilemap[y];
    if (!row) return null;
    const v = row[x];
    return v ?? null;
  }

  isSolidAt(x: number, y: number): boolean {
    const k = this.getTileAt(x, y);
    if (k === null) return true;
    return isSolidTile(k);
  }

  getLayout(): WorldLayout | null {
    return this.layout;
  }

  getTilemap(): TileKind[][] {
    return this.tilemap;
  }

  render(g: Phaser.GameObjects.Graphics): void {
    const { cols, rows } = this.world.size;

    for (let y = 0; y < rows; y++) {
      for (let x = 0; x < cols; x++) {
        const kind: TileKind = this.tilemap[y]?.[x] ?? "grass";
        // Base fill per tile kind
        if (kind === "tree") {
          // tree tile: draw grass base then tree on top (keep walkable visual distinct but solid)
          g.fillStyle(TILE_COLORS.grass, 1);
          g.fillRect(x * TILE, y * TILE, TILE, TILE);
          this.drawTree(g, x, y);
        } else if (kind === "water") {
          g.fillStyle(TILE_COLORS.water, 1);
          g.fillRect(x * TILE, y * TILE, TILE, TILE);
          // wave stripes like pokemon demo
          g.fillStyle(0x5aa0f0, 0.9);
          g.fillRect(x * TILE + 4, y * TILE + 8, 24, 3);
          g.fillRect(x * TILE + 2, y * TILE + 20, 24, 3);
        } else {
          const col = TILE_COLORS[kind] ?? this.fallback.base;
          g.fillStyle(col, 1);
          g.fillRect(x * TILE, y * TILE, TILE, TILE);
        }
      }
    }

    // faint plot outlines (buildable/clearing areas)
    if (this.layout && this.layout.plots.length > 0) {
      for (const p of this.layout.plots) {
        g.lineStyle(1, 0xffffff, 0.18);
        g.strokeRect(p.x * TILE + 1, p.y * TILE + 1, p.width * TILE - 2, p.height * TILE - 2);
        // corner ticks for visibility
        g.lineStyle(1, 0xffd700, 0.22);
        g.strokeRect(p.x * TILE, p.y * TILE, p.width * TILE, p.height * TILE);
      }
    }

    // grid lines faint
    g.lineStyle(1, 0x000000, 0.07);
    for (let x = 0; x <= cols; x++) {
      g.lineBetween(x * TILE, 0, x * TILE, rows * TILE);
    }
    for (let y = 0; y <= rows; y++) {
      g.lineBetween(0, y * TILE, cols * TILE, y * TILE);
    }
  }

  private drawTree(g: Phaser.GameObjects.Graphics, x: number, y: number): void {
    const px = x * TILE;
    const py = y * TILE;
    // trunk
    g.fillStyle(0x5c3a1e, 1);
    g.fillRect(px + 13, py + 18, 6, 14);
    // foliage (circle)
    g.fillStyle(0x1e4a10, 1);
    g.fillCircle(px + 16, py + 14, 14);
    // highlight
    g.fillStyle(0x2d6a1e, 0.9);
    g.fillCircle(px + 16, py + 12, 9);
  }

  getPixelSize(): { w: number; h: number } {
    return { w: this.world.size.cols * TILE, h: this.world.size.rows * TILE };
  }
}
