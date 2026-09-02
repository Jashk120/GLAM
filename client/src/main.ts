import "./style.css";
import Phaser from "phaser";
import { GameScene } from "./engine/Game";
import { loadScenario } from "./engine/ScenarioLoader";
import { assetStreamer } from "./assets/AssetStreamer";
import type { Scenario } from "./types/scenario";
import { STORAGE_KEYS } from "./assets/storageKeys";
import { TOAST_DURATION_MS, TOAST_COLORS } from "./engine/timingConstants";
import { TILE } from "./world/renderConstants";
import { API_CONFIG } from "./config/env";
import { ArenaRuntime } from "./arena/ArenaRuntime";

const statsBar = document.getElementById("statsBar") as HTMLElement;
const missionList = document.getElementById("missionList") as HTMLElement;
const promptEl = document.getElementById("prompt") as HTMLElement;
const modalEl = document.getElementById("modal") as HTMLElement;
const overlayEl = document.getElementById("modalOverlay") as HTMLElement;
const toastEl = document.getElementById("toast") as HTMLElement;
const titleEl = document.getElementById("scenarioTitle") as HTMLElement;
const selectEl = document.getElementById("scenarioSelect") as HTMLSelectElement;
const restartBtn = document.getElementById("restartBtn") as HTMLButtonElement;
const assetStatus = document.getElementById("assetStatus") as HTMLElement;
const arenaRoot = document.getElementById("arenaRoot") as HTMLElement;
const gameWrap = document.getElementById("gameWrap") as HTMLElement;
const missionPanel = document.getElementById("missionPanel") as HTMLElement;
const controlsHint = document.getElementById("controlsHint") as HTMLElement;

let gameScene: GameScene | null = null;
let currentScenario: Scenario | null = null;
let arenaRuntime: ArenaRuntime | null = null;
let currentSource = "/scenarios/example.json";

const generatedStore = new Map<string, Scenario>();

function showToast(msg: string, ok = true): void {
  toastEl.textContent = msg;
  toastEl.style.background = ok ? TOAST_COLORS.success : TOAST_COLORS.error;
  toastEl.style.display = "block";
  const existing = (toastEl as unknown as { _timer?: number })._timer;
  if (existing) window.clearTimeout(existing);
  (toastEl as unknown as { _timer: number })._timer = window.setTimeout(() => {
    toastEl.style.display = "none";
  }, TOAST_DURATION_MS);
}

const scene = new GameScene({
  statsBar,
  missionList,
  prompt: promptEl,
  modal: modalEl,
  overlay: overlayEl,
  toast: toastEl,
  titleEl,
});

const game = new Phaser.Game({
  type: Phaser.AUTO,
  parent: "game",
  backgroundColor: "#181828",
  width: 15 * TILE,
  height: 12 * TILE,
  scene: [scene],
  scale: {
    mode: Phaser.Scale.NONE,
    autoCenter: Phaser.Scale.CENTER_BOTH,
  },
});

game.events.on("ready", async () => {
  gameScene = scene;
  assetStatus.textContent = `Assets cached: ${assetStreamer.cachedCount()}/${assetStreamer.totalCount()}`;
  assetStreamer.onStatus((msg) => {
    assetStatus.textContent = msg;
  });

  const qp = new URLSearchParams(window.location.search);
  const scenarioParam = qp.get("scenario") ?? qp.get("id") ?? qp.get("s");
  if (scenarioParam) {
    try {
      await loadScenarioById(scenarioParam);
    } catch (err) {
      console.warn("[GLAM] Failed to load scenario from ?scenario=", scenarioParam, err);
      showToast(`Scenario "${scenarioParam}" not found — loaded example`, false);
      await bootScenario(currentSource);
    }
  } else {
    await bootScenario(currentSource);
  }

  await refreshScenarioList();
  updateAssetStatus(currentScenario);
  console.log("[GLAM] Play booted", currentSource);
});

overlayEl.addEventListener("click", (e) => {
  if (e.target === overlayEl) overlayEl.classList.remove("active");
});

async function bootScenario(source: string | object): Promise<void> {
  try {
    const sc = await loadScenario(source);
    currentScenario = sc;
    if (!gameScene) return;
    titleEl.textContent = sc.title;
    if (sc.arena) {
      gameWrap.hidden = true;
      missionPanel.hidden = true;
      controlsHint.hidden = true;
      gameScene.scene.pause();
      arenaRuntime?.destroy();
      arenaRuntime = new ArenaRuntime(arenaRoot, sc.arena);
      arenaRuntime.mount();
      updateAssetStatus(sc);
      return;
    }
    arenaRuntime?.destroy();
    arenaRuntime = null;
    gameWrap.hidden = false;
    missionPanel.hidden = false;
    controlsHint.hidden = false;
    gameScene.scene.resume();
    const { w, h } = { w: sc.world.size.cols * TILE, h: sc.world.size.rows * TILE };
    game.scale.resize(w, h);
    await gameScene.loadScenarioData(sc);
    updateAssetStatus(sc);
  } catch (err) {
    const msg = err instanceof Error ? err.message : String(err);
    showToast(`❌ Failed to load scenario: ${msg}`, false);
    console.error(err);
  }
}

