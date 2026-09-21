import type { WidgetData } from "@/widgets";
import type { Item } from "@/types";
import { cn } from "@/lib/utils";

// Board tile for an uptime monitor: status dot, uptime %, avg ping.
// config.monitorId binds it; data comes from the monitor fetcher.
export function MonitorWidget({ item, data, error }: {
	item: Item;
	data: WidgetData | undefined;
	error: string | undefined;
}) {
	const status = typeof data?.status === "string" ? data.status : undefined;
	const uptime = typeof data?.uptime24 === "number" ? data.uptime24 : undefined;
	const ping = typeof data?.ping === "number" ? data.ping : undefined;
	const monId = typeof item.config?.monitorId === "string" ? item.config.monitorId : "";

	return (
		<a
			href={monId ? `/monitors/${monId}` : "/monitors"}
			className="flex w-full items-center gap-2.5 text-left"
			onClick={(e) => e.stopPropagation()}
		>
			<span
				className={cn(
					"size-1.5 shrink-0 rounded-full",
					error ? "bg-down" : status === "up" ? "bg-up" : status === "down" ? "bg-down" : "bg-text-faint/40",
				)}
			/>
			<span className="min-w-0 flex-1 truncate text-[12.5px] font-medium">
				{typeof data?.name === "string" ? data.name : item.name}
			</span>
			<span className="shrink-0 font-mono text-[10.5px] text-text-faint">
				{error ? "err" : uptime !== undefined ? `${uptime.toFixed(1)}%` : "…"}
				{ping ? ` · ${Math.round(ping)}ms` : ""}
			</span>
		</a>
	);
}
