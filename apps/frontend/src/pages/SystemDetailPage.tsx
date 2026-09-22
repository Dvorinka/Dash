import { useCallback, useEffect, useState } from "react";
import { Link } from "wouter";
import { ArrowLeft, Copy, Trash2 } from "lucide-react";
import { api } from "@/api";
import { ChartCard, MetricChart, chartColor, type ChartRow, type Series } from "@/components/chart";
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
	const spanMs = hours * 3600_000;

	// Rows carry fixed keys plus dynamic `c:`/`m:` (containers), `s:` (sensors),
	// `g:`/`v:` (GPU util/VRAM) keys; series pick them up by name.
	const rows: ChartRow[] = stats.map((s) => {
		const r: ChartRow = {
			t: new Date(s.ts ?? "").getTime(),
			cpu: s.cpu,
			mem: s.memUsed,
			swap: s.swapUsed,
			disk: s.diskUsed,
			rx: s.netRx,
			tx: s.netTx,
			l1: s.load1,
			l5: s.load5,
			l15: s.load15,
		};
		for (const c of s.containers ?? []) {
			if (c.name) {
				r[`c:${c.name}`] = c.cpu ?? 0;
				r[`m:${c.name}`] = c.memUsed ?? 0;
			}
		}
		for (const [k, v] of Object.entries(s.temps ?? {})) r[`s:${k}`] = v;
		for (const g of s.gpu ?? []) {
			if (g.name) {
				r[`g:${g.name}`] = g.utilPct;
				r[`v:${g.name}`] = g.memUsed;
			}
		}
		return r;
	});

	const names = (ns: (string | undefined)[]) =>
		[...new Set(ns.filter((n): n is string => n !== undefined))];
	const containerNames = names(stats.flatMap((s) => (s.containers ?? []).map((c) => c.name)));
	const tempNames = names(stats.flatMap((s) => Object.keys(s.temps ?? {})));
	const gpuNames = names(stats.flatMap((s) => (s.gpu ?? []).map((g) => g.name)));
	// Zero-fill absent containers so stacked areas stay flush over restarts.
	for (const r of rows) {
		for (const n of containerNames) {
			r[`c:${n}`] ??= 0;
			r[`m:${n}`] ??= 0;
		}
	}

	const pct = (v: number) => `${v.toFixed(1)}%`;
	const cpuSeries: Series[] = [{ key: "cpu", label: "cpu", color: chartColor(0) }];
	const dockerCpuSeries: Series[] = containerNames.map((n, i) => ({ key: `c:${n}`, label: n, color: chartColor(i), stack: "a" }));
	const memSeries: Series[] = [
		{ key: "mem", label: t("systems.chartUsed"), color: chartColor(1) },
		{ key: "swap", label: "swap", color: chartColor(3) },
	];
	const dockerMemSeries: Series[] = containerNames.map((n, i) => ({ key: `m:${n}`, label: n, color: chartColor(i), stack: "a" }));
	const diskSeries: Series[] = [{ key: "disk", label: t("systems.chartUsed"), color: chartColor(2) }];
	const netSeries: Series[] = [
		{ key: "rx", label: t("systems.chartRx"), color: chartColor(4) },
		{ key: "tx", label: t("systems.chartTx"), color: chartColor(5) },
	];
	const loadSeries: Series[] = [
		{ key: "l1", label: "1 min", color: chartColor(0) },
		{ key: "l5", label: "5 min", color: chartColor(3) },
		{ key: "l15", label: "15 min", color: chartColor(5) },
	];
	const tempSeries: Series[] = tempNames.map((n, i) => ({ key: `s:${n}`, label: n, color: chartColor(i), line: true }));
	const gpuSeries: Series[] = gpuNames.map((n, i) => ({ key: `g:${n}`, label: n, color: chartColor(i) }));
	const vramSeries: Series[] = gpuNames.map((n, i) => ({ key: `v:${n}`, label: n, color: chartColor(i) }));

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
				<ChartCard title={t("systems.chartCpu")} hint={t("systems.chartCpuHint")}>
					<MetricChart rows={rows} series={cpuSeries} fmt={pct} domain={[0, 100]} spanMs={spanMs} />
				</ChartCard>
				{dockerCpuSeries.length > 0 && (
					<ChartCard title={t("systems.chartDockerCpu")} hint={t("systems.chartDockerCpuHint")} series={dockerCpuSeries}>
						<MetricChart rows={rows} series={dockerCpuSeries} fmt={pct} spanMs={spanMs} />
					</ChartCard>
				)}
				<ChartCard title={t("systems.chartMem")} hint={t("systems.chartMemHint")} series={memSeries}>
					<MetricChart rows={rows} series={memSeries} fmt={fmtBytes} spanMs={spanMs} />
				</ChartCard>
				{dockerMemSeries.length > 0 && (
					<ChartCard title={t("systems.chartDockerMem")} hint={t("systems.chartDockerMemHint")} series={dockerMemSeries}>
						<MetricChart rows={rows} series={dockerMemSeries} fmt={fmtBytes} spanMs={spanMs} />
					</ChartCard>
				)}
				<ChartCard title={t("systems.chartDisk")} hint={t("systems.chartDiskHint")}>
					<MetricChart rows={rows} series={diskSeries} fmt={fmtBytes} domain={l?.diskTotal ? [0, l.diskTotal] : undefined} spanMs={spanMs} />
				</ChartCard>
				<ChartCard title={t("systems.chartNet")} hint={t("systems.chartNetHint")} series={netSeries}>
					<MetricChart rows={rows} series={netSeries} fmt={(v) => `${fmtBytes(v)}/s`} spanMs={spanMs} />
				</ChartCard>
				<ChartCard title={t("systems.chartLoad")} hint={t("systems.chartLoadHint")} series={loadSeries}>
					<MetricChart rows={rows} series={loadSeries} fmt={(v) => v.toFixed(2)} spanMs={spanMs} />
				</ChartCard>
				{tempSeries.length > 0 && (
					<ChartCard title={t("systems.chartTemp")} hint={t("systems.chartTempHint")} series={tempSeries}>
						<MetricChart rows={rows} series={tempSeries} fmt={(v) => `${v.toFixed(0)}°C`} spanMs={spanMs} />
					</ChartCard>
				)}
				{gpuSeries.length > 0 && (
					<ChartCard title={t("systems.chartGpu")} hint={t("systems.chartGpuHint")} series={gpuSeries}>
						<MetricChart rows={rows} series={gpuSeries} fmt={pct} domain={[0, 100]} spanMs={spanMs} />
					</ChartCard>
				)}
				{vramSeries.length > 0 && (
					<ChartCard title={t("systems.chartVram")} hint={t("systems.chartVramHint")} series={vramSeries}>
						<MetricChart rows={rows} series={vramSeries} fmt={fmtBytes} spanMs={spanMs} />
					</ChartCard>
				)}
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
