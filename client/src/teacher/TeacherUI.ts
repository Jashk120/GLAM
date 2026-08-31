import { validateScenarioSync } from "../engine/ScenarioLoader";
import { assetStreamer } from "../assets/AssetStreamer";
import type { Scenario } from "../types/scenario";
import { STORAGE_KEYS } from "../assets/storageKeys";
import { STATUS_CLEAR_DELAY_MS } from "../engine/timingConstants";
import { API_CONFIG, serverOriginFromConfig } from "../config/env";

type GenerateSuccess = (scenario: Scenario) => Promise<void> | void;
type ToastFn = (msg: string, ok?: boolean) => void;

interface TeacherUIOptions {
  promptInput: HTMLTextAreaElement;
  generateBtn: HTMLButtonElement;
  genStatus: HTMLElement;
  genError: HTMLElement;
  assetStatus: HTMLElement;
  scenarioSelect: HTMLSelectElement;
  toast: ToastFn;
  onGenerated: GenerateSuccess;
  getCurrentScenario: () => Scenario | null;
}

const STORAGE_KEY = STORAGE_KEYS.lastGenerated;
const GENERATE_URL = API_CONFIG.generateUrl;

function formatDetails(details: unknown): string {
  if (Array.isArray(details)) return details.map((d) => `• ${String(d)}`).join("\n");
  if (typeof details === "string") return details;
  if (details) return String(details);
  return "";
}

export class TeacherUI {
  private opts: TeacherUIOptions;
  private generating = false;

  constructor(opts: TeacherUIOptions) {
    this.opts = opts;
    this.bindEvents();
    this.restoreLastGenerated();
    this.updateAssetStatus(null);
    assetStreamer.onStatus((msg) => {
      this.opts.assetStatus.textContent = msg;
    });
  }

  private bindEvents(): void {
    this.opts.generateBtn.addEventListener("click", () => {
      void this.handleGenerate();
    });
    const stopGameKeys = (e: KeyboardEvent) => {
      e.stopPropagation();
      const navKeys = ["ArrowUp", "ArrowDown", "ArrowLeft", "ArrowRight", "w", "a", "s", "d", "W", "A", "S", "D", "e", "E"];
      if (navKeys.includes(e.key) || e.key === " " || e.key === "Escape") {
        e.stopImmediatePropagation?.();
      }
    };
    this.opts.promptInput.addEventListener("keydown", (e) => {
      stopGameKeys(e);
      if ((e.ctrlKey || e.metaKey) && e.key === "Enter") {
        e.preventDefault();
        void this.handleGenerate();
      }
    });
    this.opts.promptInput.addEventListener("focus", () => {
      document.body.classList.add("teacher-typing");
    });
    this.opts.promptInput.addEventListener("blur", () => {
      document.body.classList.remove("teacher-typing");
    });
  }

  private setLoading(loading: boolean, text = ""): void {
    this.generating = loading;
    this.opts.generateBtn.disabled = loading;
    this.opts.generateBtn.textContent = loading ? "⏳ Generating..." : "✨ Generate Scenario";
    this.opts.genStatus.textContent = text;
    this.opts.genStatus.classList.toggle("visible", loading || text.length > 0);
    if (loading) {
      this.opts.genStatus.classList.add("loading");
      this.opts.genError.textContent = "";
      this.opts.genError.classList.remove("visible");
    } else {
      this.opts.genStatus.classList.remove("loading");
    }
  }

  private showError(message: string, details?: unknown): void {
    const detailText = details ? formatDetails(details) : "";
    const full = detailText ? `${message}\n${detailText}` : message;
    this.opts.genError.textContent = full;
    this.opts.genError.classList.add("visible");
    this.opts.genStatus.textContent = "";
    this.opts.genStatus.classList.remove("visible");
  }

  private clearError(): void {
    this.opts.genError.textContent = "";
    this.opts.genError.classList.remove("visible");
  }

