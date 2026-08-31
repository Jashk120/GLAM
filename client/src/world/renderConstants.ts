// Centralized world rendering constants — previously scattered in WorldRenderer 5-25, 95-138
// Keeps TILE, theme fallbacks, tile colors, opacities and geometry in one place.
// WorldRenderer + layouts import from here so no hardcoded duplicates.

import type { TileKind } from "./layouts";

export const TILE = 32;

export type ThemeFallback = { base: number; accent: number; path: number };

export const TILE_COLORS: Record<TileKind, number> = {
  grass: 0x8fd35a,
  path: 0xd9c98a,
  tree: 0x2d5a1e,
  water: 0x3b7dd8,
};

export const THEME_FALLBACK: Record<string, ThemeFallback> = {
  town: { base: 0x8fd35a, accent: 0x7ec850, path: 0xd9c98a },
  forest: { base: 0x4e9e2e, accent: 0x3a7d22, path: 0xc9b978 },
  desert: { base: 0xe8d49a, accent: 0xd4b97a, path: 0xc9b07a },
  school: { base: 0xa8d8ea, accent: 0x8ecae6, path: 0xddd8c4 },
};

export const DEFAULT_THEME: ThemeFallback = { base: 0x8fd35a, accent: 0x7ec850, path: 0xd9c98a };

// Opacities used in rendering
export const RENDER_OPACITY = {
  waterWave: 0.9,
  plotOutlineInner: 0.18,
  plotOutlineOuter: 0.22,
  grid: 0.07,
  trunkHighlight: 0.9,
  foliageHighlight: 0.9,
} as const;

// Geometry constants
export const RENDER_GEOMETRY = {
  waterStripeWidth: 24,
  waterStripeHeight: 3,
  waterStripeOffset1: { x: 4, y: 8 },
  waterStripeOffset2: { x: 2, y: 20 },
  plotInset: 1,
  plotInsetSize: 2,
  gridLineWidth: 1,
  treeTrunk: { x: 13, y: 18, w: 6, h: 14, color: 0x5c3a1e },
  treeFoliage: { x: 16, y: 14, r: 14, color: 0x1e4a10 },
  treeHighlight: { x: 16, y: 12, r: 9, color: 0x2d6a1e },
  waterColor: 0x3b7dd8,
  waterWaveColor: 0x5aa0f0,
  plotInnerColor: 0xffffff,
  plotOuterColor: 0xffd700,
  gridColor: 0x000000,
} as const;
