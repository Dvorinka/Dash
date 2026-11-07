import { useMemo } from "react";
import type { DevDayCount } from "@/types";
import { cn } from "@/lib/utils";

// GitHub-style contribution heatmap: ~53 week columns × 7 day rows,
// last year ending today. Cell intensity follows count quartiles.

const DAY = 86_400_000;

function level(count: number, max: number): number {
	if (count <= 0 || max <= 0) return 0;
	const r = count / max;
	if (r > 0.75) return 4;
	if (r > 0.5) return 3;
	if (r > 0.25) return 2;
	return 1;
}

const OPACITY = [0.25, 0.4, 0.6, 0.8, 1];

export function Heatmap({ days }: { days: DevDayCount[] }) {
	const { weeks, max, total } = useMemo(() => {
		const byDate = new Map(days.map((d) => [d.date, d.count]));
		const today = new Date();
		today.setHours(0, 0, 0, 0);
		// align start to Sunday, 371 days back (53 weeks)
		const start = new Date(today.getTime() - 370 * DAY);
		start.setDate(start.getDate() - start.getDay());
		let max = 0;
		let total = 0;
		const cols: { date: string; count: number; future: boolean }[][] = [];
		for (let w = 0; w < 53; w++) {
			const col: { date: string; count: number; future: boolean }[] = [];
			for (let d = 0; d < 7; d++) {
				const t = start.getTime() + (w * 7 + d) * DAY;
				const iso = new Date(t).toISOString().slice(0, 10);
				const count = byDate.get(iso) ?? 0;
				max = Math.max(max, count);
				total += count;
				col.push({ date: iso, count, future: t > today.getTime() });
			}
			cols.push(col);
		}
		return { weeks: cols, max, total };
	}, [days]);

	const monthLabels = useMemo(() => {
		const out: { col: number; label: string }[] = [];
		let lastMonth = -1;
		weeks.forEach((col, wi) => {
			const first = col.find((c) => !c.future) ?? col[0] ?? { date: "", count: 0, future: false };
			const d = new Date(first.date + "T00:00:00");
			if (d.getMonth() !== lastMonth) {
				out.push({ col: wi, label: d.toLocaleString(undefined, { month: "short" }) });
				lastMonth = d.getMonth();
			}
		});
		return out;
	}, [weeks]);

	return (
		<div>
			<div className="mb-1.5 flex items-center justify-between font-mono text-[10.5px] text-text-faint">
				<span>{total} contributions in the last year</span>
				<span>Less</span>
			</div>
			<div className="overflow-x-auto pb-1">
				<div className="inline-flex gap-[3px]">
					<div className="mr-1 flex flex-col justify-between py-0.5 font-mono text-[8.5px] leading-none text-text-faint">
						<span>Mon</span><span>Wed</span><span>Fri</span>
					</div>
					<div>
						<div className="relative mb-[3px] h-3">
							{monthLabels.map((m) => (
								<span key={m.col} className="absolute font-mono text-[9px] text-text-faint"
									style={{ left: m.col * 13 }}>
									{m.label}
								</span>
							))}
						</div>
						<div className="flex gap-[3px]">
							{weeks.map((col, wi) => (
								<div key={wi} className="flex flex-col gap-[3px]">
									{col.map((c) => (
										<span
											key={c.date}
											title={`${c.date}: ${c.count}`}
											className={cn(
												"block size-[10px] rounded-[2px]",
												c.future || c.count === 0 ? "bg-surface-hover" : "bg-up",
											)}
											style={c.future || c.count === 0 ? undefined : { opacity: OPACITY[level(c.count, max) - 1] }}
										/>
									))}
								</div>
							))}
						</div>
					</div>
				</div>
			</div>
			<div className="mt-1 flex items-center justify-end gap-1 font-mono text-[10px] text-text-faint">
				<span>Less</span>
				{OPACITY.map((o) => (
					<span key={o} className="size-[10px] rounded-[2px] bg-up" style={{ opacity: o }} />
				))}
				<span>More</span>
			</div>
		</div>
	);
}
