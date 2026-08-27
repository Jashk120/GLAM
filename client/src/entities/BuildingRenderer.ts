import type { Building } from "../types/scenario";
import { resolveAsset } from "../engine/AssetResolver";
import { TILE } from "../world/WorldRenderer";

export class BuildingRenderer {
  static render(scene: Phaser.Scene, b: Building): Phaser.GameObjects.Container {
    const resolved = resolveAsset(b.typeAssetId);
    const icon = resolved.icon;
    const w = (b.width ?? 1) * TILE;
    const h = (b.height ?? 1) * TILE;
    const x = b.position.x * TILE + w / 2;
    const y = b.position.y * TILE + h / 2;

    const rect = scene.add.rectangle(0, 0, w - 2, h - 2, 0x4a3a2a, 0.95);
    rect.setStrokeStyle(2, 0x7a6a4a, 1);

    // inner highlight
    const inner = scene.add.rectangle(0, 0, w - 6, h - 6, 0x6b5a3a, 0.5);
    inner.setStrokeStyle(1, 0xffe4a3, 0.6);

    const emoji = scene.add.text(0, 0, icon, { fontSize: `${Math.min(22, (w / 2))}px` });
    emoji.setOrigin(0.5);

    const label = scene.add.text(0, -h / 2 - 8, b.id, {
      fontSize: "8px",
      color: "#ffffff",
      backgroundColor: "#00000099",
      padding: { left: 2, right: 2, top: 1, bottom: 1 },
    });
    label.setOrigin(0.5);

    const container = scene.add.container(x, y, [rect, inner, emoji, label]);
    container.setDepth(y);
    container.setData("entityId", b.id);
    container.setData("kind", "building");
    return container;
  }
}
