import type { Mission, RequiredStat } from "../types/scenario";

function evaluateRequiredStat(cond: RequiredStat, stats: Record<string, number>): boolean {
  const value = stats[cond.stat] ?? 0;
  const target = cond.target;
  switch (cond.operator) {
    case ">=":
      return value >= target;
    case ">":
      return value > target;
    case "<=":
      return value <= target;
    case "<":
      return value < target;
    case "=":
    case "==":
      return value === target;
    case "!=":
      return value !== target;
    default:
      return false;
  }
}

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
        if (m.checkAtEnd) {
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
        if (m.requiredStat) {
          if (evaluateRequiredStat(m.requiredStat, stats)) m.done = true;
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
  return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;").replace(/'/g, "&#39;").replace(/`/g, "&#96;");
}
