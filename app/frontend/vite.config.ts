import { defineConfig } from "vite";

// The frontend is built into ../dist so the Go BFF (app/cmd/jansvca-app) can
// serve it via JANSVCA_SPA_DIR=./app/dist. During `vite dev` the dev server
// proxies /api to the BFF on :8080.
export default defineConfig({
  build: {
    outDir: "../dist",
    emptyOutDir: true,
    sourcemap: true,
  },
  server: {
    port: 5173,
    proxy: {
      "/api": {
        target: "http://localhost:8080",
        changeOrigin: true,
      },
    },
  },
});
