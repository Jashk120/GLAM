import type { ArenaNode, MultipleChoiceNode, TeachingNode } from "./arenaTypes";

export type ArenaConsoleModel =
  | { kind: "reading"; text: string; continueLabel: string }
  | { kind: "multipleChoice"; prompt: string; options: MultipleChoiceNode["options"] }
  | { kind: "complete"; title: string; summary: string };

export function stageObjectCount(node: TeachingNode, motionRevealed: boolean): number {
  const additions = motionRevealed ? (node.stage.motion ?? []).reduce((total, motion) => total + motion.count, 0) : 0;
  return node.stage.visual.count + additions;
}

export function consoleModel(node: ArenaNode): ArenaConsoleModel {
  if (node.type === "dialogue") {
    return { kind: "reading", text: node.text, continueLabel: node.continueLabel ?? "Continue" };
  }
  if (node.type === "teaching") {
    return { kind: "reading", text: node.stage.caption ?? "Watch the teaching stage, then continue when you are ready.", continueLabel: "Continue" };
  }
  if (node.type === "multipleChoice") {
    return { kind: "multipleChoice", prompt: node.prompt, options: node.options };
  }
  return { kind: "complete", title: node.title, summary: node.summary };
}
