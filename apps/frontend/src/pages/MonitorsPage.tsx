import { useCallback, useEffect, useState } from "react";
import { Link } from "wouter";
import { Pause, Play, Plus, RefreshCw, Trash2 } from "lucide-react";
import { api } from "@/api";
import type { MonitorView } from "@/types";
import { MonitorDialog } from "@/components/MonitorDialog";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

function StatusDot({ status }: { status?: string }) {
	return (
		<span
			className={cn(
				"size-1.5 shrink-0 rounded-full",
				status === "up" ? "bg-up" : status === "down" ? "bg-down" : "bg-text-faint/40",
			)}
		/>
	);
}

export function MonitorsPage() {
	const [monitors, setMonitors] = useState<MonitorView[]>([]);
	const [loaded, setLoaded] = useState(false);
	const [dlgOpen, setDlgOpen] = useState(false);
	const [editing, setEditing] = useState<MonitorView | undefined>();

	const load = useCallback(async () => {
		const { data } = await api.GET("/api/monitors");
		if (data) setMonitors(data);
		setLoaded(true);
	}, []);

	useEffect(() => {
		void load();
		const t = setInterval(() => void load(), 30_000);
		return () => clearInterval(t);
	}, [load]);

	async function act(id: string | undefined, fn: () => Promise<unknown>) {
		if (!id) return;
		await fn();
		await load();
	}

	return (
		<main className="mx-auto w-full max-w-5xl px-7 py-8">
			<div className="mb-5 flex items-center justify-between">
				<div>
					<h1 className="text-[15px] font-semibold tracking-tight">Monitors</h1>
					<p className="text-[12px] text-text-faint">Uptime checks on a schedule.</p>
				</div>
				<Button onClick={() => { setEditing(undefined); setDlgOpen(true); }}>
					<Plus size={13} strokeWidth={2.2} /> New monitor
				</Button>
			</div>

			{loaded && monitors.length === 0 ? (
				<div className="rounded-[10px] border border-dashed border-border px-6 py-14 text-center">
					<p className="text-[13px] text-text-dim">No monitors yet.</p>
					<p className="mt-1 text-[11.5px] text-text-faint">
						HTTP, TCP, ping, DNS, keyword, JSON path, or push — checks run server-side on your interval.
					</p>
				</div>
			) : (
				<div className="overflow-hidden rounded-[10px] border border-border">
					<table className="w-full text-[13px]">
						<thead>
							<tr className="border-b border-border bg-surface text-left font-mono text-[10px] uppercase tracking-[0.1em] text-text-faint">
								<th className="px-4 py-2.5 font-medium">Status</th>
								<th className="px-4 py-2.5 font-medium">Name</th>
								<th className="px-4 py-2.5 font-medium">Type</th>
								<th className="px-4 py-2.5 font-medium">Uptime 24h</th>
								<th className="px-4 py-2.5 font-medium">Uptime 30d</th>
								<th className="px-4 py-2.5 font-medium">Ping</th>
								<th className="px-4 py-2.5 font-medium">Every</th>
								<th className="px-4 py-2.5" />
							</tr>
						</thead>
						<tbody>
							{monitors.map((m) => (
								<tr key={m.id} className="border-b border-border/60 last:border-0 hover:bg-surface-hover/60">
									<td className="px-4 py-3">
										<span className="flex items-center gap-2 font-mono text-[10.5px] uppercase tracking-[0.06em] text-text-faint">
											<StatusDot status={m.status} />
											{m.status}
										</span>
									</td>
									<td className="px-4 py-3 font-medium">
										<Link href={`/monitors/${m.id}`} className="hover:underline">{m.name}</Link>
									</td>
									<td className="px-4 py-3 font-mono text-[11px] text-text-dim">{m.type}</td>
									<td className="px-4 py-3 font-mono text-[11px] text-text-dim">
										{m.stats?.uptime24h?.toFixed(1)}%
									</td>
									<td className="px-4 py-3 font-mono text-[11px] text-text-dim">
										{m.stats?.uptime30d?.toFixed(1)}%
									</td>
									<td className="px-4 py-3 font-mono text-[11px] text-text-dim">
										{m.stats?.avgPing24h ? `${m.stats.avgPing24h}ms` : "—"}
									</td>
									<td className="px-4 py-3 font-mono text-[11px] text-text-dim">{m.intervalS}s</td>
									<td className="px-4 py-3">
										<div className="flex items-center justify-end gap-1">
											<Button variant="outline" size="icon" aria-label="Check now"
												onClick={() => void act(m.id, () => api.POST("/api/monitors/{id}/check", { params: { path: { id: m.id! } } }))}>
												<RefreshCw size={12} />
											</Button>
											<Button variant="outline" size="icon" aria-label={m.active ? "Pause" : "Resume"}
												onClick={() => void act(m.id, () => api.PATCH("/api/monitors/{id}", { params: { path: { id: m.id! } }, body: { active: !m.active } }))}>
												{m.active ? <Pause size={12} /> : <Play size={12} />}
											</Button>
											<Button variant="outline" size="icon" aria-label="Delete"
												onClick={() => {
													if (confirm(`Delete monitor "${m.name}"?`)) {
														void act(m.id, () => api.DELETE("/api/monitors/{id}", { params: { path: { id: m.id! } } }));
													}
												}}>
												<Trash2 size={12} />
											</Button>
											<Button variant="outline" size="sm" onClick={() => { setEditing(m); setDlgOpen(true); }}>
												Edit
											</Button>
										</div>
									</td>
								</tr>
							))}
						</tbody>
					</table>
				</div>
			)}

			<MonitorDialog open={dlgOpen} onOpenChange={setDlgOpen} monitor={editing} onSaved={load} />
		</main>
	);
}
