import type {
  ArenaAnswerResult,
  ArenaNode,
  ArenaScript,
  AvatarVariant,
  DialogueNode,
  MultipleChoiceNode,
  NodeId,
  TeachingNode,
} from "./arenaTypes";

const FORBIDDEN_FIELDS = new Set(["code", "script", "component", "bundle"]);
const THEMES = new Set(["meadow", "town", "forest", "school", "desert"]);
const AVATAR_VARIANTS = new Set<AvatarVariant>(["girl", "boy", "neutral"]);
const SPEAKERS = new Set(["student", "mascot", "narrator"]);
const EMOTIONS = new Set(["neutral", "happy", "thinking", "encouraging", "celebrating", "concerned"]);

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function assert(condition: boolean, message: string): asserts condition {
  if (!condition) throw new Error(message);
}

function stringValue(value: unknown, path: string, maxLength = 500): string {
  assert(typeof value === "string" && value.trim().length > 0 && value.length <= maxLength, `${path} must be a non-empty string`);
  return value;
}

function assertForbiddenFields(value: unknown, path: string): void {
  if (Array.isArray(value)) {
    value.forEach((item, index) => assertForbiddenFields(item, `${path}[${index}]`));
    return;
  }
  if (!isRecord(value)) return;
  for (const [key, child] of Object.entries(value)) {
    assert(!FORBIDDEN_FIELDS.has(key), `forbidden field "${path ? `${path}.` : ""}${key}" not allowed`);
    assertForbiddenFields(child, path ? `${path}.${key}` : key);
  }
}

function validateCast(value: unknown): void {
  assert(isRecord(value), "arena.cast must be an object");
  assert(isRecord(value["student"]), "arena.cast.student must be an object");
  assert(isRecord(value["mascot"]), "arena.cast.mascot must be an object");
  const student = value["student"];
  const mascot = value["mascot"];
  assert(typeof student["variant"] === "string" && AVATAR_VARIANTS.has(student["variant"] as AvatarVariant), "arena.cast.student.variant is not valid");
  if (student["name"] !== undefined) stringValue(student["name"], "arena.cast.student.name", 64);
  if (student["expression"] !== undefined) assert(typeof student["expression"] === "string" && EMOTIONS.has(student["expression"]), "arena.cast.student.expression is not valid");
  assert(mascot["id"] === "nova-fox", "arena.cast.mascot.id must be nova-fox");
  stringValue(mascot["name"], "arena.cast.mascot.name", 64);
  if (mascot["expression"] !== undefined) assert(typeof mascot["expression"] === "string" && EMOTIONS.has(mascot["expression"]), "arena.cast.mascot.expression is not valid");
  if (mascot["side"] !== undefined) assert(mascot["side"] === "right", "arena.cast.mascot.side must be right");
}