  updateAssetStatus(scenario: Scenario | null): void {
    if (scenario) {
      this.opts.assetStatus.textContent = assetStreamer.statusTextForScenario(scenario);
    } else {
      this.opts.assetStatus.textContent = `Assets cached: ${assetStreamer.cachedCount()}/${assetStreamer.totalCount()}`;
    }
  }

  private addGeneratedOption(scenario: Scenario): void {
    const value = `generated:${scenario.id}`;
    const existing = [...this.opts.scenarioSelect.options].find((o) => o.value === value);
    const label = `✨ Generated: ${scenario.title}`;
    if (existing) {
      existing.textContent = label;
      this.opts.scenarioSelect.value = value;
      return;
    }
    const opt = document.createElement("option");
    opt.value = value;
    opt.textContent = label;
    this.opts.scenarioSelect.appendChild(opt);
    this.opts.scenarioSelect.value = value;
  }

  private restoreLastGenerated(): void {
    try {
      const raw = localStorage.getItem(STORAGE_KEY);
      if (!raw) return;
      const parsed = JSON.parse(raw) as Scenario;
      if (parsed?.id && parsed?.title) {
        this.addGeneratedOption(parsed);
        console.log("[TeacherUI] Restored lastGenerated:", parsed.id);
        // Do not auto-load; keep example as default. User can play via selector.
      }
    } catch {
      // ignore
    }
  }

  async handleGenerate(): Promise<void> {
    if (this.generating) return;
    const prompt = this.opts.promptInput.value.trim();
    if (!prompt) {
      this.showError("Please enter a prompt describing your lesson.");
      return;
    }
    this.clearError();
    this.setLoading(true, "Generating scenario...");

    console.log("[TeacherUI] Generate requested:", prompt.slice(0, 80));

    try {
      const res = await fetch(GENERATE_URL, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ prompt }),
      });

      const body = (await res.json()) as {
        scenario?: unknown;
        error?: string;
        details?: unknown;
      };

      if (!res.ok) {
        const msg = body.error ?? `Request failed (${res.status})`;
        const details = body.details;
        this.setLoading(false);
        this.showError(msg, details);
        console.warn("[TeacherUI] Generate failed:", msg, details);
        return;
      }

      if (!body.scenario) {
        this.setLoading(false);
        this.showError("Server returned no scenario.");
        return;
      }

      let validated: Scenario;
      try {
        validated = validateScenarioSync(body.scenario);
      } catch (e) {
        const msg = e instanceof Error ? e.message : String(e);
        this.setLoading(false);
        this.showError("Client validation failed:", msg);
        return;
      }

      this.setLoading(true, "Caching assets...");
      try {
        // Trigger asset preload before render (also done in loadScenario, but show status)
        await assetStreamer.preloadScenarioAssets(validated);
      } catch {
        // non-fatal
      }
      this.updateAssetStatus(validated);

      // Persist
      try {
        localStorage.setItem(STORAGE_KEY, JSON.stringify(validated));
      } catch {
        // ignore quota
      }

      // Update selector and render instantly without reload
      this.addGeneratedOption(validated);
      await this.opts.onGenerated(validated);

      this.setLoading(false);
      this.opts.genStatus.textContent = "Generated! Loaded into game.";
      this.opts.genStatus.classList.add("visible");
      window.setTimeout(() => {
        this.opts.genStatus.textContent = "";
        this.opts.genStatus.classList.remove("visible");
      }, STATUS_CLEAR_DELAY_MS);
      this.opts.toast("Generated! 🎉", true);
      console.log("[TeacherUI] Play Generated:", validated.id, validated.title);
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err);
      this.setLoading(false);
      if (msg.includes("Failed to fetch") || msg.includes("NetworkError")) {
        this.showError(`Network error — is the Go server running on ${serverOriginFromConfig()}?`);
      } else {
        this.showError("Network or server error:", msg);
      }
      console.error("[TeacherUI] Generate exception:", err);
    }
  }

  getLastGenerated(): Scenario | null {
    try {
      const raw = localStorage.getItem(STORAGE_KEY);
      if (!raw) return null;
      return JSON.parse(raw) as Scenario;
    } catch {
      return null;
    }
  }
}
