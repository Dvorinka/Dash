import { useCallback, useEffect, useState } from "react";
import { Link } from "wouter";
import { Pause, Play, Plus, RefreshCw, Trash2 } from "lucide-react";
import { api } from "@/api";
import type { DomainView } from "@/types";
import { DomainDialog } from "@/components/DomainDialog";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

function daysLabel(days: number | null | undefined, warn?: boolean) {
	if (days === null || days === undefined) return "—";
	const cls = warn || days < 0 ? "text-down" : days <= 30 ? "text-amber-500" : "text-text-dim";
	return <span className={cn("font-mono text-[11px]", cls)}>{days < 0 ? "expired" : `${days}d`}</span>;
}

export function DomainsPage() {
	const [domains, setDomains] = useState<DomainView[]>([]);
	const [loaded, setLoaded] = useState(false);
	const [dlgOpen, setDlgOpen] = useState(false);
	const [editing, setEditing] = useState<DomainView | undefined>();
	const [refreshing, setRefreshing] = useState<string | null>(null);

	const load = useCallback(async () => {
		const { data } = await api.GET("/api/domains");
		if (data) setDomains(data);
		setLoaded(true);
	}, []);

	useEffect(() => { void load(); }, [load]);

	async function act(id: string | undefined, fn: () => Promise<unknown>) {
		if (!id) return;
		await fn();
		await load();
	}

	return (
		<main className="mx-auto w-full max-w-5xl px-7 py-8">
			<div className="mb-5 flex items-center justify-between">
				<div>
					<h1 className="text-[15px] font-semibold tracking-tight">Domains</h1>
					<p className="text-[12px] text-text-faint">WHOIS, DNS, and certificate watch.</p>
				</div>
				<Button onClick={() => { setEditing(undefined); setDlgOpen(true); }}>
					<Plus size={13} strokeWidth={2.2} /> Track domain
				</Button>
			</div>

			{loaded && domains.length === 0 ? (
				<div className="rounded-[10px] border border-dashed border-border px-6 py-14 text-center">
					<p className="text-[13px] text-text-dim">No domains tracked yet.</p>
					<p className="mt-1 text-[11.5px] text-text-faint">
						Expiry, registrar, DNS records, SSL certificates, and provider detection.
					</p>
				</div>
			) : (
				<div className="overflow-hidden rounded-[10px] border border-border">
					<table className="w-full text-[13px]">
						<thead>
							<tr className="border-b border-border bg-surface text-left font-mono text-[10px] uppercase tracking-[0.1em] text-text-faint">
								<th className="px-4 py-2.5 font-medium">Domain</th>
								<th className="px-4 py-2.5 font-medium">Registrar</th>
								<th className="px-4 py-2.5 font-medium">Expires</th>
								<th className="px-4 py-2.5 font-medium">SSL</th>
								<th className="px-4 py-2.5 font-medium">Providers</th>
								<th className="px-4 py-2.5" />
							</tr>
						</thead>
						<tbody>
							{domains.map((d) => (
								<tr key={d.id} className={cn("border-b border-border/60 last:border-0 hover:bg-surface-hover/60", !d.active && "opacity-50")}>
									<td className="px-4 py-3">
										<span className="flex items-center gap-2.5 font-medium">
											{d.faviconUrl ? (
												<img src={d.faviconUrl} alt="" className="size-4 rounded-sm" loading="lazy" />
											) : null}
											<Link href={`/domains/${d.id}`} className="hover:underline">{d.name}</Link>
											{d.lookupError ? (
												<span className="font-mono text-[10px] text-down" title={d.lookupError}>lookup error</span>
											) : null}
										</span>
									</td>
									<td className="px-4 py-3 text-[11.5px] text-text-dim">{d.registrarName || "—"}</td>
									<td className="px-4 py-3">{daysLabel(d.daysUntilExpiry, d.expiring)}</td>
									<td className="px-4 py-3">{daysLabel(d.sslDaysUntilExpiry, d.sslExpiring)}</td>
									<td className="px-4 py-3 font-mono text-[10.5px] text-text-faint">
										{[d.dnsProvider, d.hostingProvider].filter(Boolean).join(" · ") || "—"}
									</td>
									<td className="px-4 py-3">
										<div className="flex items-center justify-end gap-1">
											<Button variant="outline" size="icon" aria-label="Refresh now" disabled={refreshing === d.id}
												onClick={() => {
													if (!d.id) return;
													setRefreshing(d.id);
													void act(d.id, () => api.POST("/api/domains/{id}/refresh", { params: { path: { id: d.id } } }))
														.finally(() => setRefreshing(null));
												}}>
												<RefreshCw size={12} className={cn(refreshing === d.id && "animate-spin")} />
											</Button>
											<Button variant="outline" size="icon" aria-label={d.active ? "Pause" : "Resume"}
												onClick={() => void act(d.id, () => api.PATCH("/api/domains/{id}", { params: { path: { id: d.id! } }, body: { active: !d.active } }))}>
												{d.active ? <Pause size={12} /> : <Play size={12} />}
											</Button>
											<Button variant="outline" size="icon" aria-label="Delete"
												onClick={() => {
													if (confirm(`Stop tracking "${d.name}"?`)) {
														void act(d.id, () => api.DELETE("/api/domains/{id}", { params: { path: { id: d.id! } } }));
													}
												}}>
												<Trash2 size={12} />
											</Button>
											<Button variant="outline" size="sm" onClick={() => { setEditing(d); setDlgOpen(true); }}>
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

			<DomainDialog open={dlgOpen} onOpenChange={setDlgOpen} domain={editing} onSaved={load} />
		</main>
	);
}
