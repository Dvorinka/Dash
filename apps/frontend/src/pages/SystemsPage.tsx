import { useCallback, useEffect, useState } from "react";
import { Link } from "wouter";
import { Copy, Plus, Trash2 } from "lucide-react";
import { api } from "@/api";
import type { System } from "@/types";
import { Button } from "@/components/ui/button";
import {
	Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";
import { t } from "@/i18n";

const TB = 2 ** 40;
const GB = 2 ** 30;
const MB = 2 ** 20;

export function fmtBytes(b: number) {
	if (b >= TB) return `${(b / TB).toFixed(1)} TB`;
	if (b >= GB) return `${(b / GB).toFixed(1)} GB`;
	return `${(b / MB).toFixed(0)} MB`;
}

export function fmtUptime(s: number) {
	const d = Math.floor(s / 86400);
	if (d > 0) return `${d}d ${Math.floor((s % 86400) / 3600)}h`;
	const h = Math.floor(s / 3600);
	if (h > 0) return `${h}h ${Math.floor((s % 3600) / 60)}m`;
	return `${Math.floor(s / 60)}m`;
}

export function Bar({ pct, warnAt = 90 }: { pct: number | undefined; warnAt?: number }) {
	return (
		<span className="h-1.5 w-full overflow-hidden rounded-full bg-surface-hover">
			<span
				className={cn("block h-full rounded-full", pct !== undefined && pct > warnAt ? "bg-down" : "bg-up")}
				style={{ width: `${Math.min(pct ?? 0, 100)}%` }}
			/>
		</span>
	);
}

/** The one-liner agents run; shown on create and on the detail page. */
export function agentCmd(sys: System) {
	const base = location.origin;
	return `DASH_URL=${base} DASH_TOKEN=${sys.token} dash-agent`;
}

export function SystemsPage() {
	const [systems, setSystems] = useState<System[]>([]);
	const [loaded, setLoaded] = useState(false);
	const [dlgOpen, setDlgOpen] = useState(false);
	const [name, setName] = useState("");
	const [busy, setBusy] = useState(false);
	const [created, setCreated] = useState<System | null>(null);

	const load = useCallback(async () => {
		const { data } = await api.GET("/api/systems");
		if (data) setSystems(data);
		setLoaded(true);
	}, []);

	useEffect(() => {
		void load();
		const t = setInterval(() => void load(), 15_000);
		return () => clearInterval(t);
	}, [load]);

	async function create() {
		if (!name.trim()) return;
		setBusy(true);
		const { data } = await api.POST("/api/systems", { body: { name: name.trim() } });
		setBusy(false);
		if (data) {
			setCreated(data);
			setName("");
			void load();
		}
	}

	async function remove(id: string | undefined, n: string | undefined) {
		if (!id || !confirm(t("systems.deleteConfirm", { name: n ?? "" }))) return;
		await api.DELETE("/api/systems/{id}", { params: { path: { id } } });
		await load();
	}

	return (
		<main className="mx-auto w-full max-w-5xl px-7 py-8">
			<div className="mb-5 flex items-center justify-between">
				<div>
					<h1 className="text-[15px] font-semibold tracking-tight">{t("systems.title")}</h1>
					<p className="text-[12px] text-text-faint">{t("systems.subtitle")}</p>
				</div>
				<Button onClick={() => { setCreated(null); setDlgOpen(true); }}>
					<Plus size={13} strokeWidth={2.2} /> {t("systems.new")}
				</Button>
			</div>

			{loaded && systems.length === 0 ? (
				<div className="rounded-[10px] border border-dashed border-border px-6 py-14 text-center">
					<p className="text-[13px] text-text-dim">{t("systems.empty")}</p>
					<p className="mt-1 text-[11.5px] text-text-faint">
						{t("systems.emptyHintPre")} <code className="font-mono">dash-agent</code> {t("systems.emptyHintPost")}
					</p>
				</div>
			) : (
				<div className="grid grid-cols-3 gap-3 max-[900px]:grid-cols-2 max-[560px]:grid-cols-1">
					{systems.map((sys) => {
						const l = sys.latest;
						const memPct = l?.memTotal ? ((l.memUsed ?? 0) / l.memTotal) * 100 : undefined;
						const diskPct = l?.diskTotal ? ((l.diskUsed ?? 0) / l.diskTotal) * 100 : undefined;
						const rows: [string, number | undefined, string][] = [
							["cpu", l?.cpu, l?.cpu !== undefined ? `${l.cpu.toFixed(0)}%` : "—"],
							["mem", memPct, l?.memTotal ? `${fmtBytes(l.memUsed ?? 0)} / ${fmtBytes(l.memTotal)}` : "—"],
							["disk", diskPct, l?.diskTotal ? `${fmtBytes(l.diskUsed ?? 0)} / ${fmtBytes(l.diskTotal)}` : "—"],
						];
						return (
							<div key={sys.id} className="rounded-[10px] border border-border bg-surface px-4 py-3.5">
								<div className="mb-2.5 flex items-center gap-2">
									<span className={cn("size-1.5 shrink-0 rounded-full",
										sys.status === "up" ? "bg-up" : sys.status === "down" ? "bg-down" : "bg-text-faint/40")} />
									<Link href={`/systems/${sys.id}`} className="min-w-0 flex-1 truncate text-[13px] font-medium hover:underline">
										{sys.name}
									</Link>
									<Button variant="outline" size="icon" aria-label={t("common.delete")}
										onClick={() => void remove(sys.id, sys.name)}>
										<Trash2 size={11} />
									</Button>
								</div>
								{sys.host ? (
									<p className="mb-2.5 truncate font-mono text-[10.5px] text-text-faint">
										{sys.host} · {sys.cpuModel || t("systems.cores", { n: sys.cores ?? 0 })}
										{l?.uptimeS ? ` · up ${fmtUptime(l.uptimeS)}` : ""}
									</p>
								) : (
									<p className="mb-2.5 font-mono text-[10.5px] text-text-faint">{t("systems.waiting")}</p>
								)}
								<div className="flex flex-col gap-1.5">
									{rows.map(([label, pct, txt]) => (
										<div key={label} className="flex items-center gap-2">
											<span className="w-7 font-mono text-[9.5px] uppercase text-text-faint">{label}</span>
											<span className="flex-1"><Bar pct={pct} /></span>
											<span className="w-24 text-right font-mono text-[10px] text-text-dim">{txt}</span>
										</div>
									))}
								</div>
							</div>
						);
					})}
				</div>
			)}

			<Dialog open={dlgOpen} onOpenChange={setDlgOpen}>
				<DialogContent>
					<DialogHeader>
						<DialogTitle>{t("systems.newTitle")}</DialogTitle>
						<DialogDescription>{t("systems.newDesc")}</DialogDescription>
					</DialogHeader>
					{created ? (
						<div className="flex flex-col gap-3">
							<p className="text-[12px] text-text-dim">
								{t("systems.agentCmdFor")} <span className="font-medium text-text">{created.name}</span>:
							</p>
							<div className="flex items-center gap-2">
								<code className="flex-1 overflow-x-auto whitespace-nowrap rounded-[7px] border border-border bg-bg px-3 py-2 font-mono text-[11px]">
									{agentCmd(created)}
								</code>
								<Button variant="outline" size="icon" aria-label={t("systems.copy")}
									onClick={() => void navigator.clipboard.writeText(agentCmd(created))}>
									<Copy size={12} />
								</Button>
							</div>
							<p className="text-[11px] text-text-faint">
								{t("systems.tokenWarning")}
							</p>
						</div>
					) : (
						<div className="flex flex-col gap-1.5">
							<Label htmlFor="sys-name">{t("common.name")}</Label>
							<Input id="sys-name" value={name} onChange={(e) => setName(e.target.value)}
								placeholder={t("systems.namePlaceholder")} autoFocus
								onKeyDown={(e) => { if (e.key === "Enter") void create(); }} />
						</div>
					)}
					<DialogFooter>
						{created ? (
							<Button onClick={() => setDlgOpen(false)}>{t("systems.done")}</Button>
						) : (
							<>
								<Button variant="outline" onClick={() => setDlgOpen(false)} disabled={busy}>{t("common.cancel")}</Button>
								<Button onClick={() => void create()} disabled={busy || !name.trim()}>
									{busy ? t("common.creating") : t("common.create")}
								</Button>
							</>
						)}
					</DialogFooter>
				</DialogContent>
			</Dialog>
		</main>
	);
}
