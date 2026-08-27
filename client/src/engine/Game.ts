import Phaser from "phaser";
import type { Scenario, EntityRef } from "../types/scenario";
import { WorldRenderer, TILE } from "../world/WorldRenderer";
import { CharacterRenderer } from "../entities/CharacterRenderer";
import { BuildingRenderer } from "../entities/BuildingRenderer";
import { ObjectRenderer } from "../entities/ObjectRenderer";
import { resolveAsset } from "./AssetResolver";
import { MissionManager } from "./MissionManager";
import { InteractionManager } from "../interactions/InteractionManager";

export class GameScene extends Phaser.Scene {
  private scenario!: Scenario;
  private worldRenderer!: WorldRenderer;
  private entities: EntityRef[] = [];
  private player!: Phaser.GameObjects.Container;
  private playerPos!: { x: number; y: number };
  private cursors!: Phaser.Types.Input.Keyboard.CursorKeys;
  private wasd!: Record<string, Phaser.Input.Keyboard.Key>;
  private interactKey!: Phaser.Input.Keyboard.Key;
  private missionManager!: MissionManager;
  private interactionManager!: InteractionManager;
  private stats: Record<string, number> = { coins: 40 };
  private moveCooldown = 0;
  private containers: Phaser.GameObjects.Container[] = [];
  private graphics!: Phaser.GameObjects.Graphics;
  private opts: {
    statsBar: HTMLElement; missionList: HTMLElement; prompt: HTMLElement;
    modal: HTMLElement; overlay: HTMLElement; toast: HTMLElement; titleEl: HTMLElement;
  };
  constructor(opts: GameScene["opts"]) { super("GameScene"); this.opts = opts; }
  preload(): void {}
  create(): void {
    this.cursors = this.input.keyboard!.createCursorKeys();
    this.wasd = {
      W: this.input.keyboard!.addKey(Phaser.Input.Keyboard.KeyCodes.W),
      A: this.input.keyboard!.addKey(Phaser.Input.Keyboard.KeyCodes.A),
      S: this.input.keyboard!.addKey(Phaser.Input.Keyboard.KeyCodes.S),
      D: this.input.keyboard!.addKey(Phaser.Input.Keyboard.KeyCodes.D),
    };
    this.interactKey = this.input.keyboard!.addKey(Phaser.Input.Keyboard.KeyCodes.E);
    this.graphics = this.add.graphics();
    this.player = this.add.container(0, 0);
  }
  async loadScenarioData(scenario: Scenario): Promise<void> {
    this.scenario = scenario;
    this.stats = { coins: 40, lives: 3, score: 0 };
    this.opts.titleEl.textContent = scenario.title;
    this.entities = this.buildEntityRefs(scenario);
    this.worldRenderer = new WorldRenderer(scenario.world);
    const { w, h } = this.worldRenderer.getPixelSize();
    this.scale.resize(w, h);
    this.cameras.main.setBounds(0, 0, w, h);
    for (const c of this.containers) c.destroy();
    this.containers = [];
    if (this.player) this.player.destroy();
    this.graphics.clear();
    this.worldRenderer.render(this.graphics);
    for (const e of this.entities) {
      let ctr: Phaser.GameObjects.Container | null = null;
      if (e.kind === "character") {
        const ch = scenario.characters.find((x) => x.id === e.id);
        if (ch) ctr = CharacterRenderer.render(this, ch);
      } else if (e.kind === "building") {
        const b = scenario.buildings.find((x) => x.id === e.id);
        if (b) ctr = BuildingRenderer.render(this, b);
      } else if (e.kind === "object") {
        const o = scenario.objects.find((x) => x.id === e.id);
        if (o) ctr = ObjectRenderer.render(this, o);
      }
      if (ctr) this.containers.push(ctr);
    }
    this.playerPos = { ...scenario.world.spawn };
    this.player = this.createPlayerSprite(this.playerPos.x, this.playerPos.y);
    this.renderStats();
    this.missionManager = new MissionManager(scenario.missions, this.opts.missionList, (msg, ok) => this.showToast(msg, ok), () => this.stats);
    this.interactionManager = new InteractionManager(scenario, this.entities, {
      modalEl: this.opts.modal, overlayEl: this.opts.overlay, promptEl: this.opts.prompt,
      getPlayerPos: () => this.playerPos, getStats: () => this.stats,
      updateStat: (k, d) => this.updateStat(k, d), toast: (msg, ok) => this.showToast(msg, ok),
      onMissionTrigger: (entityId) => { this.missionManager.completeByEntity(entityId); },
    });
    this.interactionManager.tick();
  }
  private buildEntityRefs(scenario: Scenario): EntityRef[] {
    const out: EntityRef[] = [];
    for (const c of scenario.characters) {
      const r = c.appearance?.spriteId ? resolveAsset(c.appearance.spriteId) : null;
      out.push({ kind: "character", id: c.id, position: c.position, name: c.name, interaction: c.interaction, solid: r ? r.solid : true, icon: r?.icon ?? "🧑", width: 1, height: 1, raw: c });
    }
    for (const b of scenario.buildings) {
      const r = resolveAsset(b.typeAssetId);
      out.push({ kind: "building", id: b.id, position: b.position, name: b.id, interaction: b.interaction, solid: r.solid, icon: r.icon, width: b.width ?? 1, height: b.height ?? 1, raw: b });
    }
    for (const o of scenario.objects) {
      const r = resolveAsset(o.assetId);
      out.push({ kind: "object", id: o.id, position: o.position, name: o.id, interaction: o.interaction, solid: r.solid, icon: r.icon, width: 1, height: 1, raw: o });
    }
    return out;
  }
  private createPlayerSprite(x: number, y: number): Phaser.GameObjects.Container {
    const px = x * TILE + TILE / 2, py = y * TILE + TILE / 2;
    const bg = this.add.rectangle(0, 0, 28, 28, 0x2d7df6, 1);
    bg.setStrokeStyle(2, 0xffffff, 1);
    const emoji = this.add.text(0, 1, "🧑", { fontSize: "18px" }); emoji.setOrigin(0.5);
    const c = this.add.container(px, py, [bg, emoji]); c.setDepth(1000); return c;
  }
  private isSolidAt(x: number, y: number): boolean {
    const { cols, rows } = this.scenario.world.size;
    if (x < 0 || y < 0 || x >= cols || y >= rows) return true;
    for (const e of this.entities) {
      if (!e.solid) continue;
      for (let dx = 0; dx < e.width; dx++) for (let dy = 0; dy < e.height; dy++) if (e.position.x + dx === x && e.position.y + dy === y) return true;
    }
    return false;
  }
  override update(time: number, _delta: number): void {
    if (!this.scenario || !this.interactionManager) return;
    if (this.interactionManager.isModalOpen()) return;
    if (Phaser.Input.Keyboard.JustDown(this.interactKey)) { this.interactionManager.handleInteractKey(); return; }
    if (time < this.moveCooldown) return;
    let dx = 0, dy = 0;
    if (Phaser.Input.Keyboard.JustDown(this.cursors.up!) || Phaser.Input.Keyboard.JustDown(this.wasd["W"])) dy = -1;
    else if (Phaser.Input.Keyboard.JustDown(this.cursors.down!) || Phaser.Input.Keyboard.JustDown(this.wasd["S"])) dy = 1;
    else if (Phaser.Input.Keyboard.JustDown(this.cursors.left!) || Phaser.Input.Keyboard.JustDown(this.wasd["A"])) dx = -1;
    else if (Phaser.Input.Keyboard.JustDown(this.cursors.right!) || Phaser.Input.Keyboard.JustDown(this.wasd["D"])) dx = 1;
    if (dx !== 0 || dy !== 0) {
      const nx = this.playerPos.x + dx, ny = this.playerPos.y + dy;
      if (!this.isSolidAt(nx, ny)) { this.playerPos.x = nx; this.playerPos.y = ny; this.player.setPosition(nx * TILE + TILE / 2, ny * TILE + TILE / 2); this.player.setDepth(1000); }
      this.moveCooldown = time + 120; this.interactionManager.tick();
    }
  }
  private renderStats(): void {
    this.opts.statsBar.innerHTML = Object.entries(this.stats).map(([k, v]) => {
      const icon = k === "coins" ? "💰" : k === "lives" ? "❤️" : "⭐";
      return `<div class="stat"><span>${icon}</span> <span id="stat_${k}">${v}</span> <span style="font-size:11px;opacity:0.7">${k}</span></div>`;
    }).join("");
  }
  private updateStat(key: string, delta: number): void {
    if (!(key in this.stats)) this.stats[key] = 0;
    this.stats[key] = (this.stats[key] ?? 0) + delta;
    const el = document.getElementById(`stat_${key}`);
    if (el) el.textContent = String(this.stats[key]); else this.renderStats();
  }
  private showToast(msg: string, ok = true): void {
    const t = this.opts.toast; t.textContent = msg; t.style.background = ok ? "#2e7d32" : "#c62828"; t.style.display = "block";
    const existing = (t as unknown as { _timer?: number })._timer;
    if (existing) window.clearTimeout(existing);
    (t as unknown as { _timer: number })._timer = window.setTimeout(() => { t.style.display = "none"; }, 2600);
  }
}
