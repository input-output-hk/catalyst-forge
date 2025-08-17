import { defineConfig } from "vite";
import react from "@vitejs/plugin-react-swc";
import path from "path";
import { componentTagger } from "lovable-tagger";

// https://vitejs.dev/config/
export default defineConfig(({ mode }) => ({
  server: {
    host: "::",
    port: 8080,
  },
  plugins: [
    react(),
    mode === 'development' &&
    componentTagger(),
  ].filter(Boolean),
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
      // Point module alias at the directory so TS can find index.d.ts and schema.d.ts
      "forge-client": path.resolve(__dirname, "./vendor/forge-client"),
    },
  },
  build: {
    rollupOptions: {
      // In case the vendored client is not present in some environments, don't fail build
      external: [
        // Keep empty; alias points to a file. If missing, externalize to avoid hard failure.
      ],
    },
  },
}));
