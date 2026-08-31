// canonical: schema/scenario.schema.json — $defs/interaction oneOf types
// Mirrored in server/scenario/interaction.go — keep in sync. CI should diff if drift.
// Single source of truth is the schema; these constants avoid hardcoded duplication.

export const INTERACTION_TYPES = ["dialogue", "mcq", "math", "shop", "information"] as const;

export type InteractionType = (typeof INTERACTION_TYPES)[number];

export const VALID_INTERACTION_TYPES: ReadonlySet<InteractionType> = new Set(INTERACTION_TYPES);

export function isValidInteractionType(t: string): t is InteractionType {
  return (VALID_INTERACTION_TYPES as Set<string>).has(t);
}
