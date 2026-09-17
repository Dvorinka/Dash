import type { Item } from "@/types";
import type { WidgetData } from "@/widgets/index";

// Shared DNS-blocker layout: big block-rate, queries + blocked counts.
// Pi-hole and AdGuard return the same fields from their fetchers.

function num(d: WidgetData | undefined, k: string): number {
	const v = d?.[k];
	return typeof v === "number" ? v : 0;
}

const fmt = (n: number) =>
	n >= 1_000_000 ? `${(n / 1_000_000).toFixed(1)}M` : n >= 1000 ? `${(n / 1000).toFixed(1)}k` : String(n);

function BlockerWidget({ item, data, error }: { item: Item; data: WidgetData | undefined; error: string | undefined }) {
	if (error) {
		return <span className="font-mono text-[11px] text-text-faint">{item.name} — {error}</span>;
	}
	const pct = num(data, "blockedPercent");
	return (
		<div className="flex min-w-0 items-center gap-4">
			<div className="flex min-w-0 flex-col gap-0.5">
				<span className="font-mono text-[22px] font-medium leading-none tracking-tight tabular-nums">
					{data ? `${pct.toFixed(1)}%` : "…"}
				</span>
				<span className="text-[10.5px] uppercase tracking-[0.1em] text-text-faint">blocked</span>
			</div>
			<div className="ml-auto flex flex-col items-end gap-1 font-mono text-[10.5px] text-text-faint">
				<span>{fmt(num(data, "queriesToday"))} queries</span>
				<span>{fmt(num(data, "blockedToday"))} blocked</span>
				{typeof data?.avgProcessMs === "number" && <span>{num(data, "avgProcessMs").toFixed(0)} ms avg</span>}
			</div>
		</div>
	);
}

export function PiholeWidget(p: { item: Item; data: WidgetData | undefined; error: string | undefined }) {
	return <BlockerWidget {...p} />;
}

export function AdguardWidget(p: { item: Item; data: WidgetData | undefined; error: string | undefined }) {
	return <BlockerWidget {...p} />;
}
