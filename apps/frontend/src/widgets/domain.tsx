import type { WidgetData } from "@/widgets";
import type { Item } from "@/types";
import { cn } from "@/lib/utils";

// Board tile for a tracked domain: favicon, name, expiry countdown.
export function DomainWidget({ item, data, error }: {
	item: Item;
	data: WidgetData | undefined;
	error: string | undefined;
}) {
	const days = typeof data?.daysUntilExpiry === "number" ? data.daysUntilExpiry : undefined;
	const sslDays = typeof data?.sslDaysUntilExpiry === "number" ? data.sslDaysUntilExpiry : undefined;
	const expiring = data?.expiring === true;
	const domainId = typeof item.config?.domainId === "string" ? item.config.domainId : "";

	return (
		<a
			href={domainId ? `/domains/${domainId}` : "/domains"}
			className="flex w-full items-center gap-2.5 text-left"
			onClick={(e) => e.stopPropagation()}
		>
			{typeof data?.faviconUrl === "string" && data.faviconUrl ? (
				<img src={data.faviconUrl} alt="" className="size-4 shrink-0 rounded-sm" loading="lazy" />
			) : null}
			<span className="min-w-0 flex-1 truncate text-[12.5px] font-medium">
				{typeof data?.name === "string" ? data.name : item.name}
			</span>
			<span className={cn(
				"shrink-0 font-mono text-[10.5px]",
				error ? "text-down" : expiring || (days !== undefined && days < 0) ? "text-down" : "text-text-faint",
			)}>
				{error ? "err" : days === undefined ? "…" : days < 0 ? "expired" : `${days}d`}
				{sslDays !== undefined && sslDays <= 30 ? ` · ssl ${sslDays}d` : ""}
			</span>
		</a>
	);
}
