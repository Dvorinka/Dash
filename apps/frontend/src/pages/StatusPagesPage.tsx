import { useCallback, useEffect, useState } from "react";
import { Link } from "wouter";
import { ExternalLink, Plus, Trash2 } from "lucide-react";
import { api } from "@/api";
import type { MaintenanceWindow, MonitorView, StatusPage } from "@/types";
import { Button } from "@/components/ui/button";
import {
	Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";

// Admin for public status pages + maintenance windows. The public view
// itself lives at /status/:slug.

export function StatusPagesPage() {
	const [pages, setPages] = useState<StatusPage[]>([]);
	const [windows, setWindows] = useState<MaintenanceWindow[]>([]);
	const [monitors, setMonitors] = useState<MonitorView[]>([]);
	const [loaded, setLoaded] = useState(false);

	const [pageOpen, setPageOpen] = useState(false);
	const [pTitle, setPTitle] = useState("");
	const [pSlug, setPSlug] = useState("");
	const [pDesc, setPDesc] = useState("");
	const [pMonitors, setPMonitors] = useState<string[]>([]);

	const [winOpen, setWinOpen] = useState(false);
	const [wTitle, setWTitle] = useState("");
	const [wStart, setWStart] = useState("");
	const [wEnd, setWEnd] = useState("");
	const [busy, setBusy] = useState(false);
	const [err, setErr] = useState("");

	const load = useCallback(async () => {
		const [{ data: p }, { data: w }, { data: m }] = await Promise.all([
			api.GET("/api/status-pages"),
			api.GET("/api/maintenance"),
			api.GET("/api/monitors"),
		]);
		if (p) setPages(p);
		if (w) setWindows(w);
		if (m) setMonitors(m);
		setLoaded(true);
	}, []);

	useEffect(() => {
		void load();
		const t = setInterval(() => void load(), 30_000);
		return () => clearInterval(t);
	}, [load]);

	async function createPage() {
		if (!pTitle.trim()) return;
		setBusy(true); setErr("");
		const { error } = await api.POST("/api/status-pages", {
			body: {
				title: pTitle.trim(),
				...(pSlug.trim() ? { slug: pSlug.trim() } : {}),
				description: pDesc.trim(),
				monitorIds: pMonitors,
			},
		});
		setBusy(false);
		if (error) { setErr("Could not create — slug may be taken."); return; }
		setPageOpen(false); setPTitle(""); setPSlug(""); setPDesc(""); setPMonitors([]);
		void load();
	}

	async function createWindow() {
		if (!wTitle.trim() || !wStart || !wEnd) return;
		setBusy(true); setErr("");
		const { error } = await api.POST("/api/maintenance", {
			body: {
				title: wTitle.trim(),
				startsAt: new Date(wStart).toISOString(),
				endsAt: new Date(wEnd).toISOString(),
			},
		});
		setBusy(false);
		if (error) { setErr("Invalid window — check times."); return; }
		setWinOpen(false); setWTitle(""); setWStart(""); setWEnd("");
		void load();
	}

	return (
		<main className="mx-auto w-full max-w-5xl px-7 py-8">
			<div className="mb-5 flex items-center justify-between">
				<div>
					<h1 className="text-[15px] font-semibold tracking-tight">Status pages</h1>
					<p className="text-[12px] text-text-faint">Public live views at <code className="font-mono">/status/&lt;slug&gt;</code>.</p>
				</div>
				<Button onClick={() => setPageOpen(true)}>
					<Plus size={13} strokeWidth={2.2} /> New page
				</Button>
			</div>

			{loaded && pages.length === 0 ? (
				<div className="mb-8 rounded-[10px] border border-dashed border-border px-6 py-10 text-center">
					<p className="text-[13px] text-text-dim">No status pages yet.</p>
				</div>
			) : (
				<div className="mb-8 flex flex-col gap-2">
					{pages.map((p) => (
						<div key={p.id} className="flex items-center gap-3 rounded-[10px] border border-border bg-surface px-4 py-3">
							<span className="min-w-0 flex-1 truncate text-[13px] font-medium">{p.title}</span>
							<code className="font-mono text-[11px] text-text-faint">/status/{p.slug}</code>
							<Link href={`/status/${p.slug}`} className="shrink-0">
								<Button variant="outline" size="icon" aria-label="Open public page"><ExternalLink size={12} /></Button>
							</Link>
							<Button variant="outline" size="icon" aria-label="Delete"
								onClick={() => {
									if (p.id && confirm(`Delete status page "${p.title}"?`)) {
										void api.DELETE("/api/status-pages/{id}", { params: { path: { id: p.id } } }).then(load);
									}
								}}>
								<Trash2 size={12} />
							</Button>
						</div>
					))}
				</div>
			)}

			<div className="mb-5 flex items-center justify-between">
				<div>
					<h2 className="text-[15px] font-semibold tracking-tight">Maintenance windows</h2>
					<p className="text-[12px] text-text-faint">Active windows mute alerts + auto-incidents and mark covered monitors.</p>
				</div>
				<Button variant="outline" onClick={() => setWinOpen(true)}>
					<Plus size={13} strokeWidth={2.2} /> Schedule
				</Button>
			</div>

			{loaded && windows.length === 0 ? (
				<div className="rounded-[10px] border border-dashed border-border px-6 py-10 text-center">
					<p className="text-[13px] text-text-dim">No maintenance scheduled.</p>
				</div>
			) : (
				<div className="flex flex-col gap-2">
					{windows.map((w) => (
						<div key={w.id} className="flex items-center gap-3 rounded-[10px] border border-border bg-surface px-4 py-3">
							<span className={cn("size-1.5 shrink-0 rounded-full", w.active ? "bg-up" : "bg-text-faint/40")} />
							<span className="min-w-0 flex-1 truncate text-[13px] font-medium">{w.title}</span>
							<span className="font-mono text-[10.5px] text-text-faint">
								{new Date(w.startsAt ?? "").toLocaleString()} → {new Date(w.endsAt ?? "").toLocaleString()}
							</span>
							<span className={cn("font-mono text-[10px] uppercase", w.active ? "text-up" : "text-text-faint")}>
								{w.active ? "active" : "scheduled"}
							</span>
							<Button variant="outline" size="icon" aria-label="Delete"
								onClick={() => {
									if (w.id && confirm(`Delete window "${w.title}"?`)) {
										void api.DELETE("/api/maintenance/{id}", { params: { path: { id: w.id } } }).then(load);
									}
								}}>
								<Trash2 size={12} />
							</Button>
						</div>
					))}
				</div>
			)}

			<Dialog open={pageOpen} onOpenChange={setPageOpen}>
				<DialogContent>
					<DialogHeader>
						<DialogTitle>New status page</DialogTitle>
						<DialogDescription>Public live view over selected monitors.</DialogDescription>
					</DialogHeader>
					<div className="flex flex-col gap-4">
						<div className="grid grid-cols-2 gap-3">
							<div className="flex flex-col gap-1.5">
								<Label htmlFor="sp-title">Title</Label>
								<Input id="sp-title" value={pTitle} onChange={(e) => setPTitle(e.target.value)} placeholder="Homelab status" autoFocus />
							</div>
							<div className="flex flex-col gap-1.5">
								<Label htmlFor="sp-slug">Slug</Label>
								<Input id="sp-slug" value={pSlug} onChange={(e) => setPSlug(e.target.value)} placeholder="auto" className="font-mono text-[12px]" />
							</div>
						</div>
						<div className="flex flex-col gap-1.5">
							<Label htmlFor="sp-desc">Description</Label>
							<Input id="sp-desc" value={pDesc} onChange={(e) => setPDesc(e.target.value)} placeholder="optional" />
						</div>
						<div className="flex flex-col gap-1.5">
							<Label>Monitors</Label>
							<div className="max-h-40 overflow-y-auto rounded-[7px] border border-border p-2">
								{monitors.map((m) => (
									<label key={m.id} className="flex items-center gap-2 px-1 py-1 text-[12px]">
										<input type="checkbox" className="accent-current"
											checked={pMonitors.includes(m.id ?? "")}
											onChange={(e) => setPMonitors((v) =>
												e.target.checked ? [...v, m.id ?? ""] : v.filter((x) => x !== m.id))} />
										{m.name}
									</label>
								))}
								{monitors.length === 0 && <p className="px-1 py-1 text-[11.5px] text-text-faint">No monitors yet — empty selection shows all.</p>}
							</div>
							<p className="text-[11px] text-text-faint">Leave all unchecked to include every active monitor.</p>
						</div>
						{err ? <p className="text-[12px] text-destructive">{err}</p> : null}
					</div>
					<DialogFooter>
						<Button variant="outline" onClick={() => setPageOpen(false)} disabled={busy}>Cancel</Button>
						<Button onClick={() => void createPage()} disabled={busy || !pTitle.trim()}>
							{busy ? "Creating…" : "Create"}
						</Button>
					</DialogFooter>
				</DialogContent>
			</Dialog>

			<Dialog open={winOpen} onOpenChange={setWinOpen}>
				<DialogContent>
					<DialogHeader>
						<DialogTitle>Schedule maintenance</DialogTitle>
						<DialogDescription>Suppresses alerts and auto-incidents for the window.</DialogDescription>
					</DialogHeader>
					<div className="flex flex-col gap-4">
						<div className="flex flex-col gap-1.5">
							<Label htmlFor="mw-title">Title</Label>
							<Input id="mw-title" value={wTitle} onChange={(e) => setWTitle(e.target.value)} placeholder="Network upgrade" autoFocus />
						</div>
						<div className="grid grid-cols-2 gap-3">
							<div className="flex flex-col gap-1.5">
								<Label htmlFor="mw-start">Starts</Label>
								<Input id="mw-start" type="datetime-local" value={wStart} onChange={(e) => setWStart(e.target.value)} />
							</div>
							<div className="flex flex-col gap-1.5">
								<Label htmlFor="mw-end">Ends</Label>
								<Input id="mw-end" type="datetime-local" value={wEnd} onChange={(e) => setWEnd(e.target.value)} />
							</div>
						</div>
						{err ? <p className="text-[12px] text-destructive">{err}</p> : null}
					</div>
					<DialogFooter>
						<Button variant="outline" onClick={() => setWinOpen(false)} disabled={busy}>Cancel</Button>
						<Button onClick={() => void createWindow()} disabled={busy || !wTitle.trim() || !wStart || !wEnd}>
							{busy ? "Saving…" : "Schedule"}
						</Button>
					</DialogFooter>
				</DialogContent>
			</Dialog>
		</main>
	);
}
