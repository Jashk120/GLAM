// Node polyfill for client modules that expect browser globals.
// Must be imported BEFORE ScenarioLoader / AssetStreamer.
const g = globalThis as unknown as Record<string, unknown>;

if (!g["localStorage"]) {
  const store = new Map<string, string>();
  g["localStorage"] = {
    getItem: (k: string) => store.get(k) ?? null,
    setItem: (k: string, v: string) => { store.set(k, String(v)); },
    removeItem: (k: string) => { store.delete(k); },
    clear: () => store.clear(),
  };
}
if (!g["window"]) {
  g["window"] = g;
}
const w = g["window"] as unknown as Record<string, unknown>;
if (!w["setTimeout"]) w["setTimeout"] = globalThis.setTimeout;
if (!w["clearTimeout"]) w["clearTimeout"] = globalThis.clearTimeout;

export {};
