import type { Scenario } from "../types/scenario";
import { hasAsset, allAssets } from "./assetRegistry";
import { STORAGE_KEYS } from "./storageKeys";
import { ASSET_FETCH_DELAY_MS } from "../engine/timingConstants";

const CACHE_KEY = STORAGE_KEYS.assetCache;

type CacheMap = Record<string, number>;

function loadCacheMap(): Map<string, number> {
  const m = new Map<string, number>();
  try {
    const raw = localStorage.getItem(CACHE_KEY);
    if (raw) {
      const obj = JSON.parse(raw) as CacheMap;
      for (const [k, v] of Object.entries(obj)) m.set(k, v);
    }
  } catch {
    // ignore
  }
  return m;
}

function saveCacheMap(map: Map<string, number>): void {
  try {
    const obj: CacheMap = {};
    for (const [k, v] of map.entries()) obj[k] = v;
    localStorage.setItem(CACHE_KEY, JSON.stringify(obj));
  } catch {
    // quota exceeded — ignore
  }
}

function totalRegistryCount(): number {
  return allAssets().length;
}

export class AssetStreamer {
  private cache: Map<string, number>;
  private listeners: Set<(status: string) => void> = new Set();

  constructor() {
    this.cache = loadCacheMap();
    // Seed unknown assets as not cached; only mark truly seen assets.
    // If cache empty on first run, hydrate from registry after simulated fetch
  }

  onStatus(cb: (status: string) => void): () => void {
    this.listeners.add(cb);
    return () => this.listeners.delete(cb);
  }

  private emit(msg: string): void {
    for (const cb of this.listeners) cb(msg);
  }

  getRequiredAssetIds(scenario: Scenario): string[] {
    const ids = new Set<string>();
    for (const b of scenario.buildings) {
      if (b.typeAssetId) ids.add(b.typeAssetId);
    }
    for (const o of scenario.objects) {
      if (o.assetId) ids.add(o.assetId);
    }
    for (const c of scenario.characters) {
      const sid = c.appearance?.spriteId;
      if (sid) ids.add(sid);
    }
    return [...ids];
  }

  checkCache(ids: string[]): string[] {
    return ids.filter((id) => !this.cache.has(id));
  }

  cachedCount(): number {
    return this.cache.size;
  }

  totalCount(): number {
    return totalRegistryCount();
  }

  isCached(id: string): boolean {
    return this.cache.has(id);
  }

  async fetchMissing(ids: string[]): Promise<void> {
    if (ids.length === 0) return;
    const missingValid = ids.filter((id) => hasAsset(id));
    if (missingValid.length === 0) return;
    this.emit(`Downloading ${missingValid.length} asset(s)...`);
    // Simulate network download per asset; design for future bundle URLs
    const promises = missingValid.map(
      (id) =>
        new Promise<void>((resolve) => {
          window.setTimeout(() => {
            this.cache.set(id, Date.now());
            resolve();
          }, ASSET_FETCH_DELAY_MS);
        }),
    );
    await Promise.all(promises);
    saveCacheMap(this.cache);
    this.emit(this.statusTextForScenario(null));
  }

  statusTextForScenario(scenario: Scenario | null): string {
    if (!scenario) {
      return `Assets cached: ${this.cachedCount()}/${this.totalCount()}`;
    }
    const required = this.getRequiredAssetIds(scenario);
    const missing = this.checkCache(required);
    if (missing.length === 0) {
      return `Assets cached: ${required.length}/${required.length} (ready) · total ${this.cachedCount()}/${this.totalCount()}`;
    }
    return `Downloading ${missing.length} asset(s)... · cached ${required.length - missing.length}/${required.length}`;
  }

  shortStatusForScenario(scenario: Scenario): string {
    const required = this.getRequiredAssetIds(scenario);
    const missing = this.checkCache(required);
    if (missing.length === 0) return `Assets cached: ${required.length}/${required.length}`;
    return `Downloading ${missing.length} asset(s)...`;
  }

  async preloadScenarioAssets(scenario: Scenario): Promise<void> {
    const required = this.getRequiredAssetIds(scenario);
    const missing = this.checkCache(required);
    if (missing.length === 0) {
      this.emit(`Assets cached: ${required.length}/${required.length}`);
      return;
    }
    this.emit(`Downloading ${missing.length} asset(s)...`);
    await this.fetchMissing(missing);
    this.emit(`Assets cached: ${required.length}/${required.length}`);
  }

  clearCache(): void {
    this.cache.clear();
    try {
      localStorage.removeItem(CACHE_KEY);
    } catch {
      // ignore
    }
    this.emit(`Assets cached: 0/${this.totalCount()}`);
  }

  /** Pre-warm: mark all registry assets cached after initial bulk fetch simulation */
  async warmAll(): Promise<void> {
    const all = allAssets().map((a) => a.id);
    const missing = this.checkCache(all);
    if (missing.length > 0) await this.fetchMissing(missing);
  }
}

export const assetStreamer = new AssetStreamer();
