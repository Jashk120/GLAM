// Centralized env/config — env-driven ports/hosts, API base URL
// Vite exposes VITE_ prefixed vars via import.meta.env; fallback to defaults for dev.
// vite.config.ts, TeacherUI.ts and main.ts import from here so ports are not duplicated.

const viteEnv = (import.meta as unknown as { env: Record<string, string | undefined> }).env;

export const API_CONFIG = {
  apiBaseUrl: viteEnv.VITE_API_BASE_URL ?? "http://localhost:8080",
  generateUrl: "/api/scenario/generate" as const,
  assetsUrl: "/api/assets" as const,
  healthUrl: "/health" as const,
} as const;

// Derive server origin for display (e.g., "http://localhost:8080" → ":8080" suffix)
export function serverOriginFromConfig(): string {
  try {
    const u = new URL(API_CONFIG.apiBaseUrl);
    return `:${u.port || "8080"}`;
  } catch {
    return ":8080";
  }
}

export function apiBaseUrl(): string {
  return API_CONFIG.apiBaseUrl;
}

// Client dev server port (Vite) — defaults mirror dev.sh
export const DEV_PORTS = {
  server: Number(viteEnv.VITE_SERVER_PORT ?? 8080),
  client: Number(viteEnv.VITE_PORT ?? 5173),
} as const;
