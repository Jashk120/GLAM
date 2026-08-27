import type { Mission } from "../types/scenario";

export class MissionManager {
  private missions: Mission[];
  private listEl: HTMLElement;
  private toastFn: (msg: string, ok?: boolean) => void;
  private statsGetter: () => Record<string, number>;

  constructor(
    missions: Mission[],
    listEl: HTMLElement,
    toastFn: (msg: string, ok?: boolean) => void,
    statsGetter: () => Record<string, number>,
  ) {
    this.missions = missions.map((m) => ({ ...m, done: m.done ?? false }));
    this.listEl = listEl;
    this.toastFn = toastFn;
    this.statsGetter = statsGetter;
    this.render();
  }

  setMissions(missions: Mission[]): void {
    this.missions = missions.map((m) => ({ ...m, done: m.done ?? false }));
    this.render();
  }

  getMissions(): Mission[] {
    return this.missions;
  }

  render(): void {
    this.listEl.innerHTML = this.missions
      .map(
        (m) => `
      <div class="mission ${m.done ? "done" : ""}">
        <div class="title">${m.done ? "✅" : "⬜"} ${escapeHtml(m.title)}</div>
        <div>${escapeHtml(m.description)}</div>
      </div>
    `,
      )
      .join("");
  }

  completeMission(id: string): void {
    const m = this.missions.find((x) => x.id === id);
    if (m && !m.done) {
      m.done = true;
      this.render();
      this.checkAllDone();
    }
  }

  completeByEntity(entityId: string): void {
    for (const m of this.missions) {
      if (!m.done && m.trigger?.entityId === entityId) {
        // checkAtEnd missions require extra validation — handled in checkAllDone or explicit call
        if (m.checkAtEnd) {
          // For save_coins type: coins >=10 check will be done in checkAllDone; allow manual completion if stats pass
          // We complete non-checkAtEnd immediately, checkAtEnd via checkEndMissions
          continue;
        }
        m.done = true;
      }
    }
    this.render();
    this.checkAllDone();
  }

  checkEndMissions(): void {
    const stats = this.statsGetter();
    for (const m of this.missions) {
      if (m.done) continue;
      if (m.checkAtEnd) {
        // Generic: if mission id save_coins, require coins >=10 else consider done when called
        if (m.id === "save_coins") {
          if ((stats["coins"] ?? 0) >= 10) m.done = true;
        } else {
          m.done = true;
        }
      }
    }
    this.render();
    this.checkAllDone(false);
  }

  private checkAllDone(tryEndCheck = true): void {
    if (tryEndCheck) {
      const pendingNonEnd = this.missions.filter((m) => !m.checkAtEnd && !m.done);
      if (pendingNonEnd.length === 0) {
        const hasEnd = this.missions.some((m) => m.checkAtEnd);
        if (hasEnd) this.checkEndMissions();
        else if (this.missions.length > 0 && this.missions.every((m) => m.done)) {
          this.toastFn("🎉 All missions complete! Great job!", true);
        }
        return;
      }
    }
    if (this.missions.length > 0 && this.missions.every((m) => m.done)) {
      this.toastFn("🎉 Scenario complete! Great job!", true);
    }
  }
}

function escapeHtml(s: string): string {
  return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
}
