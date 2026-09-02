export type ArenaTheme = "meadow" | "town" | "forest" | "school" | "desert";
export type AvatarVariant = "girl" | "boy" | "neutral";
export type Speaker = "student" | "mascot" | "narrator";
export type Emotion = "neutral" | "happy" | "thinking" | "encouraging" | "celebrating" | "concerned";
export type NodeId = string;

export interface ArenaCast {
  student: { variant: AvatarVariant; name?: string; expression?: Emotion };
  mascot: { id: "nova-fox"; name: string; expression?: Emotion; side?: "right" };
}

export interface CountingVisual {
  type: "countingObjects";
  object: "apple" | "coin" | "star";
  count: number;
}

export interface AddMotion {
  type: "add";
  count: number;
  target: "stage";
}

export interface TeachingStage {
  visual: CountingVisual;
  motion?: AddMotion[];
  caption?: string;
  replayable?: boolean;
}

export interface DialogueNode {
  id: NodeId;
  type: "dialogue";
  speaker: Speaker;
  text: string;
  emotion?: Emotion;
  mode?: "console" | "bubble" | "both";
  continueLabel?: string;
  next: NodeId;
}

export interface TeachingNode {
  id: NodeId;
  type: "teaching";
  stage: TeachingStage;
  next: NodeId;
}

export interface ArenaOption {
  id: string;
  text: string;
}

export interface FeedbackMessage {
  mascotText: string;
  next?: NodeId;
  retryPolicy?: "immediate" | "afterHint" | "continue";
}

export interface MultipleChoiceNode {
  id: NodeId;
  type: "multipleChoice";
  prompt: string;
  options: ArenaOption[];
  correctOptionIds: string[];
  feedback: { correct: FeedbackMessage; incorrect: FeedbackMessage };
}

export interface CompleteNode {
  id: NodeId;
  type: "complete";
  title: string;
  summary: string;
}

export type ArenaNode = DialogueNode | TeachingNode | MultipleChoiceNode | CompleteNode;

export interface ArenaScript {
  version: "1";
  id: string;
  title: string;
  theme: ArenaTheme;
  cast: ArenaCast;
  flow: { start: NodeId; nodes: Record<NodeId, ArenaNode> };
}

export interface ArenaAnswerResult {
  correct: boolean;
  feedback: FeedbackMessage;
}
