import type { Item } from "@/types";
import type { WidgetData } from "@/widgets/index";

// Immich — photo/video counts and library footprint.

function num(d: WidgetData | undefined, k: string): number {
	const v = d?.[k];
	return typeof v === "number" ? v : 0;
}

const fmtNum = (n: number) =>
	n >= 1_000_000 ? `${(n / 1_000_000).toFixed(1)}M` : n >= 1000 ? `${(n / 1000).toFixed(1)}k` : String(n);

const fmtBytes = (b: number) =>
	b >= 1 << 30 ? `${(b / (1 << 30)).toFixed(1)} GB` : `${(b / (1 << 20)).toFixed(0)} MB`;

export function ImmichWidget({ item, data, error }: { item: Item; data: WidgetData | undefined; error: string | undefined }) {
	if (error) {
		return <span className="font-mono text-[11px] text-text-faint">{item.name} — {error}</span>;
	}
	return (
		<div className="flex min-w-0 items-center gap-4">
			<div className="flex min-w-0 flex-col gap-0.5">
				<span className="font-mono text-[22px] font-medium leading-none tracking-tight tabular-nums">
					{data ? fmtNum(num(data, "photos")) : "…"}
				</span>
				<span className="text-[10.5px] uppercase tracking-[0.1em] text-text-faint">photos</span>
			</div>
			<div className="ml-auto flex flex-col items-end gap-1 font-mono text-[10.5px] text-text-faint">
				<span>{fmtNum(num(data, "videos"))} videos</span>
				<span>{fmtBytes(num(data, "usageBytes"))}</span>
			</div>
		</div>
	);
}
