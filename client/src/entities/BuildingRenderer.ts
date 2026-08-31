import type { Building } from "../types/scenario";
import { resolveAsset } from "../engine/AssetResolver";
import { TILE } from "../world/renderConstants";
import {
  BUILDING_STYLES,
  BUILDING_FALLBACK_STYLE,
  BUILDING_ROOF_HEIGHT_GABLED,
  BUILDING_ROOF_HEIGHT_FLAT,
  BUILDING_GEOMETRY,
  ENTITY_COLORS,
  ENTITY_FONTS,
  ENTITY_LABEL_STYLE,
  ENTITY_MIN_DIMENSION,
} from "./entityConstants";

export class BuildingRenderer {
  static render(scene: Phaser.Scene, b: Building): Phaser.GameObjects.Container {
    const resolved = resolveAsset(b.typeAssetId);
    const icon = resolved.icon;
    const w = (b.width ?? 1) * TILE;
    const h = (b.height ?? 1) * TILE;
    const bw = Math.max(ENTITY_MIN_DIMENSION, w);
    const bh = Math.max(ENTITY_MIN_DIMENSION, h);
    const cx = b.position.x * TILE + w / 2;
    const bottomY = (b.position.y + (b.height ?? 1)) * TILE;
    const s = BUILDING_STYLES[b.typeAssetId] ?? BUILDING_FALLBACK_STYLE;
    const container = scene.add.container(cx, bottomY);
    container.setDepth(bottomY);
    container.setData("entityId", b.id);
    container.setData("kind", "building");
    const g = scene.add.graphics();
    g.fillStyle(ENTITY_COLORS.shadow, ENTITY_COLORS.buildingShadowOpacity);
    g.fillEllipse(0, BUILDING_GEOMETRY.shadowEllipse.y, bw * BUILDING_GEOMETRY.shadowEllipse.wFactor, BUILDING_GEOMETRY.shadowEllipse.h);
    const left = -bw / 2 + 1;
    const right = bw / 2 - 1;
    const top = -bh;
    const bottom = 0;
    g.fillStyle(s.wall, 1);
    g.fillRect(left, top, bw - 2, bh);
    g.fillStyle(s.wallShade, 1);
    g.fillRect(right - 4, top, 4, bh);
    g.fillRect(left, bottom - 4, bw - 2, 4);
    g.lineStyle(1, ENTITY_COLORS.wallBorder, 0.9);
    g.strokeRect(left, top, bw - 2, bh);
    if (s.roofKind === "gabled") {
      const over = BUILDING_GEOMETRY.overhangGabled;
      const rh = BUILDING_ROOF_HEIGHT_GABLED;
      const rx1 = -bw / 2 - over;
      const rx2 = bw / 2 + over;
      const ryEave = top;
      const ryPeak = top - rh;
      g.fillStyle(s.roof, 1);
      g.beginPath();
      g.moveTo(rx1, ryEave);
      g.lineTo(0, ryPeak);
      g.lineTo(rx2, ryEave);
      g.lineTo(rx2, ryEave + 4);
      g.lineTo(0, ryPeak + 4);
      g.lineTo(rx1, ryEave + 4);
      g.closePath();
      g.fillPath();
      g.lineStyle(1, s.roofAccent, 1);
      g.beginPath();
      g.moveTo(rx1, ryEave);
      g.lineTo(0, ryPeak);
      g.lineTo(rx2, ryEave);
      g.strokePath();
      g.lineStyle(1, ENTITY_COLORS.windowHighlight, BUILDING_GEOMETRY.highlightOpacity);
      g.beginPath();
      g.moveTo(0, ryPeak + 2);
      g.lineTo(0, ryEave + 2);
      g.strokePath();
    } else {
      const over = BUILDING_GEOMETRY.overhangFlat;
      const rh = BUILDING_ROOF_HEIGHT_FLAT;
      const rx1 = -bw / 2 - over;
      const ryTop = top - rh;
      g.fillStyle(s.roof, 1);
      g.fillRect(rx1, ryTop, bw + over * 2, rh);
      g.fillStyle(s.roofAccent, 1);
      g.fillRect(rx1, ryTop, bw + over * 2, 3);
      g.fillRect(rx1, ryTop + rh - 2, bw + over * 2, 2);
      g.lineStyle(1, ENTITY_COLORS.wallBorder, 0.85);
      g.strokeRect(rx1, ryTop, bw + over * 2, rh);
      if (s.cross) {
        const cy0 = ryTop + rh / 2;
        g.fillStyle(0xffffff, 1);
        g.fillRect(-8, cy0 - 3, 16, 6);
        g.fillRect(-3, cy0 - 8, 6, 16);
        g.lineStyle(1, 0xd32f2f, 0.9);
        g.strokeRect(-8, cy0 - 3, 16, 6);
        g.strokeRect(-3, cy0 - 8, 6, 16);
      }
    }
    if (s.columns && bw >= ENTITY_MIN_DIMENSION) {
      g.fillStyle(0xf0f0f0, 1);
      const colW = 5;
      const colH = bh * 0.62;
      const colY = bottom - colH - 2;
      const gap = bw - 16;
      const n = bw >= 64 ? 4 : 2;
      for (let i = 0; i < n; i++) {
        const cx0 = -bw / 2 + 8 + (gap * i) / Math.max(1, n - 1);
        g.fillRect(cx0 - colW / 2, colY, colW, colH);
        g.lineStyle(1, 0x8a8a9a, 0.9);
        g.strokeRect(cx0 - colW / 2, colY, colW, colH);
      }
    }
    if (s.awning) {
      const awY = top + bh * 0.42;
      const awH = 7;
      g.fillStyle(ENTITY_COLORS.awningFill, 1);
      g.fillRect(left - 1, awY, bw, awH);
      for (let sx = left; sx < right; sx += 16) {
        g.fillStyle(s.roof, 1);
        g.fillRect(sx, awY, 8, awH);
      }
      g.lineStyle(1, ENTITY_COLORS.wallBorder, 0.8);
      g.strokeRect(left - 1, awY, bw, awH);
    }
    const winY = top + bh * 0.28;
    const winSize = BUILDING_GEOMETRY.windowSize;
    if (!s.columns) {
      if (bw >= 64) {
        const wx1 = -bw / 2 + 10;
        const wx2 = bw / 2 - 16;
        for (const wx of [wx1, wx2]) {
          g.fillStyle(ENTITY_COLORS.windowFill, 1);
          g.fillRect(wx, winY, winSize, winSize);
          g.fillStyle(ENTITY_COLORS.windowHighlight, 0.75);
          g.fillRect(wx + 2, winY + 2, 4, 3);
          g.lineStyle(1, ENTITY_COLORS.wallBorder, 0.9);
          g.strokeRect(wx, winY, winSize, winSize);
          g.lineBetween(wx + 5, winY, wx + 5, winY + winSize);
          g.lineBetween(wx, winY + 5, wx + winSize, winY + 5);
        }
      } else if (bw >= 30 && bh >= 28 && (bh > 36 || !s.awning)) {
        const y0 = s.awning ? top + 8 : winY;
        g.fillStyle(ENTITY_COLORS.windowFill, 1);
        g.fillRect(-5, y0, winSize, winSize);
        g.lineStyle(1, ENTITY_COLORS.wallBorder, 0.9);
        g.strokeRect(-5, y0, winSize, winSize);
      }
    } else {
      g.fillStyle(ENTITY_COLORS.windowFill, 1);
      g.fillRect(-5, top + 10, winSize, winSize);
      g.lineStyle(1, ENTITY_COLORS.wallBorder, 0.9);
      g.strokeRect(-5, top + 10, winSize, winSize);
    }
    const doorW = bw >= 64 ? BUILDING_GEOMETRY.doorSize.wLarge : BUILDING_GEOMETRY.doorSize.wSmall;
    const doorH = BUILDING_GEOMETRY.doorSize.h;
    const doorX = -doorW / 2;
    const doorY = bottom - doorH;
    g.fillStyle(s.door, 1);
    g.fillRect(doorX, doorY, doorW, doorH);
    g.fillStyle(ENTITY_COLORS.doorHighlight, 0.35);
    g.fillRect(doorX + 2, doorY + 3, doorW - 4, doorH - 3);
    g.fillStyle(ENTITY_COLORS.windowHighlight, 0.18);
    g.fillRect(doorX + 2, doorY + 2, doorW - 4, 3);
    g.lineStyle(1, ENTITY_COLORS.doorHighlight, 1);
    g.strokeRect(doorX, doorY, doorW, doorH);
    g.fillStyle(ENTITY_COLORS.doorKnob, 1);
    g.fillCircle(doorX + doorW - 3, doorY + doorH / 2, 1.6);
    const signW = Math.min(bw - 8, BUILDING_GEOMETRY.sign.w);
    const signH = BUILDING_GEOMETRY.sign.h;
    const signY = doorY - BUILDING_GEOMETRY.sign.offsetY;
    g.fillStyle(ENTITY_COLORS.wallBorder, 1);
    g.fillRect(-signW / 2, signY, signW, signH);
    g.fillStyle(0xf5e6c8, 1);
    g.fillRect(-signW / 2 + 1, signY + 1, signW - 2, signH - 2);
    container.add(g);
    const emoji = scene.add.text(0, signY + signH / 2, icon, { fontSize: ENTITY_FONTS.buildingEmoji });
    emoji.setOrigin(0.5);
    const label = scene.add.text(0, top - (s.roofKind === "gabled" ? BUILDING_ROOF_HEIGHT_GABLED : BUILDING_ROOF_HEIGHT_FLAT) - 9, b.id, {
      fontSize: ENTITY_FONTS.buildingLabel,
      color: ENTITY_LABEL_STYLE.textColor,
      backgroundColor: ENTITY_LABEL_STYLE.buildingBg,
      padding: ENTITY_LABEL_STYLE.padding,
    });
    label.setOrigin(0.5);
    const dot = scene.add.circle(0, bottom + 1, 3, ENTITY_COLORS.buildingDotFill, ENTITY_COLORS.buildingDotOpacity);
    dot.setStrokeStyle(1, ENTITY_COLORS.shadow, 0.9);
    container.add([emoji, label, dot]);
    return container;
  }
}
