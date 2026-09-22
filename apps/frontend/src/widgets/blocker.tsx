import type { Item } from "@/types";
import type { WidgetData } from "@/widgets/index";
import { t } from "@/i18n";

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
				<span className="text-[10.5px] uppercase tracking-[0.1em] text-text-faint">{t("widget.blockedLbl")}</span>
			</div>
			<div className="ml-auto flex flex-col items-end gap-1 font-mono text-[10.5px] text-text-faint">
				<span>{t("widget.queries", { n: fmt(num(data, "queriesToday")) })}</span>
				<span>{t("widget.blocked", { n: fmt(num(data, "blockedToday")) })}</span>
				{typeof data?.avgProcessMs === "number" && <span>{t("widget.msAvg", { n: num(data, "avgProcessMs").toFixed(0) })}</span>}
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
