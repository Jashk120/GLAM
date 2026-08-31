// Centralized entity rendering constants — previously scattered across 4 renderers + Game.ts
// Covers fallback dimensions, colors, fonts, opacities, geometry and depth.

export const ENTITY_DEPTH_HIGH = 1000;

export const ENTITY_MIN_DIMENSION = 32;

// Building geometry
export const BUILDING_ROOF_HEIGHT_GABLED = 18;
export const BUILDING_ROOF_HEIGHT_FLAT = 10;

export type RoofKind = "gabled" | "flat";
export interface BuildingStyle {
  wall: number;
  wallShade: number;
  roof: number;
  roofAccent: number;
  roofKind: RoofKind;
  door: number;
  cross?: boolean;
  columns?: boolean;
  awning?: boolean;
}

export const BUILDING_STYLES: Record<string, BuildingStyle> = {
  hospital: { wall: 0xeef2f7, wallShade: 0xc8d3e8, roof: 0xd32f2f, roofAccent: 0xa82222, roofKind: "flat", door: 0x5a4a3a, cross: true },
  bank: { wall: 0xd6dbe6, wallShade: 0xaab4c8, roof: 0x3a4a5e, roofAccent: 0x2a3a4e, roofKind: "flat", door: 0x3b2e1a, columns: true },
  shop_small: { wall: 0xf5e0b8, wallShade: 0xd9b88a, roof: 0xc0392b, roofAccent: 0x922b21, roofKind: "gabled", door: 0x5d4037, awning: true },
  shop_large: { wall: 0xf5e0b8, wallShade: 0xd9b88a, roof: 0xc0392b, roofAccent: 0x922b21, roofKind: "gabled", door: 0x5d4037, awning: true },
  house_small: { wall: 0xead8c0, wallShade: 0xc9b49a, roof: 0x8b3a24, roofAccent: 0x6d2e1c, roofKind: "gabled", door: 0x4a2e1a },
  house_large: { wall: 0xebe0cc, wallShade: 0xc8bca8, roof: 0x6d4c41, roofAccent: 0x4e342e, roofKind: "gabled", door: 0x3e2723 },
  school_main: { wall: 0xfff3c4, wallShade: 0xebd8a0, roof: 0x1a5da0, roofAccent: 0x12407a, roofKind: "flat", door: 0x5d4037 },
};

export const BUILDING_FALLBACK_STYLE: BuildingStyle = {
  wall: 0xe8dcc8,
  wallShade: 0xc8b8a0,
  roof: 0x7a3a2a,
  roofAccent: 0x5a2a1e,
  roofKind: "gabled",
  door: 0x4a2e1a,
};

// Shared rendering palette
export const ENTITY_COLORS = {
  shadow: 0x000000,
  buildingShadowOpacity: 0.22,
  characterShadowOpacity: 0.28,
  objectShadowOpacity: 0.2,
  wallBorder: 0x3a2e1e,
  doorHighlight: 0x2a1e12,
  doorKnob: 0xffd700,
  buildingDotFill: 0xffd700,
  buildingDotOpacity: 0.95,
  windowFill: 0x8ecae6,
  windowHighlight: 0xffffff,
  awningFill: 0xe8e8e8,
  objectBg: 0x2a2a44,
  objectBgOpacity: 0.92,
  objectBorder: 0xffd700,
  objectBorderOpacity: 0.65,
} as const;

export const BUILDING_GEOMETRY = {
  shadowEllipse: { wFactor: 0.9, h: 8, y: -2 },
  windowSize: 10,
  doorSize: { wLarge: 14, wSmall: 11, h: 16 },
  sign: { w: 28, h: 13, offsetY: 10 },
  overhangGabled: 5,
  overhangFlat: 3,
  highlightOpacity: 0.12,
} as const;

// Character constants
export const CHARACTER_FALLBACK = {
  defaultColor: "#3a6ea5",
  fallbackShirtColor: 0xc0392b,
  pantsColor: 0x2c3e50,
  skinColor: 0xe8b98a,
  skinStroke: 0x1a1a1a,
  capDefault: 0x3a2a1a,
  capTeacher: 0x2c3e50,
  capRanger: 0x2d5a1e,
  eyeColor: 0x000000,
  blushColor: 0xdd8a6a,
  shadow: 0x000000,
  shadowOpacity: 0.28,
  iconFallback: "🧑",
} as const;

export const CHARACTER_GEOMETRY = {
  avOffsetY: -28,
  headSize: { w: 12, h: 11 },
  bodySize: { w: 14, h: 10, y: -18 },
  capSize: { w: 12, h: 5, y: -30 },
  badgeSize: 9,
  labelOffsetY: -36,
} as const;

// Object constants
export const OBJECT_GEOMETRY = {
  shadowEllipse: { w: 16, h: 6, y: -2 },
  box: { x: -11, y: -20, w: 22, h: 16, r: 3 },
  emojiOffsetY: -12,
  emojiFontSize: "14px",
} as const;

// Fonts and labels
export const ENTITY_FONTS = {
  buildingLabel: "7px",
  buildingEmoji: "10px",
  characterLabel: "8px",
  characterBadge: "9px",
  playerLabel: "7px",
  objectEmoji: "14px",
} as const;

export const ENTITY_LABEL_STYLE = {
  buildingBg: "#00000099",
  characterBg: "#000000aa",
  playerBg: "#1a5da0",
  textColor: "#ffffff",
  padding: { left: 3, right: 3, top: 1, bottom: 1 },
} as const;

// Player-specific (was duplicated in Game.ts)
export const PLAYER_COLORS = {
  shirt: 0x2d7df6,
  pants: 0x2c3e50,
  skin: 0xffe4b8,
  cap: 0x1a5da0,
  shadowOpacity: 0.28,
} as const;

export const PLAYER_GEOMETRY = {
  labelOffsetY: -36,
} as const;
