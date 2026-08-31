export type TemplateType = "town" | "forest" | "desert" | "school";

export interface Position {
  x: number;
  y: number;
}

export interface Size {
  cols: number;
  rows: number;
}

export interface Outcome {
  stat?: string;
  delta?: number;
  toast?: string;
}

export interface Region {
  id: string;
  name?: string;
  x?: number;
  y?: number;
  width?: number;
  height?: number;
  type?: string;
}

export interface World {
  template: TemplateType;
  spawn: Position;
  size: Size;
  theme?: string;
  regions?: Region[];
}

export interface Appearance {
  spriteId?: string;
  color?: string;
}

// ----- Interactions -----
export interface InteractionDialogue {
  type: "dialogue";
  text: string;
  speaker?: string;
  cooldown?: number;
  auto?: boolean;
  onCorrect?: Outcome;
  onWrong?: Outcome;
}

export interface MCQOption {
  text: string;
  correct: boolean;
  explanation?: string;
}

export interface InteractionMCQ {
  type: "mcq";
  question: string;
  options: MCQOption[];
  allowRetry?: boolean;
  cooldown?: number;
  auto?: boolean;
  onCorrect?: Outcome;
  onWrong?: Outcome;
}

export interface InteractionMath {
  type: "math";
  question: string;
  answer: number | string;
  tolerance?: number;
  hint?: string;
  cooldown?: number;
  auto?: boolean;
  onCorrect?: Outcome;
  onWrong?: Outcome;
}

export interface ShopItem {
  id?: string;
  name: string;
  price: number;
  icon?: string;
  description?: string;
}

export interface InteractionShop {
  type: "shop";
  currency?: string;
  items: ShopItem[];
  cooldown?: number;
  auto?: boolean;
  onCorrect?: Outcome;
  onWrong?: Outcome;
}

export interface InteractionInformation {
  type: "information";
  title?: string;
  content: string;
  image?: string;
  cooldown?: number;
  auto?: boolean;
  onCorrect?: Outcome;
  onWrong?: Outcome;
}

export type Interaction =
  | InteractionDialogue
  | InteractionMCQ
  | InteractionMath
  | InteractionShop
  | InteractionInformation;

// ----- Entities -----
export interface Character {
  id: string;
  name: string;
  profession?: string;
  appearance?: Appearance;
  position: Position;
  plot?: string;
  interaction?: Interaction;
}

export interface Building {
  id: string;
  typeAssetId: string;
  position: Position;
  width?: number;
  height?: number;
  plot?: string;
  interaction?: Interaction;
}

export interface ObjectEntity {
  id: string;
  assetId: string;
  position: Position;
  plot?: string;
  interaction?: Interaction;
}

export interface MissionTrigger {
  entityId?: string;
  interactionId?: string;
  auto?: boolean;
}

export type StatOperator = ">=" | ">" | "<=" | "<" | "=" | "==" | "!=";

export interface RequiredStat {
  stat: string;
  operator: StatOperator;
  target: number;
}

export interface InitialStats {
  coins?: number;
  lives?: number;
  score?: number;
}

export interface Mission {
  id: string;
  title: string;
  description: string;
  trigger?: MissionTrigger;
  checkAtEnd?: boolean;
  requiredStat?: RequiredStat;
  done?: boolean;
}

export interface Scenario {
  id: string;
  title: string;
  version?: string;
  world: World;
  initialStats?: InitialStats;
  characters: Character[];
  buildings: Building[];
  objects: ObjectEntity[];
  missions: Mission[];
}

// Helper for unified entity handling
export type EntityKind = "character" | "building" | "object";

export interface EntityRef {
  kind: EntityKind;
  id: string;
  position: Position;
  name?: string;
  interaction?: Interaction;
  solid: boolean;
  icon: string;
  width: number;
  height: number;
  raw: Character | Building | ObjectEntity;
}
