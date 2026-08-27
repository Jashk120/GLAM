import registryData from "./registry.json";

export interface AssetEntry {
  id: string;
  type: string;
  bundle: string;
  sprite: string;
  icon: string;
  solid: boolean;
  metadata: {
    description: string;
    displayName: string;
    category: string;
  };
}

const registry: AssetEntry[] = registryData as unknown as AssetEntry[];

const byId = new Map<string, AssetEntry>();
for (const e of registry) byId.set(e.id, e);

export function getAsset(id: string): AssetEntry | undefined {
  return byId.get(id);
}

export function getAssetsByBundle(bundle: string): AssetEntry[] {
  return registry.filter((e) => e.bundle === bundle);
}

export function isSolid(id: string): boolean {
  const a = byId.get(id);
  return a ? a.solid : false;
}

export function getIcon(id: string): string {
  const a = byId.get(id);
  return a ? a.icon : "❓";
}

export function hasAsset(id: string): boolean {
  return byId.has(id);
}

export function allAssets(): AssetEntry[] {
  return [...registry];
}

export function assertAssetExists(id: string): AssetEntry {
  const a = byId.get(id);
  if (!a) throw new Error(`Unknown asset id: "${id}"`);
  return a;
}
