import { useCallback, useEffect, useState } from "react";
import { Link } from "wouter";
import { ArrowLeft, Globe, Pause, Play, RefreshCw, Shield, Trash2 } from "lucide-react";
import { api } from "@/api";
import type { DNSRecord, DomainCheck, DomainView, Provider, Subdomain } from "@/types";
import { DomainDialog } from "@/components/DomainDialog";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { t } from "@/i18n";

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

// Vendor chip shown next to records and inside the provider strip.
function ProviderBadge({ p }: { p: Provider | null | undefined }) {
	if (!p?.name) return null;
	return (
		<span className="inline-flex shrink-0 items-center gap-1.5 rounded-full border border-border/60 bg-bg px-2 py-0.5">
			{p.icon
				? <img src={p.icon} alt="" className="size-3 rounded-[3px]" loading="lazy" />
				: <Globe size={11} className="text-text-faint" />}
			<span className="whitespace-nowrap text-[10.5px] text-text-dim">{p.name}</span>
		</span>
	);
}

function ProvTile({ label, p }: { label: string; p: Provider | undefined }) {
	return (
		<div className="flex min-w-0 items-center gap-2.5 rounded-[9px] border border-border/60 bg-bg px-3 py-2.5">
			{p?.icon
				? <img src={p.icon} alt="" className="size-5 rounded-[5px]" loading="lazy" />
				: <Globe size={17} className="shrink-0 text-text-faint" />}
			<div className="min-w-0">
				<div className="font-mono text-[9px] uppercase tracking-[0.1em] text-text-faint">{label}</div>
				<div className="truncate text-[12px] font-medium text-text">{p?.name || "—"}</div>
			</div>
		</div>
	);
}

const recTypeColor = {
	A: "text-[var(--chart-1)]",
	AAAA: "text-[var(--chart-1)]",
	CNAME: "text-[var(--chart-4)]",
	NS: "text-[var(--chart-2)]",
	MX: "text-[var(--chart-3)]",
	TXT: "text-[var(--chart-5)]",
} satisfies Record<NonNullable<DNSRecord["type"]>, string>;
const recOrder = ["A", "AAAA", "CNAME", "NS", "MX", "TXT"];

function RecordRow({ r }: { r: DNSRecord }) {
	return (
		<div className="flex items-center gap-3 border-b border-border/50 px-4 py-2 last:border-0">
			<span className={cn("w-11 shrink-0 font-mono text-[9.5px] font-semibold uppercase tracking-[0.1em]",
				(r.type ? recTypeColor[r.type] : undefined) ?? "text-text-faint")}>
				{r.type}
			</span>
			<span className="min-w-0 flex-1 truncate font-mono text-[11px] text-text" title={r.value ?? ""}>
				{r.value}
				{r.type === "MX" && r.priority !== null && r.priority !== undefined
					? <span className="text-text-faint"> (pri {r.priority})</span>
					: null}
			</span>
			<ProviderBadge p={r.provider} />
		</div>
	);
}

