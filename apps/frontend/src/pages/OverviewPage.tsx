import { useCallback, useEffect, useState } from "react";
import { Link } from "wouter";
import { api } from "@/api";
import type { DevOverview, DomainView, Incident, MonitorView, System } from "@/types";
import { cn } from "@/lib/utils";
import { t } from "@/i18n";

// Overview is the join between the product's sections. Each page already
// owns its detail; this one only lists what is down, expiring, or open.

interface Pulse {
	monitors: MonitorView[];
	domains: DomainView[];
	systems: System[];
	incidents: Incident[];
	dev?: DevOverview;
	failed: boolean;
}

function daysSoon(days: number | null | undefined) {
	return days !== null && days !== undefined && days <= 30;
}

export function OverviewPage() {
	const [pulse, setPulse] = useState<Pulse | null>(null);

	const load = useCallback(async () => {
		const [monitors, domains, systems, incidents, dev] = await Promise.all([
			api.GET("/api/monitors"),
			api.GET("/api/domains"),
			api.GET("/api/systems"),
			api.GET("/api/incidents", { params: { query: { status: "open" } } }),
			api.GET("/api/dev/overview"),
		]);
		setPulse({
			monitors: monitors.data ?? [],
			domains: domains.data ?? [],
			systems: systems.data ?? [],
			incidents: incidents.data ?? [],
			...(dev.data ? { dev: dev.data } : {}),
			failed: [monitors, domains, systems, incidents, dev].some((r) => !r.response.ok),
		});
	}, []);

	useEffect(() => {
		void load();
		const id = setInterval(() => void load(), 30_000);
		return () => clearInterval(id);
	}, [load]);

	const monitors = pulse?.monitors ?? [];
	const domains = pulse?.domains ?? [];
	const systems = pulse?.systems ?? [];
	const incidents = pulse?.incidents ?? [];
	const down = monitors.filter((m) => m.status === "down");
	const expiring = domains.filter((d) => d.expiring || d.sslExpiring || daysSoon(d.daysUntilExpiry) || daysSoon(d.sslDaysUntilExpiry));
	const offline = systems.filter((s) => s.status === "down");
	const activity = (pulse?.dev?.activity ?? []).slice(0, 6);
	const clear = !!pulse && !pulse.failed && down.length === 0 && expiring.length === 0 && offline.length === 0 && incidents.length === 0;

	return (
		<main className="mx-auto w-full max-w-5xl px-5 py-8 md:px-7">
			<div className="mb-6">
				<h1 className="text-[15px] font-semibold tracking-tight">{t("overview.title")}</h1>
				<p className="text-[12px] text-text-faint">{t("overview.desc")}</p>
			</div>

			{pulse?.failed && (
				<p className="mb-4 text-[12px] text-destructive">{t("overview.partial")}</p>
			)}

			<div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-5">
				<Stat href="/monitors" label={t("nav.monitors")} value={pulse ? down.length : null} unit={t("overview.down")} meta={pulse ? t("overview.of", { n: monitors.length }) : ""} hot={down.length > 0} />
				<Stat href="/domains" label={t("nav.domains")} value={pulse ? expiring.length : null} unit={t("overview.expiring")} meta={pulse ? t("overview.of", { n: domains.length }) : ""} hot={expiring.length > 0} />
				<Stat href="/systems" label={t("nav.systems")} value={pulse ? offline.length : null} unit={t("overview.offline")} meta={pulse ? t("overview.of", { n: systems.length }) : ""} hot={offline.length > 0} />
				<Stat href="/incidents" label={t("nav.incidents")} value={pulse ? incidents.length : null} unit={t("overview.open")} meta={pulse ? t("overview.ackIncluded") : ""} hot={incidents.length > 0} />
				<Stat href="/dev" label={t("nav.dev")} value={pulse ? (pulse.dev?.openPRs ?? 0) : null} unit={t("overview.prs")} meta={pulse ? t("overview.issues", { n: pulse.dev?.openIssues ?? 0 }) : ""} hot={(pulse?.dev?.openPRs ?? 0) > 0} />
			</div>

			<div className="mt-8 grid gap-6 lg:grid-cols-2">
				<section>
					<h2 className="mb-2 font-mono text-[10.5px] uppercase tracking-[0.12em] text-text-faint">{t("overview.attention")}</h2>
					<div className="overflow-hidden rounded-[10px] border border-border">
						{!pulse ? (
							<p className="px-4 py-6 text-[12px] text-text-faint">{t("common.loading")}</p>
						) : clear ? (
							<p className="px-4 py-6 text-[13px] text-text-dim">{t("overview.allClear")}</p>
						) : (
							<>
								{down.map((m) => (
									<Row key={`mon-${m.id}`} href={`/monitors/${m.id}`} name={m.name ?? ""} kind={t("nav.monitors")} detail={m.status ?? ""} hot />
								))}
								{offline.map((s) => (
									<Row key={`sys-${s.id}`} href={`/systems/${s.id}`} name={s.name ?? ""} kind={t("nav.systems")} detail={s.host ?? s.status ?? ""} hot />
								))}
								{expiring.map((d) => (
									<Row key={`dom-${d.id}`} href={`/domains/${d.id}`} name={d.name ?? ""} kind={t("nav.domains")} detail={expiryDetail(d)} hot />
								))}
								{incidents.map((i) => (
									<Row key={`inc-${i.id}`} href="/incidents" name={i.title ?? ""} kind={i.severity ?? t("nav.incidents")} detail={i.status ?? ""} hot={i.severity === "critical"} />
								))}
							</>
						)}
					</div>
				</section>

				<section>
					<h2 className="mb-2 font-mono text-[10.5px] uppercase tracking-[0.12em] text-text-faint">{t("overview.activity")}</h2>
					<div className="overflow-hidden rounded-[10px] border border-border">
						{activity.length === 0 ? (
							<p className="px-4 py-6 text-[13px] text-text-dim">{t("overview.noActivity")}</p>
						) : activity.map((a) => (
							<a
								key={`${a.projectId ?? ""}-${a.kind ?? ""}-${a.number ?? 0}`}
								href={a.url ?? "/dev"}
								target={a.url ? "_blank" : undefined}
								rel={a.url ? "noreferrer" : undefined}
								className="flex items-center gap-3 border-b border-border/60 px-4 py-2.5 last:border-0 hover:bg-surface-hover/60"
							>
								<span className="w-14 shrink-0 font-mono text-[10px] uppercase tracking-[0.06em] text-text-faint">{a.kind}</span>
								<span className="min-w-0 flex-1 truncate text-[13px]">{a.title}</span>
								<span className="shrink-0 font-mono text-[10.5px] text-text-faint">{a.state}</span>
							</a>
						))}
					</div>
				</section>
			</div>
		</main>
	);
}

