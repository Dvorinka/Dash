import { useCallback, useEffect, useState } from "react";
import { Link } from "wouter";
import { ArrowLeft, Pause, Play, RefreshCw, Trash2 } from "lucide-react";
import { api } from "@/api";
import { ChartCard, MetricChart, chartColor, type ChartRow } from "@/components/chart";
import type { Heartbeat, MonitorView } from "@/types";
import { MonitorDialog } from "@/components/MonitorDialog";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { t } from "@/i18n";

export function MonitorDetailPage({ id }: { id: string }) {
	const [m, setM] = useState<MonitorView | null>(null);
	const [hbs, setHbs] = useState<Heartbeat[]>([]);
	const [hours, setHours] = useState(24);
	const [dlgOpen, setDlgOpen] = useState(false);

	const load = useCallback(async () => {
		const [{ data: mv }, { data: hb }] = await Promise.all([
			api.GET("/api/monitors/{id}", { params: { path: { id } } }),
			api.GET("/api/monitors/{id}/heartbeats", { params: { path: { id }, query: { hours } } }),
		]);
		if (mv) setM(mv);
		if (hb) setHbs(hb);
	}, [id, hours]);

	useEffect(() => {
		void load();
		const t = setInterval(() => void load(), 30_000);
		return () => clearInterval(t);
	}, [load]);

	if (!m) return <main className="mx-auto max-w-5xl px-7 py-8 text-[13px] text-text-faint">{t("common.loading")}</main>;

	const rows: ChartRow[] = hbs.map((h) => ({
		t: new Date(h.checkedAt ?? "").getTime(),
		ping: h.pingMs,
	}));

	return (
		<main className="mx-auto w-full max-w-5xl px-7 py-8">
			<Link href="/monitors" className="mb-4 inline-flex items-center gap-1.5 text-[12px] text-text-faint hover:text-text">
				<ArrowLeft size={12} /> {t("nav.monitors")}
			</Link>

			<div className="mb-6 flex items-start justify-between">
				<div>
					<div className="flex items-center gap-2.5">
						<span className={cn("size-2 rounded-full", m.status === "up" ? "bg-up" : m.status === "down" ? "bg-down" : "bg-text-faint/40")} />
						<h1 className="text-[17px] font-semibold tracking-tight">{m.name}</h1>
						<span className="font-mono text-[10.5px] uppercase tracking-[0.08em] text-text-faint">{m.status}</span>
					</div>
					<p className="mt-1 font-mono text-[11.5px] text-text-faint">
						{m.type} · {m.url || m.hostname}{m.port ? `:${m.port}` : ""} · {t("monitors.everyInterval", { interval: m.intervalS ?? 0 })}
					</p>
				</div>
				<div className="flex gap-2">
					<Button variant="outline" size="sm" onClick={() => void api.POST("/api/monitors/{id}/check", { params: { path: { id } } }).then(load)}>
						<RefreshCw size={12} /> {t("monitors.checkNow")}
					</Button>
					<Button variant="outline" size="sm"
						onClick={() => void api.PATCH("/api/monitors/{id}", { params: { path: { id } }, body: { active: !m.active } }).then(load)}>
						{m.active ? <Pause size={12} /> : <Play size={12} />} {m.active ? t("monitors.pause") : t("monitors.resume")}
					</Button>
					<Button variant="outline" size="sm" onClick={() => setDlgOpen(true)}>{t("common.edit")}</Button>
					<Button variant="outline" size="sm" aria-label={t("common.delete")}
						onClick={() => {
							if (confirm(t("monitors.deleteConfirm", { name: m.name ?? "" }))) {
								void api.DELETE("/api/monitors/{id}", { params: { path: { id } } }).then(() => { location.href = "/monitors"; });
							}
						}}>
						<Trash2 size={12} />
					</Button>
				</div>
			</div>

			<div className="mb-6 grid grid-cols-4 gap-3 max-[720px]:grid-cols-2">
				{[
					[t("monitors.uptime24h"), `${m.stats?.uptime24h?.toFixed(2)}%`],
					[t("monitors.uptime30d"), `${m.stats?.uptime30d?.toFixed(2)}%`],
					[t("monitors.avgLatency"), m.stats?.avgPing24h ? `${m.stats.avgPing24h} ms` : "—"],
					[t("monitors.checks"), String((m.stats?.up ?? 0) + (m.stats?.down ?? 0))],
				].map(([k, v]) => (
					<div key={k} className="rounded-[10px] border border-border bg-surface px-4 py-3">
						<div className="font-mono text-[10px] uppercase tracking-[0.1em] text-text-faint">{k}</div>
						<div className="mt-1 font-mono text-[17px] font-medium">{v}</div>
					</div>
				))}
			</div>

			<h2 className="mb-2 font-mono text-[10.5px] uppercase tracking-[0.12em] text-text-faint">{t("monitors.uptime")}</h2>
			<div className="mb-6 rounded-[10px] border border-border bg-surface px-3 py-2.5">
				<UptimeBar hbs={hbs} hours={hours} />
			</div>

			<div className="mb-6">
				<ChartCard
					title={t("monitors.responseTime")}
					height={208}
					series={[{ key: "ping", label: "ms", color: chartColor(0) }]}
					actions={
						<div className="flex gap-1">
							{[24, 168, 720].map((h) => (
								<button key={h} type="button" onClick={() => setHours(h)}
									className={cn("rounded-[6px] px-2 py-1 font-mono text-[10.5px]",
										hours === h ? "bg-surface-hover text-text" : "text-text-faint hover:text-text")}>
									{h === 24 ? "24h" : h === 168 ? "7d" : "30d"}
								</button>
							))}
						</div>
					}
				>
					<MetricChart
						rows={rows}
						series={[{ key: "ping", label: "ms", color: chartColor(0) }]}
						fmt={(v) => `${Math.round(v)} ms`}
						spanMs={hours * 3600_000}
					/>
				</ChartCard>
			</div>

			<h2 className="mb-2 font-mono text-[10.5px] uppercase tracking-[0.12em] text-text-faint">{t("monitors.recentChecks")}</h2>
			<div className="overflow-hidden rounded-[10px] border border-border">
				<table className="w-full text-[12px]">
					<tbody>
						{[...hbs].reverse().slice(0, 40).map((h) => (
							<tr key={h.id} className="border-b border-border/60 last:border-0">
								<td className="px-4 py-2 font-mono text-[10.5px] text-text-faint">
									{new Date(h.checkedAt ?? "").toLocaleString()}
								</td>
								<td className="px-4 py-2">
									<span className={cn("font-mono text-[10.5px] uppercase", h.status === "up" ? "text-up" : "text-down")}>
										{h.status}
									</span>
								</td>
								<td className="px-4 py-2 font-mono text-[11px] text-text-dim">{h.pingMs ? `${h.pingMs}ms` : "—"}</td>
								<td className="px-4 py-2 font-mono text-[11px] text-text-faint">{h.msg}</td>
							</tr>
						))}
						{hbs.length === 0 && (
							<tr><td className="px-4 py-6 text-center text-text-faint">{t("monitors.noChecks")}</td></tr>
						)}
					</tbody>
				</table>
			</div>

			<MonitorDialog open={dlgOpen} onOpenChange={setDlgOpen} monitor={m} onSaved={load} />
		</main>
	);
}

