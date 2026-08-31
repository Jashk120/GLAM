import type { Character } from "../types/scenario";
import { resolveAsset } from "../engine/AssetResolver";
import { TILE } from "../world/renderConstants";
import {
  CHARACTER_FALLBACK,
  ENTITY_FONTS,
  ENTITY_LABEL_STYLE,
} from "./entityConstants";

export class CharacterRenderer {
  static render(scene: Phaser.Scene, char: Character): Phaser.GameObjects.Container {
    const assetId = char.appearance?.spriteId;
    const resolved = assetId ? resolveAsset(assetId) : null;
    const icon = resolved?.icon ?? CHARACTER_FALLBACK.iconFallback;
    const colorStr = char.appearance?.color ?? CHARACTER_FALLBACK.defaultColor;
    let shirtColor: number = CHARACTER_FALLBACK.fallbackShirtColor;
    try {
      shirtColor = Phaser.Display.Color.HexStringToColor(colorStr).color;
    } catch {
      shirtColor = CHARACTER_FALLBACK.fallbackShirtColor;
    }

    const cx = char.position.x * TILE + TILE / 2;
    const bottomY = (char.position.y + 1) * TILE;
    const container = scene.add.container(cx, bottomY);
    container.setDepth(bottomY);
    container.setData("entityId", char.id);
    container.setData("kind", "character");

    const g = scene.add.graphics();
    g.fillStyle(CHARACTER_FALLBACK.shadow, CHARACTER_FALLBACK.shadowOpacity);
    g.fillEllipse(0, -2, 14, 5);
    g.fillStyle(CHARACTER_FALLBACK.pantsColor, 1);
    g.fillRect(-6, -6, 5, 4);
    g.fillRect(1, -6, 5, 4);
    g.fillStyle(shirtColor, 1);
    g.fillRect(-7, -18, 14, 10);
    g.fillStyle(CHARACTER_FALLBACK.shadow, 0.08);
    g.fillRect(-7, -10, 14, 2);
    g.lineStyle(1, CHARACTER_FALLBACK.skinStroke, 0.95);
    g.strokeRect(-7, -18, 14, 10);
    g.fillStyle(CHARACTER_FALLBACK.skinColor, 1);
    g.fillRect(-6, -28, 12, 11);
    g.lineStyle(1, CHARACTER_FALLBACK.skinStroke, 0.95);
    g.strokeRect(-6, -28, 12, 11);
    const isTeacher = assetId === "character_teacher";
    const isRanger = assetId === "character_ranger";
    const capColor: number = isTeacher ? CHARACTER_FALLBACK.capTeacher : isRanger ? CHARACTER_FALLBACK.capRanger : CHARACTER_FALLBACK.capDefault;
    g.fillStyle(capColor, 1);
    g.fillRect(-6, -30, 12, 5);
    g.fillRect(-7, -28, 14, 2);
    g.fillStyle(CHARACTER_FALLBACK.eyeColor, 1);
    g.fillRect(-3, -22, 2, 2);
    g.fillRect(1, -22, 2, 2);
    g.fillStyle(CHARACTER_FALLBACK.blushColor, 0.55);
    g.fillRect(-4, -19, 3, 1);
    g.fillRect(1, -19, 3, 1);
    container.add(g);
    const badge = scene.add.text(7, -27, icon, { fontSize: ENTITY_FONTS.characterBadge });
    badge.setOrigin(0.5);

    const label = scene.add.text(0, -36, char.name, {
      fontSize: ENTITY_FONTS.characterLabel,
      color: ENTITY_LABEL_STYLE.textColor,
      backgroundColor: ENTITY_LABEL_STYLE.characterBg,
      padding: ENTITY_LABEL_STYLE.padding,
    });
    label.setOrigin(0.5);

    container.add([badge, label]);
    return container;
  }
}
