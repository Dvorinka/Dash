import { useCallback, useEffect, useState } from "react";
import { Link } from "wouter";
import { api } from "@/api";
import { useBoard } from "@/board/store";
import type { DevOverview, DomainView, Incident, MonitorView, System } from "@/types";
import { formatLatency } from "@/board/primitives";
import { cn } from "@/lib/utils";
import { t } from "@/i18n";

// Overview joins the board's reachability probes with the other sections.
// Each page still owns its detail. This one lists what is down, expiring, or open,
// and the board services that are up.

interface SourceOk {
	monitors: boolean;
	domains: boolean;
	systems: boolean;
	incidents: boolean;
	dev: boolean;
}

interface Pulse {
	monitors: MonitorView[];
	domains: DomainView[];
	systems: System[];
	incidents: Incident[];
	dev?: DevOverview;
	ok: SourceOk;
}

function daysSoon(days: number | null | undefined) {
	return days !== null && days !== undefined && days <= 30;
}

export function OverviewPage() {
	const board = useBoard();
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
			ok: {
				monitors: monitors.response.ok,
				domains: domains.response.ok,
				systems: systems.response.ok,
				incidents: incidents.response.ok,
				dev: dev.response.ok,
			},
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
	const ok = pulse?.ok;
	const down = ok?.monitors ? monitors.filter((m) => m.status === "down") : [];
	const expiring = ok?.domains ? domains.filter((d) => d.expiring || d.sslExpiring || daysSoon(d.daysUntilExpiry) || daysSoon(d.sslDaysUntilExpiry)) : [];
	const offline = ok?.systems ? systems.filter((s) => s.status === "down") : [];
	const openIncidents = ok?.incidents ? incidents : [];
	const services = board.sections.flatMap((s) =>
		s.items.filter((i) => i.kind !== "widget" && i.urls.length > 0).map((item) => ({ item, section: s.name ?? "" })),
	);
	const statusReady = services.length === 0 || services.some((s) => board.statuses[s.item.id]);
	const boardDown = statusReady ? services.filter((s) => board.statuses[s.item.id]?.status === "down") : [];
	const boardUp = statusReady ? services.filter((s) => board.statuses[s.item.id]?.status === "up") : [];
	const activity = (pulse?.dev?.activity ?? []).slice(0, 6);
	const showActivity = activity.length > 0;
	const sourcesOk = !!ok && ok.monitors && ok.domains && ok.systems && ok.incidents && ok.dev;
	const anyRow = boardDown.length + down.length + expiring.length + offline.length + openIncidents.length > 0;
	const clear = sourcesOk && statusReady && !anyRow;

	return (
		<main className="mx-auto w-full max-w-[1140px] px-4 py-6 md:px-7 md:py-8">
			<div className="mb-6">
				<h1 className="text-[15px] font-semibold tracking-tight">{t("overview.title")}</h1>
				<p className="text-[12px] text-text-faint">{t("overview.desc")}</p>
			</div>

			{pulse && !sourcesOk && (
				<p className="mb-4 text-[12px] text-destructive">{t("overview.partial")}</p>
			)}

			<div className="grid grid-cols-2 gap-3 sm:grid-cols-3 xl:grid-cols-6">
				<Stat href="/" label={t("nav.board")} value={statusReady ? boardDown.length : null} unit={t("overview.down")} meta={statusReady ? t("overview.of", { n: services.length }) : ""} hot={boardDown.length > 0} />
				<Stat href="/monitors" label={t("nav.monitors")} value={ok?.monitors ? down.length : null} unit={t("overview.down")} meta={ok?.monitors ? t("overview.of", { n: monitors.length }) : ""} hot={down.length > 0} />
				<Stat href="/domains" label={t("nav.domains")} value={ok?.domains ? expiring.length : null} unit={t("overview.expiring")} meta={ok?.domains ? t("overview.of", { n: domains.length }) : ""} hot={expiring.length > 0} />
				<Stat href="/systems" label={t("nav.systems")} value={ok?.systems ? offline.length : null} unit={t("overview.offline")} meta={ok?.systems ? t("overview.of", { n: systems.length }) : ""} hot={offline.length > 0} />
				<Stat href="/incidents" label={t("nav.incidents")} value={ok?.incidents ? openIncidents.length : null} unit={t("overview.open")} meta={ok?.incidents ? t("overview.ackIncluded") : ""} hot={openIncidents.length > 0} />
				<Stat href="/dev" label={t("nav.dev")} value={ok?.dev ? (pulse?.dev?.openPRs ?? 0) : null} unit={t("overview.prs")} meta={ok?.dev ? t("overview.issues", { n: pulse?.dev?.openIssues ?? 0 }) : ""} hot={(pulse?.dev?.openPRs ?? 0) > 0} />
			</div>

			<div className={cn("mt-8 grid gap-6", showActivity && "lg:grid-cols-2")}>
				<section>
					<h2 className="mb-2 font-mono text-[10.5px] uppercase tracking-[0.12em] text-text-faint">{t("overview.attention")}</h2>
					<div className="overflow-hidden rounded-[10px] border border-border">
						{!pulse || (!anyRow && !statusReady) ? (
							<p className="px-4 py-6 text-[12px] text-text-faint">{t("common.loading")}</p>
						) : clear ? (
							<p className="px-4 py-6 text-[13px] text-text-dim">{t("overview.allClear")}</p>
						) : anyRow ? (
							<>
								{boardDown.map(({ item, section }) => (
									<Row
										key={`svc-${item.id}`}
										href={item.urls[0]?.url ?? "/"}
										blank={!!item.urls[0]}
										name={item.name}
										kind={section || t("nav.board")}
										detail={probeDetail(item.urls[0]?.url ?? "", board.statuses[item.id]?.latencyMs)}
										hot
									/>
								))}
								{down.map((m) => (
									<Row key={`mon-${m.id}`} href={`/monitors/${m.id}`} name={m.name ?? ""} kind={t("nav.monitors")} detail={m.status ?? ""} hot />
								))}
								{offline.map((s) => (
									<Row key={`sys-${s.id}`} href={`/systems/${s.id}`} name={s.name ?? ""} kind={t("nav.systems")} detail={s.host ?? s.status ?? ""} hot />
								))}
								{expiring.map((d) => (
									<Row key={`dom-${d.id}`} href={`/domains/${d.id}`} name={d.name ?? ""} kind={t("nav.domains")} detail={expiryDetail(d)} hot />
								))}
								{openIncidents.map((i) => (
									<Row key={`inc-${i.id}`} href="/incidents" name={i.title ?? ""} kind={i.severity ?? t("nav.incidents")} detail={i.status ?? ""} hot={i.severity === "critical"} />
								))}
							</>
						) : (
							<p className="px-4 py-6 text-[13px] text-text-dim">{t("overview.partial")}</p>
						)}
					</div>
				</section>

				{showActivity && (
					<section>
						<h2 className="mb-2 font-mono text-[10.5px] uppercase tracking-[0.12em] text-text-faint">{t("overview.activity")}</h2>
						<div className="overflow-hidden rounded-[10px] border border-border">
							{activity.map((a) => (
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
				)}
				{boardUp.length > 0 && (
					<section className={cn(showActivity && "lg:col-span-2")}>
						<h2 className="mb-2 font-mono text-[10.5px] uppercase tracking-[0.12em] text-text-faint">{t("overview.running")}</h2>
						<div className="overflow-hidden rounded-[10px] border border-border">
							{boardUp.map(({ item, section }) => (
								<Row
									key={`up-${item.id}`}
									href={item.urls[0]?.url ?? "/"}
									blank={!!item.urls[0]}
									name={item.name}
									kind={section || t("nav.board")}
									detail={probeDetail(item.urls[0]?.url ?? "", board.statuses[item.id]?.latencyMs)}
									up
								/>
							))}
						</div>
					</section>
				)}
			</div>
		</main>
	);
}

function hostOf(url: string) {
	try {
		return new URL(url).host;
	} catch {
		return "";
	}
}

function probeDetail(url: string, ms: number | undefined) {
	const host = hostOf(url);
	const latency = formatLatency(ms);
	if (host && latency) return `${host} · ${latency}`;
	return host || latency;
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
				<span className={cn("font-mono text-[22px] leading-none tabular-nums", hot ? "text-down" : "text-text")}>{value === null ? "-" : value}</span>
				<span className="text-[12px] text-text-dim">{unit}</span>
			</div>
			<div className="mt-1.5 text-[11px] leading-snug text-text-faint">{meta}</div>
		</Link>
	);
}

function Row({ href, name, kind, detail, hot, up, blank }: {
	href: string;
	name: string;
	kind: string;
	detail: string;
	hot?: boolean;
	up?: boolean;
	blank?: boolean;
}) {
	const className = "flex items-center gap-3 border-b border-border/60 px-4 py-2.5 last:border-0 hover:bg-surface-hover/60";
	const body = (
		<>
			<span className={cn("size-1.5 shrink-0 rounded-full", hot ? "bg-down" : up ? "bg-up" : "bg-text-faint/50")} />
			<span className="min-w-0 flex-1 truncate text-[13px]">{name}</span>
			<span className="shrink-0 font-mono text-[10px] uppercase tracking-[0.06em] text-text-faint">{kind}</span>
			<span className="max-w-[46%] shrink-0 truncate text-right font-mono text-[10.5px] text-text-dim">{detail}</span>
		</>
	);
	if (blank) {
		return <a href={href} target="_blank" rel="noreferrer" className={className}>{body}</a>;
	}
	return <Link href={href} className={className}>{body}</Link>;
}
