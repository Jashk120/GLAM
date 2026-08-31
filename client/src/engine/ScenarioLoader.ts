import type { Scenario } from "../types/scenario";
import { assetStreamer } from "../assets/AssetStreamer";
import { validateScenarioObject } from "./scenarioValidation";

export async function loadScenario(source: string | object): Promise<Scenario> {
  let raw: unknown;
  if (typeof source === "string") {
    const res = await fetch(source);
    if (!res.ok) throw new Error(`Failed to fetch scenario: ${source} — ${res.status} ${res.statusText}`);
    raw = (await res.json()) as unknown;
  } else {
    raw = source;
  }
  validateScenarioObject(raw);
  const scenario = raw as Scenario;
  await assetStreamer.preloadScenarioAssets(scenario);
  return scenario;
}

export function validateScenarioSync(obj: unknown): Scenario {
  validateScenarioObject(obj);
  return obj as Scenario;
}

export { validateScenarioObject };
