// Centralized timing constants — previously scattered in Game.ts, AssetStreamer.ts, TeacherUI.ts, main.ts

export const TOAST_DURATION_MS = 2600;
export const MOVEMENT_COOLDOWN_MS = 120;
export const ASSET_FETCH_DELAY_MS = 50;
export const STATUS_CLEAR_DELAY_MS = 3000;

export const TOAST_COLORS = {
  success: "#2e7d32",
  error: "#c62828",
} as const;
