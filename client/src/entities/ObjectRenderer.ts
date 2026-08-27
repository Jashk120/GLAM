import type { ObjectEntity } from "../types/scenario";
import { resolveAsset } from "../engine/AssetResolver";
import { TILE } from "../world/WorldRenderer";

export class ObjectRenderer {
  static render(scene: Phaser.Scene, obj: ObjectEntity): Phaser.GameObjects.Container {
    const resolved = resolveAsset(obj.assetId);
    const icon = resolved.icon;
    const x = obj.position.x * TILE + TILE / 2;
    const y = obj.position.y * TILE + TILE / 2;

    const bg = scene.add.rectangle(0, 0, 20, 20, 0x2a2a44, 0.85);
    bg.setStrokeStyle(1, 0xffd700, 0.6);
    bg.setOrigin(0.5);

    const emoji = scene.add.text(0, 0, icon, { fontSize: "16px" });
    emoji.setOrigin(0.5);

    const container = scene.add.container(x, y, [bg, emoji]);
    container.setDepth(y);
    container.setData("entityId", obj.id);
    container.setData("kind", "object");
    return container;
  }
}
