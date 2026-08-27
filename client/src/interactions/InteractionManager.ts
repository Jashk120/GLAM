import type { Scenario, EntityRef, Interaction, Outcome } from "../types/scenario";
import { showDialogue } from "../activities/DialogueActivity";
import { showMCQ } from "../activities/MCQActivity";
import { showMath } from "../activities/MathActivity";
import { showShop } from "../activities/ShopActivity";
import { showInformation } from "../activities/InformationActivity";

export interface InteractionDeps {
  modalEl: HTMLElement;
  overlayEl: HTMLElement;
  promptEl: HTMLElement;
  getPlayerPos: () => { x: number; y: number };
  getStats: () => Record<string, number>;
  updateStat: (key: string, delta: number) => void;
  toast: (msg: string, ok?: boolean) => void;
  onMissionTrigger: (entityId: string, interaction?: Interaction) => void;
}

export class InteractionManager {
  private scenario: Scenario;
  private entities: EntityRef[];
  private deps: InteractionDeps;
  private cooldowns = new Map<string, number>();
  private modalOpen = false;

  constructor(scenario: Scenario, entities: EntityRef[], deps: InteractionDeps) {
    this.scenario = scenario;
    this.entities = entities;
    this.deps = deps;
    // close modal on overlay click? keep only via buttons/ESC; allow Esc
    document.addEventListener("keydown", (e) => {
      const active = document.activeElement as HTMLElement | null;
      const typing = !!active && (active.tagName === "INPUT" || active.tagName === "TEXTAREA" || active.tagName === "SELECT" || active.isContentEditable);
      if (e.key === "Escape" && this.modalOpen && !typing) {
        this.deps.overlayEl.classList.remove("active");
        this.modalOpen = false;
      }
    });
  }

  updateEntities(scenario: Scenario, entities: EntityRef[]): void {
    this.scenario = scenario;
    this.entities = entities;
    this.cooldowns.clear();
  }

  isModalOpen(): boolean {
    return this.modalOpen;
  }

  // call each frame after movement
  tick(): void {
    if (this.modalOpen) return;
    this.updatePrompt();
    this.checkAutoTriggers();
  }

  updatePrompt(): void {
    const e = this.findNearby();
    const promptEl = this.deps.promptEl;
    if (!e || !e.interaction) {
      promptEl.classList.remove("visible");
      promptEl.textContent = "";
      return;
    }
    if (e.interaction.auto) {
      promptEl.classList.remove("visible");
      promptEl.textContent = "";
      return;
    }
    const name = e.name ?? e.id;
    promptEl.textContent = `Press E: ${name}`;
    promptEl.classList.add("visible");
  }

  handleInteractKey(): void {
    if (this.modalOpen) return;
    const e = this.findNearby();
    if (!e || !e.interaction) return;
    if (e.interaction.auto) return;
    if (this.isOnCooldown(e.id)) return;
    this.open(e);
  }

  private checkAutoTriggers(): void {
    const e = this.findNearby();
    if (!e || !e.interaction || !e.interaction.auto) return;
    if (this.isOnCooldown(e.id)) return;
    this.open(e);
  }

  private findNearby(): EntityRef | undefined {
    const p = this.deps.getPlayerPos();
    return this.entities.find((e) => manhattan(e.position, p) <= 1);
  }

  private isOnCooldown(entityId: string): boolean {
    const until = this.cooldowns.get(entityId) ?? 0;
    return Date.now() < until;
  }

  private setCooldown(entityId: string, ms: number): void {
    if (ms > 0) this.cooldowns.set(entityId, Date.now() + ms);
  }

  private applyOutcome(outcome: Outcome | undefined): void {
    if (!outcome) return;
    if (outcome.stat && typeof outcome.delta === "number") {
      this.deps.updateStat(outcome.stat, outcome.delta);
    }
    if (outcome.toast) this.deps.toast(outcome.toast);
  }

  private onInteractionComplete(entity: EntityRef, success: boolean): void {
    // cooldown
    const cd = entity.interaction?.cooldown ?? 0;
    this.setCooldown(entity.id, cd);
    // mission linkage
    this.deps.onMissionTrigger(entity.id, entity.interaction);
    // Apply onCorrect/onWrong when provided by MCQ/math handlers separately; for dialogue/info/shop also handle
    void success;
  }