function expiryDetail(d: DomainView) {
	const days = d.daysUntilExpiry;
	if (days === null || days === undefined) return d.sslExpiring ? "ssl" : "";
	return days < 0 ? t("domains.expired") : `${days}d`;
}

function Stat({ href, label, value, unit, meta, hot }: {
	href: string;
	label: string;
	value: number | null;
	unit: string;
	meta: string;
	hot: boolean;
}) {
	return (
		<Link href={href} className="rounded-[10px] border border-border bg-surface px-4 py-3 transition-colors hover:bg-surface-hover">
			<div className="font-mono text-[10px] uppercase tracking-[0.1em] text-text-faint">{label}</div>
			<div className="mt-2 flex items-baseline gap-1.5">
				<span className={cn("font-mono text-[22px] leading-none", hot ? "text-down" : "text-text")}>{value === null ? "-" : value}</span>
				<span className="text-[12px] text-text-dim">{unit}</span>
			</div>
			<div className="mt-1.5 truncate text-[11px] text-text-faint">{meta}</div>
		</Link>
	);
}

function Row({ href, name, kind, detail, hot }: {
	href: string;
	name: string;
	kind: string;
	detail: string;
	hot?: boolean;
}) {
	return (
		<Link href={href} className="flex items-center gap-3 border-b border-border/60 px-4 py-2.5 last:border-0 hover:bg-surface-hover/60">
			<span className={cn("size-1.5 shrink-0 rounded-full", hot ? "bg-down" : "bg-text-faint/50")} />
			<span className="min-w-0 flex-1 truncate text-[13px]">{name}</span>
			<span className="shrink-0 font-mono text-[10px] uppercase tracking-[0.06em] text-text-faint">{kind}</span>
			<span className="w-16 shrink-0 text-right font-mono text-[10.5px] text-text-dim">{detail}</span>
		</Link>
	);
}
