import { useEffect, useState } from "react";
import type { Item } from "@/types";
import { cfgStr } from "@/widgets/index";

// Clock — fully client-side; ticks once a second.
export function ClockWidget({ item }: { item: Item }) {
	const tz = cfgStr(item, "timezone") || undefined;
	const [now, setNow] = useState(() => new Date());

	useEffect(() => {
		const t = setInterval(() => setNow(new Date()), 1000);
		return () => clearInterval(t);
	}, []);

	const time = new Intl.DateTimeFormat(undefined, {
		hour: "2-digit", minute: "2-digit", second: "2-digit", hour12: false, timeZone: tz,
	}).format(now);
	const date = new Intl.DateTimeFormat(undefined, {
		weekday: "short", month: "short", day: "numeric", timeZone: tz,
	}).format(now);

	return (
		<div className="flex min-w-0 flex-col justify-center gap-0.5">
			<span className="font-mono text-[22px] font-medium leading-none tracking-tight tabular-nums">
				{time}
			</span>
			<span className="truncate text-[10.5px] uppercase tracking-[0.1em] text-text-faint">
				{date}{tz ? ` · ${tz.split("/").pop()?.replace("_", " ")}` : ""}
			</span>
		</div>
	);
}
