import type { WidgetData } from "@/widgets";
import type { Item } from "@/types";

// JSON value — renders the scalar the backend extracted at config.path.
export function JsonPathWidget({ item, data, error }: {
	item: Item;
	data: WidgetData | undefined;
	error: string | undefined;
}) {
	const label = typeof data?.label === "string" && data.label ? data.label : item.name;
	const value = data?.value;
	return (
		<div className="flex min-w-0 flex-col justify-center gap-0.5">
			<span className="truncate text-[10.5px] uppercase tracking-[0.1em] text-text-faint">
				{label}
			</span>
			<span className="truncate font-mono text-[18px] font-medium leading-tight">
				{error ? "err" : value === undefined ? "…" : typeof value === "object" ? JSON.stringify(value) : String(value)}
			</span>
		</div>
	);
}
