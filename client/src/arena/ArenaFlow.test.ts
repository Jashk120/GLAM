import { ArenaFlow, validateArenaScript } from "./ArenaFlow.js";

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

function expectThrow(fn: () => void, message: string): void {
  try {
    fn();
  } catch (error) {
    const text = error instanceof Error ? error.message : String(error);
    if (text.includes(message)) return;
    throw new Error(`expected "${message}" in "${text}"`);
  }
  throw new Error(`expected throw containing "${message}"`);
}

function currentId(flow: ArenaFlow): string {
  return flow.current.id;
}

function arenaScript(): Record<string, unknown> {
  return {
    version: "1",
    id: "adding-apples",
    title: "Adding Apples",
    theme: "meadow",
    cast: {
      student: { variant: "girl", name: "Ava" },
      mascot: { id: "nova-fox", name: "Nova" },
    },
    flow: {
      start: "welcome",
      nodes: {
        welcome: {
          id: "welcome", type: "dialogue", speaker: "mascot",
          text: "Let us count apples together.", next: "show-apples",
        },
        "show-apples": {
          id: "show-apples", type: "teaching",
          stage: {
            visual: { type: "countingObjects", object: "apple", count: 3 },
            motion: [{ type: "add", count: 2, target: "stage" }],
            caption: "Three plus two makes five.", replayable: true,
          },
          next: "answer",
        },
        answer: {
          id: "answer", type: "multipleChoice", prompt: "How many apples now?",
          options: [
            { id: "four", text: "4" }, { id: "five", text: "5" },
            { id: "six", text: "6" }, { id: "seven", text: "7" },
          ],
          correctOptionIds: ["five"],
          feedback: {
            correct: { mascotText: "Correct!", next: "complete" },
            incorrect: { mascotText: "Try counting again.", retryPolicy: "afterHint" },
          },
        },
        complete: { id: "complete", type: "complete", title: "Great work!", summary: "You added apples." },
      },
    },
  };
}

ok("accepts a complete arena script", () => {
  validateArenaScript(arenaScript());
});

ok("rejects a missing next-node reference", () => {
  const script = arenaScript();
  const nodes = ((script["flow"] as Record<string, unknown>)["nodes"] as Record<string, Record<string, unknown>>);
  nodes["welcome"]["next"] = "missing";
  expectThrow(() => validateArenaScript(script), "does not exist");
});

ok("rejects arbitrary executable fields", () => {
  const script = arenaScript();
  script["code"] = "alert(1)";
  expectThrow(() => validateArenaScript(script), "forbidden field");
});

ok("rejects MCQ activities without a correct option", () => {
  const script = arenaScript();
  const nodes = ((script["flow"] as Record<string, unknown>)["nodes"] as Record<string, Record<string, unknown>>);
  nodes["answer"]["correctOptionIds"] = ["unknown"];
  expectThrow(() => validateArenaScript(script), "correctOptionIds");
});

ok("rejects a teaching motion without its fixed stage target", () => {
  const script = arenaScript();
  const nodes = ((script["flow"] as Record<string, unknown>)["nodes"] as Record<string, Record<string, unknown>>);
  const stage = nodes["show-apples"]["stage"] as Record<string, unknown>;
  const motions = stage["motion"] as Array<Record<string, unknown>>;
  delete motions[0]["target"];
  expectThrow(() => validateArenaScript(script), "stage.motion is not valid");
});

ok("flow advances through dialogue and teaching nodes", () => {
  const flow = new ArenaFlow(validateArenaScript(arenaScript()));
  if (currentId(flow) !== "welcome") throw new Error("start node not opened");
  flow.continue();
  if (currentId(flow) !== "show-apples") throw new Error("dialogue did not advance");
  flow.continue();
  if (currentId(flow) !== "answer") throw new Error("teaching did not advance");
});

ok("flow keeps an incorrect answer active and advances a correct answer", () => {
  const flow = new ArenaFlow(validateArenaScript(arenaScript()));
  flow.continue();
  flow.continue();
  if (flow.submitAnswer(["four"]).correct) throw new Error("incorrect answer marked correct");
  if (currentId(flow) !== "answer") throw new Error("incorrect answer advanced flow");
  if (!flow.submitAnswer(["five"]).correct) throw new Error("correct answer marked incorrect");
  if (currentId(flow) !== "complete") throw new Error("correct answer did not complete flow");
});

console.log(`\n${passed} passed, ${failed} failed of ${passed + failed}`);
if (failed > 0) throw new Error(`${failed} test(s) failed`);
