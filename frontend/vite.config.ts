import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import { tanstackStart } from "@tanstack/react-start/plugin/vite";
import { nitro } from "nitro/vite";
import path from "path";

// TanStack Start in SPA mode (see src/start.ts: defaultSsr false) —
// client-rendered like before, plus a Node server for server functions
// (runtime envs, etc.). Served with `node .output/server/index.mjs`.
export default defineConfig({
  plugins: [
    tanstackStart(), // MUST come before react(); includes file-based routing
    nitro(),
    react(),
    tailwindcss(),
  ],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  server: {
    port: 3000,
  },
});
