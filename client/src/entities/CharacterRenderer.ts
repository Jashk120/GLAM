import type { Character } from "../types/scenario";
import { resolveAsset } from "../engine/AssetResolver";
import { TILE } from "../world/WorldRenderer";

export class CharacterRenderer {
  static render(scene: Phaser.Scene, char: Character): Phaser.GameObjects.Container {
    const assetId = char.appearance?.spriteId;
    const resolved = assetId ? resolveAsset(assetId) : null;
    const icon = resolved?.icon ?? "🧑";
    const x = char.position.x * TILE + TILE / 2;
    const y = char.position.y * TILE + TILE / 2;

    const bg = scene.add.rectangle(0, 0, 28, 28, Phaser.Display.Color.HexStringToColor(char.appearance?.color ?? "#3a6ea5").color, 0.92);
    bg.setStrokeStyle(2, 0xffffff, 0.9);
    bg.setOrigin(0.5);

    const label = scene.add.text(0, -18, char.name, {
      fontSize: "9px",
      color: "#ffffff",
      backgroundColor: "#000000aa",
      padding: { left: 2, right: 2, top: 1, bottom: 1 },
    });
    label.setOrigin(0.5);

    const emoji = scene.add.text(0, 2, icon, {
      fontSize: "18px",
    });
    emoji.setOrigin(0.5);

    const container = scene.add.container(x, y, [bg, emoji, label]);
    container.setDepth(y);
    // store id for lookup
    container.setData("entityId", char.id);
    container.setData("kind", "character");
    return container;
  }
}
