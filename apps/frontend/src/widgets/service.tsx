import type { Item } from "@/types";
import type { WidgetData } from "@/widgets";
import { cn } from "@/lib/utils";
import { t } from "@/i18n";

// Shared layout for the *arr/autobrr/bazarr/overseerr service widgets.
// Every backend fetcher returns the same flat svcData shape; whichever
// fields are set get a line on the right.

function num(d: WidgetData | undefined, k: string): number | undefined {
	const v = d?.[k];
	return typeof v === "number" ? v : undefined;
}

function str(d: WidgetData | undefined, k: string): string | undefined {
	const v = d?.[k];
	return typeof v === "string" && v ? v : undefined;
}

export function ServiceWidget({ item, data, error }: {
	item: Item;
	data: WidgetData | undefined;
	error: string | undefined;
}) {
	if (error) {
		return <span className="font-mono text-[11px] text-text-faint">{item.name} — {error}</span>;
	}
	const status = str(data, "status");
	const version = str(data, "version");
	const warning = str(data, "warning");
	const issues = num(data, "issues") ?? 0;
	const update = data?.updateAvailable === true;

	const name = str(data, "name");
	const stats: string[] = [];
	const queue = num(data, "queueTotal");
	const pending = num(data, "pendingRequests");
	const indexers = num(data, "indexers");
	const releases = num(data, "releases");
	const streams = num(data, "streams");
	const up = num(data, "up");
	const total = num(data, "total");
	const speed = str(data, "speed");
	const paused = data?.paused === true;
	if (name) stats.push(name);
	if (queue) stats.push(t("widget.queue", { n: queue }));
	if (pending) stats.push(t("widget.pending", { n: pending }));
	if (indexers) stats.push(t("widget.indexers", { n: indexers }));
	if (releases) stats.push(t("widget.releases", { n: releases }));
	if (streams) stats.push(t("widget.streams", { n: streams }));
	if (up !== undefined && total !== undefined && total > 0) stats.push(t("widget.upOf", { up, total }));
	if (speed) stats.push(speed);
	if (paused) stats.push(t("widget.paused"));
	if (issues) stats.push(t("widget.issues", { n: issues }));
	if (update) stats.push(t("widget.update"));

	return (
		<div className="flex min-w-0 items-center gap-4">
			<div className="flex min-w-0 items-center gap-2">
				<span
					className={cn(
						"size-1.5 shrink-0 rounded-full",
						status === "online" ? "bg-up" : status === "warning" ? "bg-down" : "bg-text-faint/40",
					)}
				/>
				<span className="truncate font-mono text-[15px] font-medium leading-tight">
					{status ?? "…"}
				</span>
			</div>
			<div className="ml-auto flex min-w-0 flex-col items-end gap-0.5 font-mono text-[10.5px] text-text-faint">
				{version && <span className="truncate">v{version}</span>}
				{stats.map((s) => <span key={s} className="truncate">{s}</span>)}
				{warning && <span className="max-w-[180px] truncate text-down">{warning}</span>}
			</div>
		</div>
	);
}
