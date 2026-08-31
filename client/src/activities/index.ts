export { showDialogue } from "./DialogueActivity";
export { showMCQ } from "./MCQActivity";
export { showMath } from "./MathActivity";
export { showShop } from "./ShopActivity";
export { showInformation } from "./InformationActivity";

// canonical: schema/scenario.schema.json — interaction types
// Re-export shared constants to avoid duplication; source is client/src/types/interactionTypes.ts
export { INTERACTION_TYPES, VALID_INTERACTION_TYPES, isValidInteractionType } from "../types/interactionTypes";
export type { InteractionType } from "../types/interactionTypes";
