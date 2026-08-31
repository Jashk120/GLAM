import type { World } from "../types/scenario";
import { getLayout, isSolidTile } from "./layouts";
import type { TileKind, WorldLayout } from "./layouts";
import {
  TILE,
  TILE_COLORS,
  THEME_FALLBACK,
  DEFAULT_THEME,
  RENDER_OPACITY,
  RENDER_GEOMETRY,
} from "./renderConstants";
import type { ThemeFallback } from "./renderConstants";

export { TILE };

function themeColors(world: World): ThemeFallback {
  const key = world.theme ?? world.template;
  return THEME_FALLBACK[key] ?? THEME_FALLBACK["town"] ?? DEFAULT_THEME;
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
          g.fillStyle(RENDER_GEOMETRY.waterWaveColor, RENDER_OPACITY.waterWave);
          g.fillRect(x * TILE + RENDER_GEOMETRY.waterStripeOffset1.x, y * TILE + RENDER_GEOMETRY.waterStripeOffset1.y, RENDER_GEOMETRY.waterStripeWidth, RENDER_GEOMETRY.waterStripeHeight);
          g.fillRect(x * TILE + RENDER_GEOMETRY.waterStripeOffset2.x, y * TILE + RENDER_GEOMETRY.waterStripeOffset2.y, RENDER_GEOMETRY.waterStripeWidth, RENDER_GEOMETRY.waterStripeHeight);
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
        g.lineStyle(RENDER_GEOMETRY.gridLineWidth, RENDER_GEOMETRY.plotInnerColor, RENDER_OPACITY.plotOutlineInner);
        g.strokeRect(p.x * TILE + RENDER_GEOMETRY.plotInset, p.y * TILE + RENDER_GEOMETRY.plotInset, p.width * TILE - RENDER_GEOMETRY.plotInsetSize, p.height * TILE - RENDER_GEOMETRY.plotInsetSize);
        // corner ticks for visibility
        g.lineStyle(RENDER_GEOMETRY.gridLineWidth, RENDER_GEOMETRY.plotOuterColor, RENDER_OPACITY.plotOutlineOuter);
        g.strokeRect(p.x * TILE, p.y * TILE, p.width * TILE, p.height * TILE);
      }
    }

    // grid lines faint
    g.lineStyle(RENDER_GEOMETRY.gridLineWidth, RENDER_GEOMETRY.gridColor, RENDER_OPACITY.grid);
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
    g.fillStyle(RENDER_GEOMETRY.treeTrunk.color, 1);
    g.fillRect(px + RENDER_GEOMETRY.treeTrunk.x, py + RENDER_GEOMETRY.treeTrunk.y, RENDER_GEOMETRY.treeTrunk.w, RENDER_GEOMETRY.treeTrunk.h);
    // foliage (circle)
    g.fillStyle(RENDER_GEOMETRY.treeFoliage.color, 1);
    g.fillCircle(px + RENDER_GEOMETRY.treeFoliage.x, py + RENDER_GEOMETRY.treeFoliage.y, RENDER_GEOMETRY.treeFoliage.r);
    // highlight
    g.fillStyle(RENDER_GEOMETRY.treeHighlight.color, RENDER_OPACITY.foliageHighlight);
    g.fillCircle(px + RENDER_GEOMETRY.treeHighlight.x, py + RENDER_GEOMETRY.treeHighlight.y, RENDER_GEOMETRY.treeHighlight.r);
  }

  getPixelSize(): { w: number; h: number } {
    return { w: this.world.size.cols * TILE, h: this.world.size.rows * TILE };
  }
}
