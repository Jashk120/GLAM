import type { ObjectEntity } from "../types/scenario";
import { resolveAsset } from "../engine/AssetResolver";
import { TILE } from "../world/renderConstants";
import { ENTITY_COLORS, OBJECT_GEOMETRY, ENTITY_FONTS } from "./entityConstants";

export class ObjectRenderer {
  static render(scene: Phaser.Scene, obj: ObjectEntity): Phaser.GameObjects.Container {
    const resolved = resolveAsset(obj.assetId);
    const icon = resolved.icon;
    const cx = obj.position.x * TILE + TILE / 2;
    const bottomY = (obj.position.y + 1) * TILE;
    const g = scene.add.graphics();
    g.fillStyle(ENTITY_COLORS.shadow, ENTITY_COLORS.objectShadowOpacity);
    g.fillEllipse(0, OBJECT_GEOMETRY.shadowEllipse.y, OBJECT_GEOMETRY.shadowEllipse.w, OBJECT_GEOMETRY.shadowEllipse.h);
    g.fillStyle(ENTITY_COLORS.objectBg, ENTITY_COLORS.objectBgOpacity);
    g.fillRoundedRect(OBJECT_GEOMETRY.box.x, OBJECT_GEOMETRY.box.y, OBJECT_GEOMETRY.box.w, OBJECT_GEOMETRY.box.h, OBJECT_GEOMETRY.box.r);
    g.lineStyle(1, ENTITY_COLORS.objectBorder, ENTITY_COLORS.objectBorderOpacity);
    g.strokeRoundedRect(OBJECT_GEOMETRY.box.x, OBJECT_GEOMETRY.box.y, OBJECT_GEOMETRY.box.w, OBJECT_GEOMETRY.box.h, OBJECT_GEOMETRY.box.r);
    const emoji = scene.add.text(0, OBJECT_GEOMETRY.emojiOffsetY, icon, { fontSize: ENTITY_FONTS.objectEmoji });
    emoji.setOrigin(0.5);
    const container = scene.add.container(cx, bottomY, [g, emoji]);
    container.setDepth(bottomY);
    container.setData("entityId", obj.id);
    container.setData("kind", "object");
    return container;
  }
}