/** UptimeBar renders the check history as a strip of status-colored bars —
 * the window is split into fixed buckets; a bucket is "down" if any check in
 * it failed, so outages stay visible at long ranges. */
function UptimeBar({ hbs, hours }: { hbs: Heartbeat[]; hours: number }) {
	const BUCKETS = 90;
	const now = Date.now();
	const span = hours * 3600_000;
	const start = now - span;
	const buckets: { status: "up" | "down" | "empty"; from: number; to: number; n: number }[] = [];
	for (let i = 0; i < BUCKETS; i++) {
		buckets.push({ status: "empty", from: start + (i * span) / BUCKETS, to: start + ((i + 1) * span) / BUCKETS, n: 0 });
	}
	for (const h of hbs) {
		const t = new Date(h.checkedAt ?? "").getTime();
		if (Number.isNaN(t) || t < start) continue;
		const i = Math.min(BUCKETS - 1, Math.floor(((t - start) / span) * BUCKETS));
		const b = buckets[i];
		if (!b) continue;
		b.n++;
		if (h.status === "down") b.status = "down";
		else if (b.status === "empty") b.status = "up";
	}
	const fmt = (t: number) => new Date(t).toLocaleString([], { month: "short", day: "numeric", hour: "2-digit", minute: "2-digit" });
	return (
		<div className="flex h-7 items-stretch gap-[2px]">
			{buckets.map((b, i) => (
				<div
					key={i}
					title={`${fmt(b.from)} – ${fmt(b.to)} · ${b.n === 0 ? t("monitors.noChecksBucket") : t("monitors.bucketChecks", { status: b.status, n: b.n })}`}
					className={cn(
						"flex-1 rounded-[2px]",
						b.status === "down" ? "bg-down" : b.status === "up" ? "bg-up/80 hover:bg-up" : "bg-border/50",
					)}
				/>
			))}
		</div>
	);
}
