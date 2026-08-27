import "./style.css";
import Phaser from "phaser";
import { GameScene } from "./engine/Game";
import { loadScenario } from "./engine/ScenarioLoader";
import { TeacherUI } from "./teacher/TeacherUI";
import { assetStreamer } from "./assets/AssetStreamer";
import type { Scenario } from "./types/scenario";

const statsBar = document.getElementById("statsBar") as HTMLElement;
const missionList = document.getElementById("missionList") as HTMLElement;
const promptEl = document.getElementById("prompt") as HTMLElement;
const modalEl = document.getElementById("modal") as HTMLElement;
const overlayEl = document.getElementById("modalOverlay") as HTMLElement;
const toastEl = document.getElementById("toast") as HTMLElement;
const titleEl = document.getElementById("scenarioTitle") as HTMLElement;
const selectEl = document.getElementById("scenarioSelect") as HTMLSelectElement;
const restartBtn = document.getElementById("restartBtn") as HTMLButtonElement;

let gameScene: GameScene | null = null;
let currentScenario: Scenario | null = null;
let currentSource = "/scenarios/example.json";

// in-memory store for generated scenarios keyed by select value
const generatedStore = new Map<string, Scenario>();

function showToast(msg: string, ok = true): void {
  toastEl.textContent = msg;
  toastEl.style.background = ok ? "#2e7d32" : "#c62828";
  toastEl.style.display = "block";
  const existing = (toastEl as unknown as { _timer?: number })._timer;
  if (existing) window.clearTimeout(existing);
  (toastEl as unknown as { _timer: number })._timer = window.setTimeout(() => {
    toastEl.style.display = "none";
  }, 2600);
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
  width: 15 * 32,
  height: 12 * 32,
  scene: [scene],
  scale: {
    mode: Phaser.Scale.NONE,
    autoCenter: Phaser.Scale.CENTER_BOTH,
  },
});

game.events.on("ready", async () => {
  gameScene = scene;
  await bootScenario(currentSource);
  // Initialize asset status after first load
  teacherUI.updateAssetStatus(currentScenario);
  console.log("[GLAM] Load Example: booted", currentSource);
});

overlayEl.addEventListener("click", (e) => {
  if (e.target === overlayEl) overlayEl.classList.remove("active");
});

async function bootScenario(source: string | object): Promise<void> {
  try {
    const sc = await loadScenario(source);
    currentScenario = sc;
    if (!gameScene) return;
    const { w, h } = { w: sc.world.size.cols * 32, h: sc.world.size.rows * 32 };
    game.scale.resize(w, h);
    await gameScene.loadScenarioData(sc);
    titleEl.textContent = sc.title;
    teacherUI.updateAssetStatus(sc);
  } catch (err) {
    const msg = err instanceof Error ? err.message : String(err);
    showToast(`❌ Failed to load scenario: ${msg}`, false);
    console.error(err);
  }
}

selectEl.addEventListener("change", () => {
  const val = selectEl.value;
  if (val.startsWith("generated:")) {
    const sc = generatedStore.get(val);
    if (sc) {
      console.log("[GLAM] Play Generated:", sc.id);
      void bootScenario(sc as unknown as object);
      return;
    }
    // Fallback: try localStorage
    try {
      const raw = localStorage.getItem("glam_lastGenerated");
      if (raw) {
        const parsed = JSON.parse(raw) as Scenario;
        if (`generated:${parsed.id}` === val) {
          generatedStore.set(val, parsed);
          void bootScenario(parsed as unknown as object);
          return;
        }
      }
    } catch {
      // ignore
    }
    showToast("Generated scenario not found in memory. Please generate again.", false);
    return;
  }
  currentSource = val;
  void bootScenario(currentSource);
});

restartBtn.addEventListener("click", () => {
  if (currentScenario) void bootScenario(currentScenario as unknown as object);
  else void bootScenario(currentSource);
});

// Expose for dev
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

// Teacher UI wiring
const promptInput = document.getElementById("promptInput") as HTMLTextAreaElement;
const generateBtn = document.getElementById("generateBtn") as HTMLButtonElement;
const genStatus = document.getElementById("genStatus") as HTMLElement;
const genError = document.getElementById("genError") as HTMLElement;
const assetStatus = document.getElementById("assetStatus") as HTMLElement;

const teacherUI = new TeacherUI({
  promptInput,
  generateBtn,
  genStatus,
  genError,
  assetStatus,
  scenarioSelect: selectEl,
  toast: showToast,
  getCurrentScenario: () => currentScenario,
  onGenerated: async (scenario: Scenario) => {
    const key = `generated:${scenario.id}`;
    generatedStore.set(key, scenario);
    console.log("[GLAM] Teacher -> Go -> Phaser flow: loadScenario", scenario.id);
    await bootScenario(scenario as unknown as object);
  },
});

// Initial asset status before scenario loads
assetStatus.textContent = `Assets cached: ${assetStreamer.cachedCount()}/${assetStreamer.totalCount()}`;

// Restore generated from localStorage into store + selector on startup
try {
  const raw = localStorage.getItem("glam_lastGenerated");
  if (raw) {
    const parsed = JSON.parse(raw) as Scenario;
    if (parsed?.id) generatedStore.set(`generated:${parsed.id}`, parsed);
  }
} catch {
  // ignore
}