function validateNode(nodeId: string, value: unknown): void {
  assert(isRecord(value), `arena.flow.nodes.${nodeId} must be an object`);
  assert(value["id"] === nodeId, `arena.flow.nodes.${nodeId}.id must match its key`);
  const type = value["type"];
  assert(typeof type === "string", `arena.flow.nodes.${nodeId}.type is required`);
  if (type === "dialogue") {
    const node = value as unknown as DialogueNode;
    assert(SPEAKERS.has(node.speaker), `arena.flow.nodes.${nodeId}.speaker is not valid`);
    stringValue(node.text, `arena.flow.nodes.${nodeId}.text`, 1000);
    stringValue(node.next, `arena.flow.nodes.${nodeId}.next`, 64);
  } else if (type === "teaching") {
    const node = value as unknown as TeachingNode;
    assert(isRecord(node.stage), `arena.flow.nodes.${nodeId}.stage must be an object`);
    const visual = node.stage.visual;
    assert(isRecord(visual) && visual.type === "countingObjects", `arena.flow.nodes.${nodeId}.stage.visual is not valid`);
    assert(visual.object === "apple" || visual.object === "coin" || visual.object === "star", `arena.flow.nodes.${nodeId}.stage.visual.object is not valid`);
    assert(typeof visual.count === "number" && Number.isInteger(visual.count) && visual.count >= 0 && visual.count <= 20, `arena.flow.nodes.${nodeId}.stage.visual.count must be 0-20`);
    if (node.stage.motion !== undefined) {
      assert(Array.isArray(node.stage.motion) && node.stage.motion.length <= 5, `arena.flow.nodes.${nodeId}.stage.motion must have at most 5 items`);
      for (const motion of node.stage.motion) {
        assert(isRecord(motion) && motion.type === "add" && motion.target === "stage" && typeof motion.count === "number" && Number.isInteger(motion.count) && motion.count > 0 && motion.count <= 20, `arena.flow.nodes.${nodeId}.stage.motion is not valid`);
      }
    }
    stringValue(node.next, `arena.flow.nodes.${nodeId}.next`, 64);
  } else if (type === "multipleChoice") {
    const node = value as unknown as MultipleChoiceNode;
    stringValue(node.prompt, `arena.flow.nodes.${nodeId}.prompt`, 500);
    assert(Array.isArray(node.options) && node.options.length >= 2 && node.options.length <= 5, `arena.flow.nodes.${nodeId}.options must have 2-5 items`);
    const optionIds = new Set<string>();
    for (const option of node.options) {
      assert(isRecord(option), `arena.flow.nodes.${nodeId}.options item must be an object`);
      const optionId = stringValue(option["id"], `arena.flow.nodes.${nodeId}.options.id`, 64);
      assert(!optionIds.has(optionId), `arena.flow.nodes.${nodeId}.options ids must be unique`);
      optionIds.add(optionId);
      stringValue(option["text"], `arena.flow.nodes.${nodeId}.options.text`, 200);
    }
    assert(Array.isArray(node.correctOptionIds) && node.correctOptionIds.length >= 1, `arena.flow.nodes.${nodeId}.correctOptionIds is required`);
    for (const correctId of node.correctOptionIds) assert(optionIds.has(correctId), `arena.flow.nodes.${nodeId}.correctOptionIds contains an unknown option`);
    assert(isRecord(node.feedback) && isRecord(node.feedback.correct) && isRecord(node.feedback.incorrect), `arena.flow.nodes.${nodeId}.feedback requires correct and incorrect messages`);
    stringValue(node.feedback.correct.mascotText, `arena.flow.nodes.${nodeId}.feedback.correct.mascotText`, 500);
    stringValue(node.feedback.incorrect.mascotText, `arena.flow.nodes.${nodeId}.feedback.incorrect.mascotText`, 500);
  } else if (type === "complete") {
    stringValue(value["title"], `arena.flow.nodes.${nodeId}.title`, 120);
    stringValue(value["summary"], `arena.flow.nodes.${nodeId}.summary`, 1000);
  } else {
    throw new Error(`arena.flow.nodes.${nodeId}.type "${type}" is not valid`);
  }
}

function nextNodeIds(node: ArenaNode): NodeId[] {
  if (node.type === "dialogue" || node.type === "teaching") return [node.next];
  if (node.type === "multipleChoice") return [node.feedback.correct.next].filter((id): id is NodeId => Boolean(id));
  return [];
}

export function validateArenaScript(value: unknown): ArenaScript {
  assertForbiddenFields(value, "arena");
  assert(isRecord(value), "arena must be an object");
  assert(value["version"] === "1", "arena.version must be 1");
  stringValue(value["id"], "arena.id", 64);
  stringValue(value["title"], "arena.title", 120);
  assert(typeof value["theme"] === "string" && THEMES.has(value["theme"]), "arena.theme is not valid");
  validateCast(value["cast"]);
  assert(isRecord(value["flow"]), "arena.flow must be an object");
  const flow = value["flow"];
  const start = stringValue(flow["start"], "arena.flow.start", 64);
  assert(isRecord(flow["nodes"]), "arena.flow.nodes must be an object");
  const rawNodes = flow["nodes"];
  const entries = Object.entries(rawNodes);
  assert(entries.length >= 1 && entries.length <= 30, "arena.flow.nodes must have 1-30 nodes");
  for (const [nodeId, node] of entries) validateNode(nodeId, node);
  assert(Object.prototype.hasOwnProperty.call(rawNodes, start), "arena.flow.start does not exist");
  const script = value as unknown as ArenaScript;
  for (const node of Object.values(script.flow.nodes)) {
    for (const next of nextNodeIds(node)) assert(Object.prototype.hasOwnProperty.call(script.flow.nodes, next), `arena.flow next node "${next}" does not exist`);
  }
  return script;
}

export class ArenaFlow {
  private activeId: NodeId;

  constructor(private readonly script: ArenaScript) {
    this.activeId = script.flow.start;
  }

  get current(): ArenaNode {
    return this.script.flow.nodes[this.activeId];
  }

  continue(): void {
    const node = this.current;
    if (node.type !== "dialogue" && node.type !== "teaching") throw new Error(`node "${node.id}" cannot continue`);
    this.activeId = node.next;
  }

  submitAnswer(answerIds: string[]): ArenaAnswerResult {
    const node = this.current;
    if (node.type !== "multipleChoice") throw new Error(`node "${node.id}" does not accept answers`);
    const correct = answerIds.length === node.correctOptionIds.length && answerIds.every((id) => node.correctOptionIds.includes(id));
    const feedback = correct ? node.feedback.correct : node.feedback.incorrect;
    if (correct && feedback.next) this.activeId = feedback.next;
    return { correct, feedback };
  }
}
