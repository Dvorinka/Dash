import { useCallback, useEffect, useState } from "react";
import { Link } from "wouter";
import { ArrowLeft, Pause, Play, RefreshCw, Shield, Trash2 } from "lucide-react";
import { api } from "@/api";
import type { DomainCheck, DomainView } from "@/types";
import { DomainDialog } from "@/components/DomainDialog";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

function Row({ k, v, mono }: { k: string; v: React.ReactNode; mono?: boolean }) {
	if (v === undefined || v === null || v === "") return null;
	return (
		<div className="flex items-baseline justify-between gap-4 border-b border-border/50 px-4 py-2 last:border-0">
			<span className="shrink-0 font-mono text-[10.5px] uppercase tracking-[0.08em] text-text-faint">{k}</span>
			<span className={cn("truncate text-right text-[12px] text-text", mono && "font-mono text-[11px]")}>{v}</span>
		</div>
	);
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
	return (
		<div className="overflow-hidden rounded-[10px] border border-border bg-surface">
			<h2 className="border-b border-border px-4 py-2.5 font-mono text-[10.5px] uppercase tracking-[0.12em] text-text-faint">{title}</h2>
			<div>{children}</div>
		</div>
	);
}

function fmtDate(iso: string | null | undefined) {
	if (!iso) return "—";
	return new Date(iso).toLocaleDateString(undefined, { year: "numeric", month: "short", day: "numeric" });
}

