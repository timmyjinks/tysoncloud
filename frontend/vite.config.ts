import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import { tanstackStart } from "@tanstack/react-start/plugin/vite";
import { nitro } from "nitro/vite";
import path from "path";

// TanStack Start, SPA mode + Nitro (standard Node/Docker serving via
// `node .output/server/index.mjs`). Client-rendered; the server exists for
// server functions (runtime envs).
export default defineConfig({
  plugins: [
    tanstackStart({ spa: { enabled: true } }), // MUST come before react()
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
