// canonical defaults for player stats — referenced when scenario.initialStats missing
// Keep in sync with schema/scenario.schema.json $defs/initialStats defaults documentation.
// Also mirrored conceptually in server prompt docs but not hardcoded there.

export const DEFAULT_INITIAL_STATS = {
  coins: 40,
  lives: 3,
  score: 0,
} as const;

export type DefaultStats = typeof DEFAULT_INITIAL_STATS;
