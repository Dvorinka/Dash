import { bento } from "./bento";
import { cards } from "./cards";
import { editorial } from "./editorial";
import { terminal } from "./terminal";
import type { RendererViews } from "./types";
import type { RendererName } from "@/types";

// Renderer registry — settings.renderer picks the active one.
export const rendererList = ["bento", "cards", "index", "console"] as const satisfies readonly RendererName[];
export const renderers: Record<RendererName, RendererViews> = {
	bento,
	cards,
	index: editorial,
	console: terminal,
};

export const rendererLabels: Record<RendererName, string> = {
	bento: "Bento",
	cards: "Cards",
	index: "Index",
	console: "Console",
};

export const rendererDescriptions: Record<RendererName, string> = {
	bento: "mixed-size tile grid",
	cards: "uniform compact cards",
	index: "editorial list · masthead",
	console: "terminal panels · mono",
};