async function loadScenarioById(id: string): Promise<void> {
  const clean = id.trim();
  if (!clean) throw new Error("empty id");

  if (generatedStore.has(`generated:${clean}`)) {
    const sc = generatedStore.get(`generated:${clean}`)!;
    await bootScenario(sc as unknown as object);
    return;
  }
  if (generatedStore.has(clean)) {
    const sc = generatedStore.get(clean)!;
    await bootScenario(sc as unknown as object);
    return;
  }

  const url = `${API_CONFIG.scenariosUrl}/${encodeURIComponent(clean)}`;
  const res = await fetch(url);
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    const detail = (body as { error?: string }).error ?? `${res.status} ${res.statusText}`;
    throw new Error(detail);
  }
  const body = (await res.json()) as { scenario?: unknown };
  if (!body.scenario) throw new Error("Server returned no scenario");
  const sc = body.scenario as Scenario;
  generatedStore.set(`generated:${sc.id}`, sc);
  generatedStore.set(sc.id, sc);
  await bootScenario(sc as unknown as object);
  addGeneratedOption(sc);
  selectEl.value = `generated:${sc.id}`;
}

function addGeneratedOption(scenario: Scenario): void {
  const value = `generated:${scenario.id}`;
  const existing = [...selectEl.options].find((o) => o.value === value);
  const label = `✨ ${scenario.title}`;
  if (existing) {
    existing.textContent = label;
    return;
  }
  const opt = document.createElement("option");
  opt.value = value;
  opt.textContent = label;
  selectEl.appendChild(opt);
}

function updateAssetStatus(scenario: Scenario | null): void {
  if (scenario) {
    assetStatus.textContent = assetStreamer.statusTextForScenario(scenario);
  } else {
    assetStatus.textContent = `Assets cached: ${assetStreamer.cachedCount()}/${assetStreamer.totalCount()}`;
  }
}

async function refreshScenarioList(): Promise<void> {
  try {
    const res = await fetch(API_CONFIG.scenariosUrl);
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    const data = (await res.json()) as { scenarios?: Array<{ id: string; title: string; filename: string; generated: boolean }> };
    const list = data.scenarios ?? [];
    if (list.length === 0) return;

    for (const item of list) {
      if (item.id === "money-management-town") continue;
      const value = `generated:${item.id}`;
      if ([...selectEl.options].some((o) => o.value === value)) continue;
      const opt = document.createElement("option");
      opt.value = value;
      opt.textContent = item.generated ? `✨ ${item.title}` : item.title;
      selectEl.appendChild(opt);

      try {
        const r = await fetch(`${API_CONFIG.scenariosUrl}/${encodeURIComponent(item.id)}`);
        if (r.ok) {
          const b = (await r.json()) as { scenario?: Scenario };
          if (b.scenario) generatedStore.set(value, b.scenario as Scenario);
        }
      } catch {
        // ignore per-item preload failure
      }
    }

    if (list.length > 1) {
      console.log(`[GLAM] Loaded ${list.length} scenarios from server`);
    }
  } catch (err) {
    console.warn("[GLAM] Could not fetch scenarios list, using local cache", err);
    try {
      const raw = localStorage.getItem(STORAGE_KEYS.lastGenerated);
      if (raw) {
        const parsed = JSON.parse(raw) as Scenario;
        if (parsed?.id) {
          generatedStore.set(`generated:${parsed.id}`, parsed);
          addGeneratedOption(parsed);
        }
      }
    } catch {
      // ignore
    }
  }
}

selectEl.addEventListener("change", () => {
  const val = selectEl.value;
  if (val.startsWith("generated:")) {
    const sc = generatedStore.get(val);
    if (sc) {
      void bootScenario(sc as unknown as object);
      return;
    }
    const id = val.replace(/^generated:/, "");
    void loadScenarioById(id);
    return;
  }
  currentSource = val;
  void bootScenario(currentSource);
});

restartBtn.addEventListener("click", () => {
  if (currentScenario) void bootScenario(currentScenario as unknown as object);
  else void bootScenario(currentSource);
});

declare global {
  interface Window {
    Glam: {
      loadScenario: (src: string | object) => Promise<Scenario>;
      getScenario: () => Scenario | null;
      getGame: () => Phaser.Game;
    };
  }
}

window.Glam = {
  loadScenario: async (src: string | object) => {
    await bootScenario(src);
    if (!currentScenario) throw new Error("Failed to load");
    return currentScenario;
  },
  getScenario: () => currentScenario,
  getGame: () => game,
};

try {
  const raw = localStorage.getItem(STORAGE_KEYS.lastGenerated);
  if (raw) {
    const parsed = JSON.parse(raw) as Scenario;
    if (parsed?.id) generatedStore.set(`generated:${parsed.id}`, parsed);
  }
} catch {
  // ignore
}
