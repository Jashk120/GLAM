import { stageObjectCount, consoleModel } from "./ArenaPresentation.js";
import { validateArenaScript } from "./ArenaFlow.js";

let passed = 0;
let failed = 0;

function ok(name: string, fn: () => void): void {
  try {
    fn();
    passed++;
    console.log(`✅ ${name}`);
  } catch (error) {
    failed++;
    console.error(`❌ ${name}: ${error instanceof Error ? error.message : String(error)}`);
  }
}

const script = validateArenaScript({
  version: "1", id: "adding-apples", title: "Adding Apples", theme: "meadow",
  cast: { student: { variant: "boy" }, mascot: { id: "nova-fox", name: "Nova" } },
  flow: {
    start: "teach",
    nodes: {
      teach: {
        id: "teach", type: "teaching",
        stage: { visual: { type: "countingObjects", object: "apple", count: 3 }, motion: [{ type: "add", count: 2, target: "stage" }], caption: "Count with Nova." },
        next: "answer",
      },
      answer: {
        id: "answer", type: "multipleChoice", prompt: "How many?",
        options: [{ id: "four", text: "4" }, { id: "five", text: "5" }], correctOptionIds: ["five"],
        feedback: { correct: { mascotText: "Correct!", next: "complete" }, incorrect: { mascotText: "Try again." } },
      },
      complete: { id: "complete", type: "complete", title: "Done", summary: "Nice work." },
    },
  },
});

ok("stage object count changes only after the teaching motion is revealed", () => {
  const node = script.flow.nodes["teach"];
  if (node.type !== "teaching") throw new Error("expected teaching node");
  if (stageObjectCount(node, false) !== 3) throw new Error("initial count incorrect");
  if (stageObjectCount(node, true) !== 5) throw new Error("revealed count incorrect");
});

ok("console model exposes a readable teaching caption", () => {
  const node = script.flow.nodes["teach"];
  const model = consoleModel(node);
  if (model.kind !== "reading" || !model.text.includes("Count with Nova")) throw new Error("teaching console model incorrect");
});

ok("console model exposes MCQ options and a submit action", () => {
  const node = script.flow.nodes["answer"];
  const model = consoleModel(node);
  if (model.kind !== "multipleChoice" || model.options.length !== 2) throw new Error("MCQ console model incorrect");
});

console.log(`\n${passed} passed, ${failed} failed of ${passed + failed}`);
if (failed > 0) throw new Error(`${failed} test(s) failed`);