export function DomainDetailPage({ id }: { id: string }) {
	const [d, setD] = useState<DomainView | null>(null);
	const [checks, setChecks] = useState<DomainCheck[]>([]);
	const [subs, setSubs] = useState<Subdomain[]>([]);
	const [dlgOpen, setDlgOpen] = useState(false);
	const [refreshing, setRefreshing] = useState(false);
	const [subScanning, setSubScanning] = useState(false);

	const load = useCallback(async () => {
		const [{ data: dv }, { data: ck }, { data: sb }] = await Promise.all([
			api.GET("/api/domains/{id}", { params: { path: { id } } }),
			api.GET("/api/domains/{id}/checks", { params: { path: { id } } }),
			api.GET("/api/domains/{id}/subdomains", { params: { path: { id } } }),
		]);
		if (dv) setD(dv);
		if (ck) setChecks(ck);
		if (sb) setSubs(sb);
	}, [id]);

	useEffect(() => { void load(); }, [load]);

	if (!d) return <main className="mx-auto max-w-5xl px-7 py-8 text-[13px] text-text-faint">{t("common.loading")}</main>;

	const expiring = d.expiring || (d.daysUntilExpiry ?? Infinity) < 0;

	return (
		<main className="mx-auto w-full max-w-5xl px-7 py-8">
			<Link href="/domains" className="mb-4 inline-flex items-center gap-1.5 text-[12px] text-text-faint hover:text-text">
				<ArrowLeft size={12} /> {t("nav.domains")}
			</Link>

			<div className="mb-6 flex items-start justify-between">
				<div className="flex items-center gap-3">
					{d.faviconUrl ? <img src={d.faviconUrl} alt="" className="size-8 rounded-[7px]" /> : null}
					<div>
						<h1 className="text-[17px] font-semibold tracking-tight">{d.name}</h1>
						<p className="mt-0.5 flex items-center gap-2 font-mono text-[11px] text-text-faint">
							{d.tld ? `.${d.tld}` : ""}
							{d.dnssec === "signed" && <span className="flex items-center gap-1 text-up"><Shield size={10} /> dnssec</span>}
							{d.transferLock ? <span>{t("domains.transferLock")}</span> : null}
							{d.privacyEnabled ? <span>{t("domains.whoisPrivacy")}</span> : null}
							{d.autoRenew ? <span>{t("domains.autoRenew")}</span> : null}
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
						<RefreshCw size={12} className={cn(refreshing && "animate-spin")} /> {t("common.refresh")}
					</Button>
					<Button variant="outline" size="sm"
						onClick={() => void api.PATCH("/api/domains/{id}", { params: { path: { id } }, body: { active: !d.active } }).then(load)}>
						{d.active ? <Pause size={12} /> : <Play size={12} />} {d.active ? t("monitors.pause") : t("monitors.resume")}
					</Button>
					<Button variant="outline" size="sm" onClick={() => setDlgOpen(true)}>{t("common.edit")}</Button>
					<Button variant="outline" size="sm" aria-label={t("common.delete")}
						onClick={() => {
							if (confirm(t("domains.deleteConfirm", { name: d.name ?? "" }))) {
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
					<div className="font-mono text-[10px] uppercase tracking-[0.1em] text-text-faint">{t("domains.domainExpires")}</div>
					<div className={cn("mt-1 font-mono text-[17px] font-medium", expiring && "text-down")}>
						{d.daysUntilExpiry !== null && d.daysUntilExpiry !== undefined
							? (d.daysUntilExpiry < 0 ? t("domains.expired") : `${d.daysUntilExpiry}d`)
							: "—"}
					</div>
					<div className="font-mono text-[10px] text-text-faint">{fmtDate(d.expiryDate)}</div>
				</div>
				<div className="rounded-[10px] border border-border bg-surface px-4 py-3">
					<div className="font-mono text-[10px] uppercase tracking-[0.1em] text-text-faint">{t("domains.sslExpires")}</div>
					<div className={cn("mt-1 font-mono text-[17px] font-medium", d.sslExpiring && "text-down")}>
						{d.sslDaysUntilExpiry !== null && d.sslDaysUntilExpiry !== undefined
							? (d.sslDaysUntilExpiry < 0 ? t("domains.expired") : `${d.sslDaysUntilExpiry}d`)
							: "—"}
					</div>
					<div className="font-mono text-[10px] text-text-faint">{fmtDate(d.sslValidTo)}</div>
				</div>
				<div className="rounded-[10px] border border-border bg-surface px-4 py-3">
					<div className="font-mono text-[10px] uppercase tracking-[0.1em] text-text-faint">{t("domains.registrar")}</div>
					<div className="mt-1 truncate text-[13px] font-medium">{d.registrarName || "—"}</div>
					<div className="font-mono text-[10px] text-text-faint">{d.caProvider || "—"}</div>
				</div>
				<div className="rounded-[10px] border border-border bg-surface px-4 py-3">
					<div className="font-mono text-[10px] uppercase tracking-[0.1em] text-text-faint">{t("domains.checked")}</div>
					<div className="mt-1 truncate text-[13px] font-medium">
						{d.lastChecked ? new Date(d.lastChecked).toLocaleDateString() : "—"}
					</div>
					<div className="font-mono text-[10px] text-text-faint">every {d.intervalH}h</div>
				</div>
			</div>

			{/* Vendor strip — registrar, DNS, hosting, email, CA */}
			<Section title={t("domains.providers")}>
				<div className="grid grid-cols-5 gap-2 p-2.5 max-[900px]:grid-cols-3 max-[560px]:grid-cols-2">
					<ProvTile label={t("domains.registrar")} p={d.providers?.registrar} />
					<ProvTile label={t("domains.dns")} p={d.providers?.dns} />
					<ProvTile label={t("domains.hosting")} p={d.providers?.hosting} />
					<ProvTile label={t("domains.email")} p={d.providers?.email} />
					<ProvTile label={t("domains.tlsCert")} p={d.providers?.ca} />
				</div>
			</Section>

			<div className="mt-3 grid grid-cols-2 gap-3 max-[820px]:grid-cols-1">
				<Section title={t("domains.registration")}>
					<Row k={t("domains.created")} v={fmtDate(d.creationDate)} />
					<Row k={t("domains.updated")} v={fmtDate(d.updatedDate)} />
					<Row k={t("domains.registrar")} v={d.registrarName} />
					<Row k="IANA ID" v={d.registrarId} mono />
					<Row k="Registry ID" v={d.registryDomainId} mono />
					<Row k={t("domains.registrant")} v={[d.registrantName, d.registrantOrg].filter(Boolean).join(" / ")} />
					<Row k={t("domains.country")} v={d.registrantCountry} />
					<Row k={t("domains.abuse")} v={d.abuseEmail} mono />
					{d.statuses && d.statuses.length > 0 ? <Row k={t("domains.statusLbl")} v={d.statuses.join(", ")} mono /> : null}
				</Section>

				<Section title={t("domains.dns")}>
					{[...(d.records ?? [])]
						.sort((a, b) => recOrder.indexOf(a.type ?? "") - recOrder.indexOf(b.type ?? ""))
						.map((r, i) => <RecordRow key={i} r={r} />)}
					{(d.records ?? []).length === 0 ? (
						<p className="px-4 py-4 text-center text-[12px] text-text-faint">—</p>
					) : null}
				</Section>

				<Section title={t("domains.tlsCert")}>
					<Row k={t("domains.issuer")} v={d.sslIssuer} />
					<Row k="CA" v={d.providers?.ca ? <ProviderBadge p={d.providers.ca} /> : d.caProvider} />
					<Row k={t("domains.valid")} v={`${fmtDate(d.sslValidFrom)} → ${fmtDate(d.sslValidTo)}`} mono />
					<Row k={t("domains.subject")} v={d.sslSubject} mono />
					<Row k={t("domains.key")} v={d.sslKeySize ? `${d.sslKeySize} bit` : ""} mono />
					<Row k={t("domains.algorithm")} v={d.sslSigAlgo} mono />
					{d.sslAltNames && d.sslAltNames.length > 0 ? (
						<Row k="SANs" v={d.sslAltNames.slice(0, 5).join(", ") + (d.sslAltNames.length > 5 ? ` +${d.sslAltNames.length - 5}` : "")} mono />
					) : null}
					<Row k="SHA-256" v={d.sslFingerprint ? d.sslFingerprint.slice(0, 29) + "…" : ""} mono />
				</Section>

				<Section title={t("domains.host")}>
					<Row k={t("domains.location")} v={[d.hostCity, d.hostRegion, d.hostCountry].filter(Boolean).join(", ")} />
					<Row k={t("domains.isp")} v={d.hostIsp} />
					<Row k={t("domains.org")} v={d.hostOrg} />
					<Row k="AS" v={d.hostAs} mono />
					<Row k={t("domains.hosting")} v={d.providers?.hosting ? <ProviderBadge p={d.providers.hosting} /> : d.hostingProvider} />
					<Row k={t("domains.checked")} v={d.lastChecked ? new Date(d.lastChecked).toLocaleString() : "—"} mono />
				</Section>
			</div>

			{Object.keys(d.headers ?? {}).length > 0 ? (
				<details className="mt-3 overflow-hidden rounded-[10px] border border-border bg-surface">
					<summary className="cursor-pointer px-4 py-2.5 font-mono text-[10.5px] uppercase tracking-[0.12em] text-text-faint">
						{t("domains.responseHeaders", { n: Object.keys(d.headers ?? {}).length })}
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

			<div className="mb-2 mt-6 flex items-center justify-between">
				<h2 className="font-mono text-[10.5px] uppercase tracking-[0.12em] text-text-faint">
					{t("domains.subdomains")}{subs.length > 0 ? ` (${subs.length})` : ""}
				</h2>
				<Button variant="ghost" size="sm" disabled={subScanning}
					onClick={() => {
						setSubScanning(true);
						void api.POST("/api/domains/{id}/subdomains/refresh", { params: { path: { id } } })
							.then(({ data }) => { if (data) setSubs(data); })
							.finally(() => setSubScanning(false));
					}}>
					<RefreshCw size={11} className={cn(subScanning && "animate-spin")} /> {t("domains.scanShort")}
				</Button>
			</div>
			<div className="overflow-hidden rounded-[10px] border border-border">
				<table className="w-full text-[12px]">
					<tbody>
						{subs.map((s) => (
							<tr key={s.name} className="border-b border-border/60 last:border-0">
								<td className="px-4 py-2 font-mono text-[11px]">{s.name}</td>
								<td className="px-4 py-2 font-mono text-[11px] text-text-dim">
									{(s.ips ?? []).join(", ") || "—"}
								</td>
								<td className="px-4 py-2 text-right font-mono text-[10.5px] text-text-faint">
									{fmtDate(s.lastSeen)}
								</td>
							</tr>
						))}
						{subs.length === 0 && (
							<tr><td className="px-4 py-6 text-center text-text-faint">
								{t("domains.noSubsFound")}
							</td></tr>
						)}
					</tbody>
				</table>
			</div>

			<h2 className="mb-2 mt-6 font-mono text-[10.5px] uppercase tracking-[0.12em] text-text-faint">{t("domains.checkHistory")}</h2>
			<div className="overflow-hidden rounded-[10px] border border-border">
				<table className="w-full text-[12px]">
					<tbody>
						{checks.map((c) => (
							<tr key={c.id} className="border-b border-border/60 last:border-0">
								<td className="px-4 py-2 font-mono text-[10.5px] text-text-faint">{new Date(c.checkedAt ?? "").toLocaleString()}</td>
								<td className="px-4 py-2 font-mono text-[11px] text-text-dim">{t("domains.expPrefix")} {fmtDate(c.expiryDate)}</td>
								<td className="px-4 py-2 font-mono text-[11px] text-text-dim">{t("domains.sslPrefix")} {fmtDate(c.sslValidTo)}</td>
								<td className="px-4 py-2 font-mono text-[11px] text-text-faint">{(c.ipv4 ?? []).join(", ")}</td>
							</tr>
						))}
						{checks.length === 0 && (
							<tr><td className="px-4 py-6 text-center text-text-faint">{t("domains.noSnapshots")}</td></tr>
						)}
					</tbody>
				</table>
			</div>

			<DomainDialog open={dlgOpen} onOpenChange={setDlgOpen} domain={d} onSaved={load} />
		</main>
	);
}
