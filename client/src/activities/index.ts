export { showDialogue } from "./DialogueActivity";
export { showMCQ } from "./MCQActivity";
export { showMath } from "./MathActivity";
export { showShop } from "./ShopActivity";
export { showInformation } from "./InformationActivity";

import type { Interaction } from "../types/scenario";

export type InteractionType = Interaction["type"];

export const INTERACTION_TYPES: InteractionType[] = ["dialogue", "mcq", "math", "shop", "information"];
