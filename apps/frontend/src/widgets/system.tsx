import type { WidgetData } from "@/widgets";
import type { Item } from "@/types";
import { cn } from "@/lib/utils";

// Board tile for a monitored system: status dot + cpu/mem/disk mini bars.
// config.systemId binds it; data comes from the system fetcher.
export function SystemWidget({ item, data, error }: {
	item: Item;
	data: WidgetData | undefined;
	error: string | undefined;
}) {
	const status = typeof data?.status === "string" ? data.status : undefined;
	const sysId = typeof item.config?.systemId === "string" ? item.config.systemId : "";
	const cpu = typeof data?.cpu === "number" ? data.cpu : undefined;
	const memTotal = typeof data?.memTotal === "number" ? data.memTotal : 0;
	const memUsed = typeof data?.memUsed === "number" ? data.memUsed : 0;
	const diskTotal = typeof data?.diskTotal === "number" ? data.diskTotal : 0;
	const diskUsed = typeof data?.diskUsed === "number" ? data.diskUsed : 0;
	const memPct = memTotal > 0 ? (memUsed / memTotal) * 100 : undefined;
	const diskPct = diskTotal > 0 ? (diskUsed / diskTotal) * 100 : undefined;

	const bar = (pct: number | undefined, label: string) => (
		<span className="flex items-center gap-1" title={`${label}: ${pct !== undefined ? pct.toFixed(0) + "%" : "—"}`}>
			<span className="font-mono text-[9px] uppercase text-text-faint">{label}</span>
			<span className="h-1 w-8 overflow-hidden rounded-full bg-surface-hover">
				<span
					className={cn("block h-full rounded-full", pct !== undefined && pct > 90 ? "bg-down" : "bg-up")}
					style={{ width: `${Math.min(pct ?? 0, 100)}%` }}
				/>
			</span>
		</span>
	);

	return (
		<a
			href={sysId ? `/systems/${sysId}` : "/systems"}
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
			{error ? (
				<span className="shrink-0 font-mono text-[10.5px] text-text-faint">err</span>
			) : (
				<span className="flex shrink-0 items-center gap-2">
					{bar(cpu, "cpu")}
					{bar(memPct, "mem")}
					{bar(diskPct, "dsk")}
				</span>
			)}
		</a>
	);
}
