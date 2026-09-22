import "@fontsource-variable/geist";
import "@fontsource-variable/geist-mono";
import "@fontsource/instrument-serif";
import "@fontsource-variable/jetbrains-mono";
import "./index.css";

import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import App from "./App.tsx";

const root = document.getElementById("root");
if (!root) throw new Error("#root missing");

createRoot(root).render(
	<StrictMode>
		<App />
	</StrictMode>,
);

// PWA: register the service worker only in production builds — in dev it would
// fight Vite's HMR by caching modules.
if (import.meta.env.PROD && "serviceWorker" in navigator) {
	window.addEventListener("load", () => void navigator.serviceWorker.register("/sw.js"));
}
