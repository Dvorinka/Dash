import { useCallback, useEffect, useState } from "react";
import { api } from "@/api";
import type { PublicStatus } from "@/types";
import { cn } from "@/lib/utils";

// Public status page — clean read-only view; refreshes every 30s.
// Route: /status/:slug -> GET /api/status-pages/:slug/public.

// status strings arrive as plain `string` — map with functions, not
// open dictionaries, so the lookup type stays honest.
function overallBanner(overall?: string) {
	if (overall === "down") return { label: "Major outage", cls: "bg-down" };
	if (overall === "maintenance") return { label: "Under maintenance", cls: "bg-text-faint/60" };
	if (overall === "up") return { label: "All systems operational", cls: "bg-up" };
	return { label: "Checking…", cls: "bg-text-faint/40" };
}

function dotCls(status?: string) {
	if (status === "up") return "bg-up";
	if (status === "down") return "bg-down";
	if (status === "maintenance") return "bg-text-faint/60";
	return "bg-text-faint/40";
}

export function StatusPublicPage({ slug }: { slug: string }) {
	const [st, setSt] = useState<PublicStatus | null>(null);
	const [missing, setMissing] = useState(false);

	const load = useCallback(async () => {
		const { data, response } = await api.GET("/api/status-pages/{slug}/public", { params: { path: { slug } } });
		if (response.status === 404) setMissing(true);
		if (data) setSt(data);
	}, [slug]);

	useEffect(() => {
		void load();
		const t = setInterval(() => void load(), 30_000);
		return () => clearInterval(t);
	}, [load]);

	if (missing) return <main className="mx-auto max-w-2xl px-7 py-16 text-center text-[13px] text-text-faint">Status page not found.</main>;
	if (!st) return <main className="mx-auto max-w-2xl px-7 py-16 text-center text-[13px] text-text-faint">Loading…</main>;

	const banner = overallBanner(st.overall);

	return (
		<main className="mx-auto w-full max-w-2xl px-7 py-10">
			<div className="mb-6 text-center">
				<h1 className="text-[17px] font-semibold tracking-tight">{st.title}</h1>
				{st.description ? <p className="mt-1 text-[12px] text-text-faint">{st.description}</p> : null}
			</div>

			<div className={cn("mb-8 rounded-[10px] px-5 py-3.5 text-center text-[13px] font-medium text-white", banner.cls)}>
				{banner.label}
			</div>

			{(st.monitors ?? []).length > 0 && (
				<section className="mb-6">
					<h2 className="mb-2 font-mono text-[10.5px] uppercase tracking-[0.12em] text-text-faint">Services</h2>
					<div className="overflow-hidden rounded-[10px] border border-border">
						{(st.monitors ?? []).map((m) => (
							<div key={m.id} className="flex items-center gap-3 border-b border-border/60 px-4 py-2.5 last:border-0">
								<span className={cn("size-1.5 shrink-0 rounded-full", dotCls(m.status))} />
								<span className="min-w-0 flex-1 truncate text-[13px]">{m.name}</span>
								<span className="font-mono text-[10.5px] text-text-faint">{m.uptime24h?.toFixed(1)}%</span>
								<span className="w-20 text-right font-mono text-[10.5px] uppercase text-text-faint">{m.status}</span>
							</div>
						))}
					</div>
				</section>
			)}

			{(st.systems ?? []).length > 0 && (
				<section className="mb-6">
					<h2 className="mb-2 font-mono text-[10.5px] uppercase tracking-[0.12em] text-text-faint">Systems</h2>
					<div className="overflow-hidden rounded-[10px] border border-border">
						{(st.systems ?? []).map((s) => (
							<div key={s.id} className="flex items-center gap-3 border-b border-border/60 px-4 py-2.5 last:border-0">
								<span className={cn("size-1.5 shrink-0 rounded-full", dotCls(s.status))} />
								<span className="min-w-0 flex-1 truncate text-[13px]">{s.name}</span>
								<span className="font-mono text-[10.5px] uppercase text-text-faint">{s.status}</span>
							</div>
						))}
					</div>
				</section>
			)}

			{(st.maintenance ?? []).length > 0 && (
				<section className="mb-6">
					<h2 className="mb-2 font-mono text-[10.5px] uppercase tracking-[0.12em] text-text-faint">Maintenance</h2>
					<div className="overflow-hidden rounded-[10px] border border-border">
						{(st.maintenance ?? []).map((w) => (
							<div key={w.id} className="flex items-center gap-3 border-b border-border/60 px-4 py-2.5 last:border-0">
								<span className="min-w-0 flex-1 truncate text-[13px]">{w.title}</span>
								<span className="font-mono text-[10.5px] text-text-faint">
									{new Date(w.startsAt ?? "").toLocaleString()} → {new Date(w.endsAt ?? "").toLocaleString()}
								</span>
							</div>
						))}
					</div>
				</section>
			)}

			{(st.incidents ?? []).length > 0 && (
				<section className="mb-6">
					<h2 className="mb-2 font-mono text-[10.5px] uppercase tracking-[0.12em] text-text-faint">Incidents</h2>
					<div className="flex flex-col gap-2">
						{(st.incidents ?? []).map((inc) => (
							<div key={inc.id} className="rounded-[10px] border border-border px-4 py-3">
								<div className="flex items-center gap-3">
									<span className="min-w-0 flex-1 truncate text-[13px] font-medium">{inc.title}</span>
									<span className="font-mono text-[10.5px] uppercase text-text-faint">{inc.status}</span>
								</div>
								{(inc.updates ?? []).slice(-3).map((u) => (
									<p key={u.id} className="mt-1.5 text-[11.5px] text-text-faint">
										<span className="font-mono text-[10px] uppercase">{u.status}</span> — {u.message}
									</p>
								))}
							</div>
						))}
					</div>
				</section>
			)}

			<p className="mt-10 text-center font-mono text-[10px] text-text-faint">powered by dash</p>
		</main>
	);
}
