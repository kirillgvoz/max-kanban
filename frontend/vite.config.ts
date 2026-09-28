import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  base: "/max-kanban/",
  plugins: [react()],
  server: {
    port: 3002,
    proxy: {
      "/max-kanban/api": "http://127.0.0.1:9300/api",
      "/max-kanban/webhook": "http://127.0.0.1:9300/webhook",
    },
  },
  build: {
    outDir: "dist",
    sourcemap: false,
  },
});
