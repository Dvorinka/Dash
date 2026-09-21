import { useCallback, useEffect, useState } from "react";
import { Link } from "wouter";
import { ArrowLeft, Pause, Play, RefreshCw, Trash2 } from "lucide-react";
import { Area, AreaChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
import { api } from "@/api";
import type { Heartbeat, MonitorView } from "@/types";
import { MonitorDialog } from "@/components/MonitorDialog";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

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

	if (!m) return <main className="mx-auto max-w-5xl px-7 py-8 text-[13px] text-text-faint">Loading…</main>;

	const chart = hbs.map((h) => ({
		t: new Date(h.checkedAt ?? "").toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }),
		ping: h.pingMs,
		up: h.status === "up",
	}));

	return (
		<main className="mx-auto w-full max-w-5xl px-7 py-8">
			<Link href="/monitors" className="mb-4 inline-flex items-center gap-1.5 text-[12px] text-text-faint hover:text-text">
				<ArrowLeft size={12} /> Monitors
			</Link>

			<div className="mb-6 flex items-start justify-between">
				<div>
					<div className="flex items-center gap-2.5">
						<span className={cn("size-2 rounded-full", m.status === "up" ? "bg-up" : m.status === "down" ? "bg-down" : "bg-text-faint/40")} />
						<h1 className="text-[17px] font-semibold tracking-tight">{m.name}</h1>
						<span className="font-mono text-[10.5px] uppercase tracking-[0.08em] text-text-faint">{m.status}</span>
					</div>
					<p className="mt-1 font-mono text-[11.5px] text-text-faint">
						{m.type} · {m.url || m.hostname}{m.port ? `:${m.port}` : ""} · every {m.intervalS}s
					</p>
				</div>
				<div className="flex gap-2">
					<Button variant="outline" size="sm" onClick={() => void api.POST("/api/monitors/{id}/check", { params: { path: { id } } }).then(load)}>
						<RefreshCw size={12} /> Check now
					</Button>
					<Button variant="outline" size="sm"
						onClick={() => void api.PATCH("/api/monitors/{id}", { params: { path: { id } }, body: { active: !m.active } }).then(load)}>
						{m.active ? <Pause size={12} /> : <Play size={12} />} {m.active ? "Pause" : "Resume"}
					</Button>
					<Button variant="outline" size="sm" onClick={() => setDlgOpen(true)}>Edit</Button>
					<Button variant="outline" size="sm" aria-label="Delete"
						onClick={() => {
							if (confirm(`Delete monitor "${m.name}"?`)) {
								void api.DELETE("/api/monitors/{id}", { params: { path: { id } } }).then(() => { location.href = "/monitors"; });
							}
						}}>
						<Trash2 size={12} />
					</Button>
				</div>
			</div>

			<div className="mb-6 grid grid-cols-4 gap-3 max-[720px]:grid-cols-2">
				{[
					["Uptime 24h", `${m.stats?.uptime24h?.toFixed(2)}%`],
					["Uptime 30d", `${m.stats?.uptime30d?.toFixed(2)}%`],
					["Avg ping 24h", m.stats?.avgPing24h ? `${m.stats.avgPing24h} ms` : "—"],
					["Checks", String((m.stats?.up ?? 0) + (m.stats?.down ?? 0))],
				].map(([k, v]) => (
					<div key={k} className="rounded-[10px] border border-border bg-surface px-4 py-3">
						<div className="font-mono text-[10px] uppercase tracking-[0.1em] text-text-faint">{k}</div>
						<div className="mt-1 font-mono text-[17px] font-medium">{v}</div>
					</div>
				))}
			</div>

			<div className="mb-2 flex items-center justify-between">
				<h2 className="font-mono text-[10.5px] uppercase tracking-[0.12em] text-text-faint">Response time</h2>
				<div className="flex gap-1">
					{[24, 168, 720].map((h) => (
						<button key={h} type="button" onClick={() => setHours(h)}
							className={cn("rounded-[6px] px-2 py-1 font-mono text-[10.5px]",
								hours === h ? "bg-surface-hover text-text" : "text-text-faint hover:text-text")}>
							{h === 24 ? "24h" : h === 168 ? "7d" : "30d"}
						</button>
					))}
				</div>
			</div>
			<div className="mb-6 h-56 rounded-[10px] border border-border bg-surface p-3">
				<ResponsiveContainer width="100%" height="100%">
					<AreaChart data={chart} margin={{ top: 4, right: 4, bottom: 0, left: -18 }}>
						<CartesianGrid stroke="var(--border)" strokeDasharray="3 3" vertical={false} />
						<XAxis dataKey="t" tick={{ fontSize: 10, fill: "var(--text-faint)" }} tickLine={false} axisLine={false} minTickGap={40} />
						<YAxis tick={{ fontSize: 10, fill: "var(--text-faint)" }} tickLine={false} axisLine={false} unit=" ms" width={62} />
						<Tooltip
							contentStyle={{ background: "var(--surface)", border: "1px solid var(--border)", borderRadius: 8, fontSize: 11 }}
							labelStyle={{ color: "var(--text-faint)" }}
						/>
						<Area type="monotone" dataKey="ping" stroke="var(--up)" fill="var(--up)" fillOpacity={0.12} strokeWidth={1.5} dot={false} isAnimationActive={false} />
					</AreaChart>
				</ResponsiveContainer>
			</div>

			<h2 className="mb-2 font-mono text-[10.5px] uppercase tracking-[0.12em] text-text-faint">Recent checks</h2>
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
							<tr><td className="px-4 py-6 text-center text-text-faint">No checks in range.</td></tr>
						)}
					</tbody>
				</table>
			</div>

			<MonitorDialog open={dlgOpen} onOpenChange={setDlgOpen} monitor={m} onSaved={load} />
		</main>
	);
}
