import { bento } from "./bento";
import { cards } from "./cards";
import type { RendererViews } from "./types";
import type { RendererName } from "@/types";

// Renderer registry — settings.renderer picks the active one.
export const rendererList = ["bento", "cards"] as const satisfies readonly RendererName[];
export const renderers: Record<RendererName, RendererViews> = { bento, cards };

export const rendererLabels: Record<RendererName, string> = {
	bento: "Bento",
	cards: "Cards",
};
