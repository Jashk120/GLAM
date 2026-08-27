import type { World } from "../types/scenario";

export const TILE = 32;

type ThemeColors = { base: number; accent: number; path: number };

const THEME_MAP: Record<string, ThemeColors> = {
  town: { base: 0x8fd35a, accent: 0x7ec850, path: 0xd9c98a },
  forest: { base: 0x4e9e2e, accent: 0x3a7d22, path: 0xc9b978 },
  desert: { base: 0xe8d49a, accent: 0xd4b97a, path: 0xc9b07a },
  school: { base: 0xa8d8ea, accent: 0x8ecae6, path: 0xddd8c4 },
};

function themeColors(world: World): ThemeColors {
  const key = world.theme ?? world.template;
  return THEME_MAP[key] ?? THEME_MAP["town"] ?? { base: 0x8fd35a, accent: 0x7ec850, path: 0xd9c98a };
}

function variation(x: number, y: number): number {
  return (x * 928371 + y * 12923) % 100;
}

export class WorldRenderer {
  private world: World;
  private colors: ThemeColors;

  constructor(world: World) {
    this.world = world;
    this.colors = themeColors(world);
  }

  render(g: Phaser.GameObjects.Graphics): void {
    const { cols, rows } = this.world.size;
    const { base, accent, path } = this.colors;
    const tmpl = this.world.template;

    for (let y = 0; y < rows; y++) {
      for (let x = 0; x < cols; x++) {
        const seed = variation(x, y);
        let col = base;
        if (tmpl === "forest") {
          col = seed < 15 ? accent : base;
        } else if (tmpl === "desert") {
          col = seed < 18 ? path : base;
          if (seed < 5) col = accent;
        } else if (tmpl === "school") {
          col = seed < 12 ? accent : base;
          if (seed < 4) col = path;
        } else {
          // town
          col = seed < 10 ? path : base;
          if (seed < 3) col = accent;
        }
        g.fillStyle(col, 1);
        g.fillRect(x * TILE, y * TILE, TILE, TILE);
        // subtle border variation
        if (seed < 6) {
          g.fillStyle(0x000000, 0.06);
          g.fillRect(x * TILE, y * TILE, TILE, 2);
        }
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

  getPixelSize(): { w: number; h: number } {
    return { w: this.world.size.cols * TILE, h: this.world.size.rows * TILE };
  }
}
