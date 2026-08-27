import { getAsset, getIcon, isSolid } from "../assets/assetRegistry";
import type { AssetEntry } from "../assets/assetRegistry";

export interface ResolvedAsset {
  entry: AssetEntry | null;
  icon: string;
  solid: boolean;
  bundle: string | null;
}

const cache = new Map<string, ResolvedAsset>();

export function resolveAsset(assetId: string): ResolvedAsset {
  const cached = cache.get(assetId);
  if (cached) return cached;
  const entry = getAsset(assetId) ?? null;
  const res: ResolvedAsset = {
    entry,
    icon: entry ? entry.icon : getIcon(assetId) || "❓",
    solid: entry ? entry.solid : isSolid(assetId),
    bundle: entry ? entry.bundle : null,
  };
  // fallback icon
  if (!entry) {
    res.icon = "❓";
    res.solid = false;
  }
  cache.set(assetId, res);
  return res;
}

export function clearCache(): void {
  cache.clear();
}

export function resolveSpritePath(assetId: string): string | null {
  const e = getAsset(assetId);
  return e ? e.sprite : null;
}