export function DomainDetailPage({ id }: { id: string }) {
	const [d, setD] = useState<DomainView | null>(null);
	const [checks, setChecks] = useState<DomainCheck[]>([]);
	const [dlgOpen, setDlgOpen] = useState(false);
	const [refreshing, setRefreshing] = useState(false);

	const load = useCallback(async () => {
		const [{ data: dv }, { data: ck }] = await Promise.all([
			api.GET("/api/domains/{id}", { params: { path: { id } } }),
			api.GET("/api/domains/{id}/checks", { params: { path: { id } } }),
		]);
		if (dv) setD(dv);
		if (ck) setChecks(ck);
	}, [id]);

	useEffect(() => { void load(); }, [load]);

	if (!d) return <main className="mx-auto max-w-5xl px-7 py-8 text-[13px] text-text-faint">Loading…</main>;

	const expiring = d.expiring || (d.daysUntilExpiry ?? Infinity) < 0;

	return (
		<main className="mx-auto w-full max-w-5xl px-7 py-8">
			<Link href="/domains" className="mb-4 inline-flex items-center gap-1.5 text-[12px] text-text-faint hover:text-text">
				<ArrowLeft size={12} /> Domains
			</Link>

			<div className="mb-6 flex items-start justify-between">
				<div className="flex items-center gap-3">
					{d.faviconUrl ? <img src={d.faviconUrl} alt="" className="size-8 rounded-[7px]" /> : null}
					<div>
						<h1 className="text-[17px] font-semibold tracking-tight">{d.name}</h1>
						<p className="mt-0.5 flex items-center gap-2 font-mono text-[11px] text-text-faint">
							{d.tld ? `.${d.tld}` : ""}
							{d.dnssec === "signed" && <span className="flex items-center gap-1 text-up"><Shield size={10} /> dnssec</span>}
							{d.transferLock ? <span>transfer-locked</span> : null}
							{d.privacyEnabled ? <span>whois privacy</span> : null}
							{d.autoRenew ? <span>auto-renew</span> : null}
						</p>
					</div>
				</div>
				<div className="flex gap-2">
					<Button variant="outline" size="sm" disabled={refreshing}
						onClick={() => {
							setRefreshing(true);
							void api.POST("/api/domains/{id}/refresh", { params: { path: { id } } })
								.then(load).finally(() => setRefreshing(false));
						}}>
						<RefreshCw size={12} className={cn(refreshing && "animate-spin")} /> Refresh
					</Button>
					<Button variant="outline" size="sm"
						onClick={() => void api.PATCH("/api/domains/{id}", { params: { path: { id } }, body: { active: !d.active } }).then(load)}>
						{d.active ? <Pause size={12} /> : <Play size={12} />} {d.active ? "Pause" : "Resume"}
					</Button>
					<Button variant="outline" size="sm" onClick={() => setDlgOpen(true)}>Edit</Button>
					<Button variant="outline" size="sm" aria-label="Delete"
						onClick={() => {
							if (confirm(`Stop tracking "${d.name}"?`)) {
								void api.DELETE("/api/domains/{id}", { params: { path: { id } } }).then(() => { location.href = "/domains"; });
							}
						}}>
						<Trash2 size={12} />
					</Button>
				</div>
			</div>

			{d.lookupError ? (
				<p className="mb-4 rounded-[8px] border border-down/40 bg-down/10 px-3 py-2 font-mono text-[11px] text-down">
					lookup: {d.lookupError}
				</p>
			) : null}

			<div className="mb-6 grid grid-cols-4 gap-3 max-[720px]:grid-cols-2">
				<div className="rounded-[10px] border border-border bg-surface px-4 py-3">
					<div className="font-mono text-[10px] uppercase tracking-[0.1em] text-text-faint">Domain expires</div>
					<div className={cn("mt-1 font-mono text-[17px] font-medium", expiring && "text-down")}>
						{d.daysUntilExpiry !== null && d.daysUntilExpiry !== undefined
							? (d.daysUntilExpiry < 0 ? "expired" : `${d.daysUntilExpiry}d`)
							: "—"}
					</div>
					<div className="font-mono text-[10px] text-text-faint">{fmtDate(d.expiryDate)}</div>
				</div>
				<div className="rounded-[10px] border border-border bg-surface px-4 py-3">
					<div className="font-mono text-[10px] uppercase tracking-[0.1em] text-text-faint">SSL expires</div>
					<div className={cn("mt-1 font-mono text-[17px] font-medium", d.sslExpiring && "text-down")}>
						{d.sslDaysUntilExpiry !== null && d.sslDaysUntilExpiry !== undefined
							? (d.sslDaysUntilExpiry < 0 ? "expired" : `${d.sslDaysUntilExpiry}d`)
							: "—"}
					</div>
					<div className="font-mono text-[10px] text-text-faint">{fmtDate(d.sslValidTo)}</div>
				</div>
				<div className="rounded-[10px] border border-border bg-surface px-4 py-3">
					<div className="font-mono text-[10px] uppercase tracking-[0.1em] text-text-faint">Registrar</div>
					<div className="mt-1 truncate text-[13px] font-medium">{d.registrarName || "—"}</div>
					<div className="font-mono text-[10px] text-text-faint">{d.caProvider || "—"}</div>
				</div>
				<div className="rounded-[10px] border border-border bg-surface px-4 py-3">
					<div className="font-mono text-[10px] uppercase tracking-[0.1em] text-text-faint">Providers</div>
					<div className="mt-1 truncate text-[13px] font-medium">{d.dnsProvider || "—"}</div>
					<div className="font-mono text-[10px] text-text-faint">{d.hostingProvider || d.emailProvider || "—"}</div>
				</div>
			</div>

			<div className="grid grid-cols-2 gap-3 max-[820px]:grid-cols-1">
				<Section title="Registration">
					<Row k="Created" v={fmtDate(d.creationDate)} />
					<Row k="Updated" v={fmtDate(d.updatedDate)} />
					<Row k="Registrar" v={d.registrarName} />
					<Row k="IANA ID" v={d.registrarId} mono />
					<Row k="Registry ID" v={d.registryDomainId} mono />
					<Row k="Registrant" v={[d.registrantName, d.registrantOrg].filter(Boolean).join(" / ")} />
					<Row k="Country" v={d.registrantCountry} />
					<Row k="Abuse" v={d.abuseEmail} mono />
					{d.statuses && d.statuses.length > 0 ? <Row k="Status" v={d.statuses.join(", ")} mono /> : null}
				</Section>

				<Section title="DNS">
					{d.nameServers?.map((n) => <Row key={n} k="NS" v={n} mono />)}
					{d.mxRecords?.map((m) => <Row key={m} k="MX" v={m} mono />)}
					{d.ipv4?.map((a) => <Row key={a} k="A" v={a} mono />)}
					{d.ipv6?.map((a) => <Row key={a} k="AAAA" v={a} mono />)}
					{d.cname ? <Row k="CNAME" v={d.cname} mono /> : null}
					{d.txtRecords?.slice(0, 4).map((t, i) => <Row key={i} k="TXT" v={t.length > 60 ? t.slice(0, 60) + "…" : t} mono />)}
					{d.dnsProvider ? <Row k="DNS provider" v={d.dnsProvider} /> : null}
					{d.emailProvider ? <Row k="Email" v={d.emailProvider} /> : null}
				</Section>

				<Section title="TLS certificate">
					<Row k="Issuer" v={d.sslIssuer} />
					<Row k="CA" v={d.caProvider} />
					<Row k="Valid" v={`${fmtDate(d.sslValidFrom)} → ${fmtDate(d.sslValidTo)}`} mono />
					<Row k="Subject" v={d.sslSubject} mono />
					<Row k="Key" v={d.sslKeySize ? `${d.sslKeySize} bit` : ""} mono />
					<Row k="Algorithm" v={d.sslSigAlgo} mono />
					{d.sslAltNames && d.sslAltNames.length > 0 ? (
						<Row k="SANs" v={d.sslAltNames.slice(0, 5).join(", ") + (d.sslAltNames.length > 5 ? ` +${d.sslAltNames.length - 5}` : "")} mono />
					) : null}
					<Row k="SHA-256" v={d.sslFingerprint ? d.sslFingerprint.slice(0, 29) + "…" : ""} mono />
				</Section>

				<Section title="Host">
					<Row k="Location" v={[d.hostCity, d.hostRegion, d.hostCountry].filter(Boolean).join(", ")} />
					<Row k="ISP" v={d.hostIsp} />
					<Row k="Org" v={d.hostOrg} />
					<Row k="AS" v={d.hostAs} mono />
					<Row k="Hosting" v={d.hostingProvider} />
					<Row k="Checked" v={d.lastChecked ? new Date(d.lastChecked).toLocaleString() : "—"} mono />
				</Section>
			</div>

			{Object.keys(d.headers ?? {}).length > 0 ? (
				<details className="mt-3 overflow-hidden rounded-[10px] border border-border bg-surface">
					<summary className="cursor-pointer px-4 py-2.5 font-mono text-[10.5px] uppercase tracking-[0.12em] text-text-faint">
						Response headers ({Object.keys(d.headers ?? {}).length})
					</summary>
					<div className="border-t border-border px-4 py-2">
						{Object.entries(d.headers ?? {}).map(([k, v]) => (
							<div key={k} className="flex gap-3 py-0.5 font-mono text-[11px]">
								<span className="w-44 shrink-0 text-text-faint">{k}</span>
								<span className="truncate text-text-dim">{v}</span>
							</div>
						))}
					</div>
				</details>
			) : null}

			<h2 className="mb-2 mt-6 font-mono text-[10.5px] uppercase tracking-[0.12em] text-text-faint">Check history</h2>
			<div className="overflow-hidden rounded-[10px] border border-border">
				<table className="w-full text-[12px]">
					<tbody>
						{checks.map((c) => (
							<tr key={c.id} className="border-b border-border/60 last:border-0">
								<td className="px-4 py-2 font-mono text-[10.5px] text-text-faint">{new Date(c.checkedAt ?? "").toLocaleString()}</td>
								<td className="px-4 py-2 font-mono text-[11px] text-text-dim">exp {fmtDate(c.expiryDate)}</td>
								<td className="px-4 py-2 font-mono text-[11px] text-text-dim">ssl {fmtDate(c.sslValidTo)}</td>
								<td className="px-4 py-2 font-mono text-[11px] text-text-faint">{(c.ipv4 ?? []).join(", ")}</td>
							</tr>
						))}
						{checks.length === 0 && (
							<tr><td className="px-4 py-6 text-center text-text-faint">No snapshots yet.</td></tr>
						)}
					</tbody>
				</table>
			</div>

			<DomainDialog open={dlgOpen} onOpenChange={setDlgOpen} domain={d} onSaved={load} />
		</main>
	);
}
