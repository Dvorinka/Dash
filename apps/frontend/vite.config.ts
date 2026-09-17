import path from "node:path";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import { defineConfig } from "vite";

export default defineConfig({
	plugins: [react(), tailwindcss()],
	resolve: {
		alias: { "@": path.resolve(__dirname, "src") },
	},
	server: {
		// Backend API during development (DASH_DEV=1 go run ./cmd/dash).
		proxy: { "/api": "http://localhost:8080" },
	},
});