  open(entity: EntityRef): void {
    const it = entity.interaction;
    if (!it) return;
    this.modalOpen = true;

    const wrapClose = (cb: () => void) => {
      cb();
      this.modalOpen = false;
      this.updatePrompt();
    };

    const name = entity.name ?? entity.id;

    switch (it.type) {
      case "dialogue": {
        showDialogue(this.deps.modalEl, this.deps.overlayEl, name, it, () => {
          wrapClose(() => {
            const cd = it.cooldown ?? 0;
            this.setCooldown(entity.id, cd);
            this.deps.onMissionTrigger(entity.id, it);
            if (it.onCorrect?.toast) this.deps.toast(it.onCorrect.toast);
          });
        });
        // intercept overlay click to close
        this.deps.overlayEl.onclick = (ev) => {
          if (ev.target === this.deps.overlayEl) {
            this.deps.overlayEl.classList.remove("active");
            this.modalOpen = false;
            this.setCooldown(entity.id, it.cooldown ?? 0);
            this.deps.onMissionTrigger(entity.id, it);
          }
        };
        break;
      }
      case "mcq": {
        showMCQ(this.deps.modalEl, this.deps.overlayEl, name, it, (correct) => {
          wrapClose(() => {
            this.setCooldown(entity.id, it.cooldown ?? 0);
            if (correct) {
              this.applyOutcome(it.onCorrect);
              if (!it.onCorrect?.toast) this.deps.toast("✅ Correct!", true);
              this.deps.onMissionTrigger(entity.id, it);
            } else {
              this.applyOutcome(it.onWrong);
              // allowRetry keeps mission not completed; if onWrong no toast handled
            }
            // Mission only on correct for MCQ by spec (pass_quiz). But generic entity linkage completes only on correct.
            // InteractionManager caller expects mission on correct; we gate:
            if (correct) this.deps.onMissionTrigger(entity.id, it);
          });
        });
        break;
      }
      case "math": {
        showMath(this.deps.modalEl, this.deps.overlayEl, name, it, (correct) => {
          wrapClose(() => {
            this.setCooldown(entity.id, it.cooldown ?? 0);
            if (correct) {
              this.applyOutcome(it.onCorrect);
              if (!it.onCorrect?.toast) this.deps.toast("✅ Correct!", true);
              this.deps.onMissionTrigger(entity.id, it);
            } else {
              this.applyOutcome(it.onWrong);
              if (!it.onWrong?.toast) this.deps.toast("❌ Try again.", false);
            }
          });
        });
        break;
      }
      case "shop": {
        showShop(
          this.deps.modalEl,
          this.deps.overlayEl,
          name,
          it,
          () => this.deps.getStats()[it.currency ?? "coins"] ?? 0,
          (idx) => {
            const item = it.items[idx];
            if (!item) return { success: false, message: "Invalid item" };
            const cur = it.currency ?? "coins";
            const coins = this.deps.getStats()[cur] ?? 0;
            if (coins < item.price) {
              this.deps.toast("❌ Not enough coins!", false);
              return { success: false, message: "Not enough coins" };
            }
            this.deps.updateStat(cur, -item.price);
            this.deps.toast(`Bought ${item.name}!`, true);
            // mission: buy triggers on first purchase
            this.deps.onMissionTrigger(entity.id, it);
            // shop doesn't close on purchase, stays open
            return { success: true, message: `Bought ${item.name}` };
          },
          () => {
            wrapClose(() => {
              this.setCooldown(entity.id, it.cooldown ?? 0);
            });
          },
        );
        // modalOpen stays true until Leave clicked; wrapClose handled there
        break;
      }
      case "information": {
        showInformation(this.deps.modalEl, this.deps.overlayEl, it, () => {
          wrapClose(() => {
            this.setCooldown(entity.id, it.cooldown ?? 0);
            this.deps.onMissionTrigger(entity.id, it);
            if (it.onCorrect?.toast) this.deps.toast(it.onCorrect.toast);
          });
        });
        break;
      }
    }
  }
}

function manhattan(a: { x: number; y: number }, b: { x: number; y: number }): number {
  return Math.abs(a.x - b.x) + Math.abs(a.y - b.y);
}
