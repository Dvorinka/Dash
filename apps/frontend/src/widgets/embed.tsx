import type { Item } from "@/types";
import { cfgStr } from "@/widgets/index";
import { t } from "@/i18n";

// Embed — sandboxed iframe. Fully client-side; the tile chrome still shows
// the item name above the frame.
export function EmbedWidget({ item }: { item: Item }) {
	const url = cfgStr(item, "url");
	// Block script-capable schemes; http(s)/relative URLs are fine.
	const bad = /^\s*(javascript|data|vbscript|file):/i.test(url);
	if (!url || bad) {
		return <span className="text-[11px] text-text-faint">{t("widget.embedHint")}</span>;
	}
	return (
		<iframe
			src={url}
			title={item.name || "embed"}
			sandbox="allow-scripts allow-same-origin allow-forms"
			referrerPolicy="no-referrer"
			className="pointer-events-auto h-full min-h-[120px] w-full rounded-[6px] border border-border bg-bg"
		/>
	);
}
