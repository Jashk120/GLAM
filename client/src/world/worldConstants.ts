// canonical: schema/scenario.schema.json — world.size and position bounds
// Mirrored in server/world/constants.go — keep in sync. CI registry check covers drift.

export const WORLD_COLS_MIN = 8;
export const WORLD_COLS_MAX = 30;
export const WORLD_ROWS_MIN = 8;
export const WORLD_ROWS_MAX = 20;
export const WORLD_POS_MIN = 0;
export const WORLD_POS_MAX = 30;

// Forest layout — mirrored in server/world/forest.go
export const FOREST_TREE_DENSITY_THRESHOLD = 13; // v < 13 → ~13% density
export const FOREST_CLEARING_W_FACTOR_A = 0.27;
export const FOREST_CLEARING_H_FACTOR = 0.25;
export const FOREST_CLEARING_W_FACTOR_B = 0.33;
export const FOREST_CLEARING_MIN_SIZE = 2;
export const FOREST_HASH_A = 37;
export const FOREST_HASH_B = 71;
export const FOREST_HASH_MOD = 19;
export const FOREST_HASH_RANGE = 100;
