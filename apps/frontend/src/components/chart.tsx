// Shared time-series chart primitives — the Beszel-grade look on our tokens:
// titled cards with a subtitle, soft area fills over a real time axis,
// unit-aware Y ticks, a dotted tooltip sorted by value, and legend chips.
import type { ReactNode } from "react";
import {
	Area,
	CartesianGrid,
	ComposedChart,
	Line,
	ResponsiveContainer,
	Tooltip,
	XAxis,
	YAxis,
} from "recharts";
import { cn } from "@/lib/utils";
import { t } from "@/i18n";

/** One chart row: epoch ms plus named series values. */
export type ChartRow = { t: number } & Record<string, number | undefined>;

export interface Series {
	key: string;
	label: string;
	color: string;
	/** Shared value → stacked areas (container charts). */
	stack?: string;
	/** Render as a bare line, no fill (temperatures). */
	line?: boolean;
}

export const CHART_COLORS = [
	"var(--chart-1)",
	"var(--chart-2)",
	"var(--chart-3)",
	"var(--chart-4)",
	"var(--chart-5)",
	"var(--chart-6)",
	"var(--chart-7)",
	"var(--chart-8)",
];
export const chartColor = (i: number) => CHART_COLORS[i % CHART_COLORS.length] ?? "var(--accent)";

const axisTick = { fontSize: 10, fill: "var(--text-faint)" };

/** Auto Y-domain with headroom, rounding to a readable ceiling. */
export function niceMax(dataMax: number): number {
	if (!Number.isFinite(dataMax) || dataMax <= 0) return 1;
	if (dataMax > 10) return Math.ceil(dataMax * 1.05);
	if (dataMax > 1) return Math.ceil(dataMax * 10.5) / 10;
	return Math.ceil(dataMax * 105) / 100;
}

/** Tick format by range: HH:mm up to 2 days, day+time to 2 weeks, then dates. */
export function xTick(spanMs: number): (v: number) => string {
	if (spanMs <= 48 * 3600_000)
		return (v) => new Date(v).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
	if (spanMs <= 14 * 86400_000)
		return (v) =>
			new Date(v).toLocaleString([], { month: "short", day: "numeric", hour: "2-digit", minute: "2-digit" });
	return (v) => new Date(v).toLocaleDateString([], { month: "short", day: "numeric" });
}

const fullTime = (ms: number) =>
	new Date(ms).toLocaleString([], {
		month: "short",
		day: "numeric",
		hour: "2-digit",
		minute: "2-digit",
		second: "2-digit",
	});

interface TipItem {
	name?: string;
	value?: number | string;
	color?: string;
	stroke?: string;
	dataKey?: string | number;
	payload?: ChartRow;
}

/** Dotted tooltip: timestamp header, per-series rows sorted by value desc. */
function ChartTip({
	active,
	payload,
	fmt,
}: {
	active?: boolean;
	payload?: TipItem[];
	fmt: (v: number) => string;
}) {
	if (!active || !payload?.length) return null;
	const t0 = payload[0]?.payload?.t;
	const rows = payload
		.filter((p): p is TipItem & { value: number } => typeof p.value === "number")
		.sort((a, b) => b.value - a.value);
	if (!rows.length) return null;
	return (
		<div className="min-w-28 rounded-lg border border-border-strong bg-popover px-2.5 py-1.5 text-[11px] shadow-xl">
			{t0 !== undefined && <div className="mb-1 font-medium text-text-dim">{fullTime(t0)}</div>}
			<div className="grid gap-1">
				{rows.map((p) => (
					<div key={String(p.dataKey ?? p.name)} className="flex items-center gap-2">
						<span
							className="size-2 shrink-0 rounded-[2px]"
							style={{ background: p.color ?? p.stroke ?? "var(--text-faint)" }}
						/>
						<span className="flex-1 truncate text-text-faint">{p.name}</span>
						<span className="font-mono font-medium">{fmt(p.value)}</span>
					</div>
				))}
			</div>
		</div>
	);
}

/** Area/line chart on a time-scaled X axis. Empty data renders a hint, not a blank box. */
export function MetricChart({
	rows,
	series,
	fmt,
	domain,
	spanMs,
}: {
	rows: ChartRow[];
	series: Series[];
	fmt: (v: number) => string;
	domain?: [number, number] | undefined;
	spanMs: number;
}) {
	if (!rows.length) {
		return (
			<div className="flex h-full items-center justify-center text-[11px] text-text-faint">
				{t("chart.noData")}
			</div>
		);
	}
	return (
		<ResponsiveContainer width="100%" height="100%">
			<ComposedChart data={rows} margin={{ top: 4, right: 8, bottom: 0, left: 0 }}>
				<CartesianGrid stroke="var(--border)" strokeDasharray="3 3" vertical={false} />
				<XAxis
					dataKey="t"
					type="number"
					scale="time"
					domain={["dataMin", "dataMax"]}
					tick={axisTick}
					tickLine={false}
					axisLine={false}
					tickFormatter={xTick(spanMs)}
					minTickGap={28}
				/>
				<YAxis
					tick={axisTick}
					tickLine={false}
					axisLine={false}
					width="auto"
					domain={domain ?? [0, niceMax]}
					tickFormatter={(v) => fmt(Number(v))}
				/>
				<Tooltip content={<ChartTip fmt={fmt} />} cursor={{ stroke: "var(--border-strong)" }} />
				{series.map((s) =>
					s.line ? (
						<Line
							key={s.key}
							dataKey={s.key}
							name={s.label}
							type="monotone"
							stroke={s.color}
							strokeWidth={1.5}
							dot={false}
							connectNulls
							isAnimationActive={false}
						/>
					) : (
						<Area
							key={s.key}
							dataKey={s.key}
							name={s.label}
							type="monotone"
							stroke={s.color}
							fill={s.color}
							fillOpacity={s.stack ? 0.65 : 0.14}
							strokeWidth={1.5}
							dot={false}
							{...(s.stack !== undefined ? { stackId: s.stack } : {})}
							isAnimationActive={false}
						/>
					),
				)}
			</ComposedChart>
		</ResponsiveContainer>
	);
}

/** Titled chart card in the Beszel layout: title + subtitle on top, legend chips below. */
export function ChartCard({
	title,
	hint,
	actions,
	series,
	height = 160,
	children,
}: {
	title: string;
	hint?: string;
	actions?: ReactNode;
	series?: Series[];
	height?: number;
	children: ReactNode;
}) {
	return (
		<div className="rounded-[10px] border border-border bg-surface p-3 pt-3.5">
			<div className="mb-2 flex items-start justify-between gap-2 px-1">
				<div>
					<h3 className="text-[13px] font-medium tracking-tight">{title}</h3>
					{hint ? <p className="text-[10.5px] text-text-faint">{hint}</p> : null}
				</div>
				{actions}
			</div>
			<div style={{ height }}>{children}</div>
			{series && series.length > 1 && (
				<div className="mt-2 flex flex-wrap gap-x-3.5 gap-y-1 px-1">
					{series.map((s) => (
						<span
							key={s.key}
							className="flex items-center gap-1.5 font-mono text-[10px] text-text-faint"
						>
							<span className="size-2 rounded-[2px]" style={{ background: s.color }} />
							{s.label}
						</span>
					))}
				</div>
			)}
		</div>
	);
}
