import "@fontsource-variable/geist";
import "@fontsource-variable/geist-mono";
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
