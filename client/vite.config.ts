import { defineConfig, loadEnv } from "vite";

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, ".", "");
  const clientPort = Number(env.VITE_PORT ?? env.PORT_CLIENT ?? 5173);
  const serverPort = Number(env.PORT ?? env.VITE_SERVER_PORT ?? 8080);
  const apiTarget = env.VITE_API_BASE_URL ?? `http://localhost:${serverPort}`;
  return {
    server: {
      host: true,
      port: clientPort,
      proxy: {
        "/api": {
          target: apiTarget,
          changeOrigin: true,
        },
      },
    },
    publicDir: "public",
  };
});
