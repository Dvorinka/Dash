import { useCallback, useEffect, useState } from "react";
import { Link } from "wouter";
import { ArrowLeft, Copy, Trash2 } from "lucide-react";
import { Area, AreaChart, CartesianGrid, Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
import { api } from "@/api";
import type { System, SystemStat } from "@/types";
import { agentCmd, fmtBytes, fmtUptime } from "@/pages/SystemsPage";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { t } from "@/i18n";

export function SystemDetailPage({ id }: { id: string }) {
	const [sys, setSys] = useState<System | null>(null);
	const [stats, setStats] = useState<SystemStat[]>([]);
	const [hours, setHours] = useState(24);
	const [notFound, setNotFound] = useState(false);

	const load = useCallback(async () => {
		const [{ data: s, response }, { data: st }] = await Promise.all([
			api.GET("/api/systems/{id}", { params: { path: { id } } }),
			api.GET("/api/systems/{id}/stats", { params: { path: { id }, query: { hours } } }),
		]);
		if (response.status === 404) setNotFound(true);
		if (s) setSys(s);
		if (st) setStats(st);
	}, [id, hours]);

	useEffect(() => {
		void load();
		const t = setInterval(() => void load(), 15_000);
		return () => clearInterval(t);
	}, [load]);

	if (notFound) return (
		<main className="mx-auto max-w-5xl px-7 py-8 text-[13px] text-text-faint">
			<Link href="/systems" className="inline-flex items-center gap-1.5 hover:text-text"><ArrowLeft size={12} /> {t("nav.systems")}</Link>
			<p className="mt-4">{t("systems.notFound")}</p>
		</main>
	);
	if (!sys) return <main className="mx-auto max-w-5xl px-7 py-8 text-[13px] text-text-faint">{t("common.loading")}</main>;

	const l = sys.latest;
	const chart = stats.map((s) => ({
		t: new Date(s.ts ?? "").toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }),
		cpu: s.cpu,
		memPct: s.memTotal ? ((s.memUsed ?? 0) / s.memTotal) * 100 : undefined,
		rx: (s.netRx ?? 0) / 1024,
		tx: (s.netTx ?? 0) / 1024,
		load: s.load1,
	}));
	const axisTick = { fontSize: 10, fill: "var(--text-faint)" };
	const tipStyle = { background: "var(--surface)", border: "1px solid var(--border)", borderRadius: 8, fontSize: 11 };

	return (
		<main className="mx-auto w-full max-w-5xl px-7 py-8">
			<Link href="/systems" className="mb-4 inline-flex items-center gap-1.5 text-[12px] text-text-faint hover:text-text">
				<ArrowLeft size={12} /> {t("nav.systems")}
			</Link>

			<div className="mb-6 flex items-start justify-between">
				<div>
					<div className="flex items-center gap-2.5">
						<span className={cn("size-2 rounded-full",
							sys.status === "up" ? "bg-up" : sys.status === "down" ? "bg-down" : "bg-text-faint/40")} />
						<h1 className="text-[17px] font-semibold tracking-tight">{sys.name}</h1>
						<span className="font-mono text-[10.5px] uppercase tracking-[0.08em] text-text-faint">{sys.status}</span>
					</div>
					<p className="mt-1 font-mono text-[11.5px] text-text-faint">
						{sys.host ? `${sys.host} · ` : ""}{sys.os}/{sys.arch}
						{sys.cpuModel ? ` · ${sys.cpuModel}` : ""}{sys.cores ? ` · ${t("systems.cores", { n: sys.cores })}` : ""}
						{sys.lastSeenAt ? ` · ${t("systems.seenAt", { time: new Date(sys.lastSeenAt).toLocaleTimeString() })}` :  ` · ${t("systems.neverSeen")}`}
					</p>
				</div>
				<div className="flex gap-2">
					<Button variant="outline" size="sm" aria-label={t("common.delete")}
						onClick={() => {
							if (confirm(t("systems.deleteConfirm", { name: sys.name ?? "" }))) {
								void api.DELETE("/api/systems/{id}", { params: { path: { id } } }).then(() => { location.href = "/systems"; });
							}
						}}>
						<Trash2 size={12} />
					</Button>
				</div>
			</div>

			<div className="mb-6 grid grid-cols-4 gap-3 max-[720px]:grid-cols-2">
				{[
					[t("systems.cpu"), l?.cpu !== undefined ? `${l.cpu.toFixed(1)}%` : "—"],
					[t("systems.mem"), l?.memTotal ? `${fmtBytes(l.memUsed ?? 0)} / ${fmtBytes(l.memTotal)}` : "—"],
					[t("systems.disk") + " /", l?.diskTotal ? `${fmtBytes(l.diskUsed ?? 0)} / ${fmtBytes(l.diskTotal)}` : "—"],
					[t("systems.uptime"), l?.uptimeS ? fmtUptime(l.uptimeS) : "—"],
				].map(([k, v]) => (
					<div key={k} className="rounded-[10px] border border-border bg-surface px-4 py-3">
						<div className="font-mono text-[10px] uppercase tracking-[0.1em] text-text-faint">{k}</div>
						<div className="mt-1 font-mono text-[15px] font-medium">{v}</div>
					</div>
				))}
			</div>

			<div className="mb-2 flex items-center justify-between">
				<h2 className="font-mono text-[10.5px] uppercase tracking-[0.12em] text-text-faint">{t("systems.usage")}</h2>
				<div className="flex gap-1">
					{[1, 6, 24, 168].map((h) => (
						<button key={h} type="button" onClick={() => setHours(h)}
							className={cn("rounded-[6px] px-2 py-1 font-mono text-[10.5px]",
								hours === h ? "bg-surface-hover text-text" : "text-text-faint hover:text-text")}>
							{h === 1 ? "1h" : h === 6 ? "6h" : h === 24 ? "24h" : "7d"}
						</button>
					))}
				</div>
			</div>

			<div className="mb-6 grid grid-cols-2 gap-3 max-[720px]:grid-cols-1">
				<div className="h-44 rounded-[10px] border border-border bg-surface p-3">
					<div className="mb-1 font-mono text-[9.5px] uppercase tracking-[0.1em] text-text-faint">{t("systems.cpuPct")}</div>
					<ResponsiveContainer width="100%" height="85%">
						<AreaChart data={chart} margin={{ top: 4, right: 4, bottom: 0, left: -22 }}>
							<CartesianGrid stroke="var(--border)" strokeDasharray="3 3" vertical={false} />
							<XAxis dataKey="t" tick={axisTick} tickLine={false} axisLine={false} minTickGap={40} />
							<YAxis tick={axisTick} tickLine={false} axisLine={false} domain={[0, 100]} unit="%" width={48} />
							<Tooltip contentStyle={tipStyle} labelStyle={{ color: "var(--text-faint)" }} />
							<Area type="monotone" dataKey="cpu" stroke="var(--up)" fill="var(--up)" fillOpacity={0.12} strokeWidth={1.5} dot={false} isAnimationActive={false} />
						</AreaChart>
					</ResponsiveContainer>
				</div>
				<div className="h-44 rounded-[10px] border border-border bg-surface p-3">
					<div className="mb-1 font-mono text-[9.5px] uppercase tracking-[0.1em] text-text-faint">{t("systems.memPct")}</div>
					<ResponsiveContainer width="100%" height="85%">
						<AreaChart data={chart} margin={{ top: 4, right: 4, bottom: 0, left: -22 }}>
							<CartesianGrid stroke="var(--border)" strokeDasharray="3 3" vertical={false} />
							<XAxis dataKey="t" tick={axisTick} tickLine={false} axisLine={false} minTickGap={40} />
							<YAxis tick={axisTick} tickLine={false} axisLine={false} domain={[0, 100]} unit="%" width={48} />
							<Tooltip contentStyle={tipStyle} labelStyle={{ color: "var(--text-faint)" }} />
							<Area type="monotone" dataKey="memPct" stroke="var(--up)" fill="var(--up)" fillOpacity={0.12} strokeWidth={1.5} dot={false} isAnimationActive={false} />
						</AreaChart>
					</ResponsiveContainer>
				</div>
				<div className="h-44 rounded-[10px] border border-border bg-surface p-3">
					<div className="mb-1 font-mono text-[9.5px] uppercase tracking-[0.1em] text-text-faint">{t("systems.netKbs")}</div>
					<ResponsiveContainer width="100%" height="85%">
						<LineChart data={chart} margin={{ top: 4, right: 4, bottom: 0, left: -18 }}>
							<CartesianGrid stroke="var(--border)" strokeDasharray="3 3" vertical={false} />
							<XAxis dataKey="t" tick={axisTick} tickLine={false} axisLine={false} minTickGap={40} />
							<YAxis tick={axisTick} tickLine={false} axisLine={false} width={58} />
							<Tooltip contentStyle={tipStyle} labelStyle={{ color: "var(--text-faint)" }} />
							<Line type="monotone" dataKey="rx" name="rx" stroke="var(--up)" strokeWidth={1.5} dot={false} isAnimationActive={false} />
							<Line type="monotone" dataKey="tx" name="tx" stroke="var(--text-faint)" strokeWidth={1.5} dot={false} isAnimationActive={false} />
						</LineChart>
					</ResponsiveContainer>
				</div>
				<div className="h-44 rounded-[10px] border border-border bg-surface p-3">
					<div className="mb-1 font-mono text-[9.5px] uppercase tracking-[0.1em] text-text-faint">{t("systems.load1m")}</div>
					<ResponsiveContainer width="100%" height="85%">
						<AreaChart data={chart} margin={{ top: 4, right: 4, bottom: 0, left: -22 }}>
							<CartesianGrid stroke="var(--border)" strokeDasharray="3 3" vertical={false} />
							<XAxis dataKey="t" tick={axisTick} tickLine={false} axisLine={false} minTickGap={40} />
							<YAxis tick={axisTick} tickLine={false} axisLine={false} width={48} />
							<Tooltip contentStyle={tipStyle} labelStyle={{ color: "var(--text-faint)" }} />
							<Area type="monotone" dataKey="load" stroke="var(--up)" fill="var(--up)" fillOpacity={0.12} strokeWidth={1.5} dot={false} isAnimationActive={false} />
						</AreaChart>
					</ResponsiveContainer>
				</div>
			</div>

			<div className="grid grid-cols-2 gap-3 max-[720px]:grid-cols-1">
				{l?.temps && Object.keys(l.temps).length > 0 && (
					<div className="rounded-[10px] border border-border">
						<h2 className="border-b border-border bg-surface px-4 py-2.5 font-mono text-[10px] uppercase tracking-[0.1em] text-text-faint">
							{t("systems.temperatures")}
						</h2>
						<table className="w-full text-[12px]">
							<tbody>
								{Object.entries(l.temps).map(([k, v]) => (
									<tr key={k} className="border-b border-border/60 last:border-0">
										<td className="px-4 py-2 text-text-dim">{k}</td>
										<td className="px-4 py-2 text-right font-mono text-[11px]">{v.toFixed(1)}°C</td>
									</tr>
								))}
							</tbody>
						</table>
					</div>
				)}
				{l?.containers && l.containers.length > 0 && (
					<div className="rounded-[10px] border border-border">
						<h2 className="border-b border-border bg-surface px-4 py-2.5 font-mono text-[10px] uppercase tracking-[0.1em] text-text-faint">
							{t("systems.containers")}
						</h2>
						<table className="w-full text-[12px]">
							<tbody>
								{l.containers.map((ctr) => (
									<tr key={ctr.name} className="border-b border-border/60 last:border-0">
										<td className="px-4 py-2 font-mono text-[11px] text-text-dim">{ctr.name}</td>
										<td className="px-4 py-2 text-right font-mono text-[11px] text-text-faint">
											{ctr.state === "running" && (ctr.cpu || ctr.memUsed) ? (
												<>
													{(ctr.cpu ?? 0).toFixed(1)}%
													{" · "}
													{fmtBytes(ctr.memUsed ?? 0)}
													{ctr.memLimit ? ` / ${fmtBytes(ctr.memLimit)}` : ""}
												</>
											) : null}
										</td>
										<td className="px-4 py-2 text-right">
											<span className={cn("font-mono text-[10.5px] uppercase",
												ctr.state === "running" ? "text-up" : "text-text-faint")}>
												{ctr.state}
											</span>
										</td>
									</tr>
								))}
							</tbody>
						</table>
					</div>
				)}

				{l?.smart && l.smart.length > 0 && (
					<div className="rounded-[10px] border border-border">
						<h2 className="border-b border-border bg-surface px-4 py-2.5 font-mono text-[10px] uppercase tracking-[0.1em] text-text-faint">
							{t("systems.drivesSmart")}
						</h2>
						<table className="w-full text-[12px]">
							<tbody>
								{l.smart.map((d) => (
									<tr key={d.device} className="border-b border-border/60 last:border-0">
										<td className="px-4 py-2 font-mono text-[11px] text-text-dim">{d.device}</td>
										<td className="px-4 py-2 text-[11px] text-text-dim">{d.model}</td>
										<td className="px-4 py-2 text-right font-mono text-[11px]">
											{d.tempC ? `${d.tempC.toFixed(0)}°C` : ""}
										</td>
										<td className="px-4 py-2 text-right">
											<span className={cn("font-mono text-[10.5px] uppercase",
												d.passed === false ? "text-destructive" : "text-up")}>
												{d.passed === false ? t("systems.failing") : d.passed ? "ok" : "—"}
											</span>
										</td>
									</tr>
								))}
							</tbody>
						</table>
					</div>
				)}

				{l?.zfs && l.zfs.length > 0 && (
					<div className="rounded-[10px] border border-border">
						<h2 className="border-b border-border bg-surface px-4 py-2.5 font-mono text-[10px] uppercase tracking-[0.1em] text-text-faint">
							{t("systems.zfs")}
						</h2>
						<table className="w-full text-[12px]">
							<tbody>
								{l.zfs.map((p) => (
									<tr key={p.name} className="border-b border-border/60 last:border-0">
										<td className="px-4 py-2 font-mono text-[11px] text-text-dim">{p.name}</td>
										<td className="px-4 py-2 font-mono text-[11px] text-text-dim">
											{p.size ? `${fmtBytes((p.size ?? 0) - (p.free ?? 0))} / ${fmtBytes(p.size)}` : ""}
										</td>
										<td className="px-4 py-2 text-right">
											<span className={cn("font-mono text-[10.5px] uppercase",
												p.health === "ONLINE" ? "text-up" : "text-destructive")}>
												{p.health}
											</span>
										</td>
									</tr>
								))}
							</tbody>
						</table>
					</div>
				)}

				{l?.gpu && l.gpu.length > 0 && (
					<div className="rounded-[10px] border border-border">
						<h2 className="border-b border-border bg-surface px-4 py-2.5 font-mono text-[10px] uppercase tracking-[0.1em] text-text-faint">
							GPU
						</h2>
						<table className="w-full text-[12px]">
							<tbody>
								{l.gpu.map((g) => (
									<tr key={g.name} className="border-b border-border/60 last:border-0">
										<td className="px-4 py-2 font-mono text-[11px] text-text-dim">{g.name}</td>
										<td className="px-4 py-2 text-right font-mono text-[11px] text-text-dim">
											{[
												g.utilPct ? `${g.utilPct.toFixed(0)}%` : "",
												g.tempC ? `${g.tempC.toFixed(0)}°C` : "",
												g.memTotal ? `${fmtBytes(g.memUsed ?? 0)} / ${fmtBytes(g.memTotal)}` : "",
											].filter(Boolean).join(" · ")}
										</td>
									</tr>
								))}
							</tbody>
						</table>
					</div>
				)}
			</div>

			<div className="mt-6 rounded-[10px] border border-border bg-surface px-4 py-3">
				<div className="mb-1.5 font-mono text-[10px] uppercase tracking-[0.1em] text-text-faint">{t("systems.agentCmd")}</div>
				<div className="flex items-center gap-2">
					<code className="flex-1 overflow-x-auto whitespace-nowrap font-mono text-[11px] text-text-dim">
						{agentCmd(sys)}
					</code>
					<Button variant="outline" size="icon" aria-label={t("systems.copy")}
						onClick={() => void navigator.clipboard.writeText(agentCmd(sys))}>
						<Copy size={12} />
					</Button>
				</div>
			</div>
		</main>
	);
}
