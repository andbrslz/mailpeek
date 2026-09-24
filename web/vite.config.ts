import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

const target = process.env.MAILPEEK_URL ?? "http://localhost:8026";

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    proxy: { "/api": { target, changeOrigin: true } },
  },
  build: {
    outDir: "dist",
    emptyOutDir: true,
    copyPublicDir: true,
  },
});
