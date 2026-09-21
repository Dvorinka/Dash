import { useCallback, useEffect, useState } from "react";
import { Plus, Trash2 } from "lucide-react";
import { api } from "@/api";
import type { Incident } from "@/types";
import { Button } from "@/components/ui/button";
import {
	Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { cn } from "@/lib/utils";

const SEV_COLOR = {
	critical: "text-down",
	major: "text-down",
	minor: "text-text-faint",
} satisfies Record<NonNullable<Incident["severity"]>, string>;

type Status = NonNullable<Incident["status"]>;
const NEXT: Partial<Record<Status, { label: string; to: Status }[]>> = {
	open: [{ label: "Acknowledge", to: "ack" }, { label: "Resolve", to: "resolved" }],
	ack: [{ label: "Resolve", to: "resolved" }],
	resolved: [{ label: "Close", to: "closed" }],
};

export function IncidentsPage() {
	const [incidents, setIncidents] = useState<Incident[]>([]);
	const [loaded, setLoaded] = useState(false);
	const [showAll, setShowAll] = useState(false);
	const [dlgOpen, setDlgOpen] = useState(false);
	const [title, setTitle] = useState("");
	const [sev, setSev] = useState("major");
	const [msg, setMsg] = useState("");
	const [busy, setBusy] = useState(false);
	const [expanded, setExpanded] = useState<string | null>(null);
	const [note, setNote] = useState("");

	const load = useCallback(async () => {
		const { data } = await api.GET("/api/incidents", showAll ? {} : { params: { query: { status: "open" } } });
		if (data) setIncidents(data);
		setLoaded(true);
	}, [showAll]);

	useEffect(() => {
		void load();
		const t = setInterval(() => void load(), 30_000);
		return () => clearInterval(t);
	}, [load]);

	async function create() {
		if (!title.trim()) return;
		setBusy(true);
		// The select only offers the three severities, but its value is a
		// plain string — narrow at the boundary instead of asserting.
		const severity = sev === "critical" || sev === "minor" ? sev : "major";
		await api.POST("/api/incidents", { body: { title: title.trim(), severity, message: msg || undefined } });
		setBusy(false);
		setDlgOpen(false);
		setTitle(""); setMsg("");
		void load();
	}

	async function transition(id: string | undefined, to: Status, message?: string) {
		if (!id) return;
		await api.PATCH("/api/incidents/{id}", { params: { path: { id } }, body: { status: to, message } });
		await load();
	}

	async function addNote(id: string | undefined) {
		if (!id || !note.trim()) return;
		await api.POST("/api/incidents/{id}/updates", { params: { path: { id } }, body: { message: note.trim() } });
		setNote("");
		await load();
	}

	return (
		<main className="mx-auto w-full max-w-5xl px-7 py-8">
			<div className="mb-5 flex items-center justify-between">
				<div>
					<h1 className="text-[15px] font-semibold tracking-tight">Incidents</h1>
					<p className="text-[12px] text-text-faint">Auto-opened on monitor outages, or tracked manually.</p>
				</div>
				<div className="flex items-center gap-2">
					<Button variant="outline" size="sm" onClick={() => setShowAll((v) => !v)}>
						{showAll ? "Open only" : "Show all"}
					</Button>
					<Button onClick={() => setDlgOpen(true)}>
						<Plus size={13} strokeWidth={2.2} /> New incident
					</Button>
				</div>
			</div>

			{loaded && incidents.length === 0 ? (
				<div className="rounded-[10px] border border-dashed border-border px-6 py-14 text-center">
					<p className="text-[13px] text-text-dim">No open incidents.</p>
				</div>
			) : (
				<div className="flex flex-col gap-2.5">
					{incidents.map((inc) => (
						<div key={inc.id} className="rounded-[10px] border border-border bg-surface px-4 py-3">
							<div className="flex items-center gap-3">
								<span className={cn("font-mono text-[10px] uppercase tracking-[0.08em]",
									inc.severity ? SEV_COLOR[inc.severity] : "text-text-faint")}>
									{inc.severity}
								</span>
								<button type="button" className="min-w-0 flex-1 truncate text-left text-[13px] font-medium hover:underline"
									onClick={() => setExpanded(expanded === inc.id ? null : (inc.id ?? null))}>
									{inc.title}
								</button>
								<span className="font-mono text-[10.5px] uppercase text-text-faint">{inc.status}</span>
								<span className="font-mono text-[10.5px] text-text-faint">
									{inc.createdAt ? new Date(inc.createdAt).toLocaleString() : ""}
								</span>
								{(inc.status ? (NEXT[inc.status] ?? []) : []).map((n) => (
									<Button key={n.to} variant="outline" size="sm" onClick={() => void transition(inc.id, n.to)}>
										{n.label}
									</Button>
								))}
								<Button variant="outline" size="icon" aria-label="Delete"
									onClick={() => {
										if (inc.id && confirm(`Delete incident "${inc.title}"?`)) {
											void api.DELETE("/api/incidents/{id}", { params: { path: { id: inc.id } } }).then(load);
										}
									}}>
									<Trash2 size={12} />
								</Button>
							</div>
							{expanded === inc.id && (
								<div className="mt-3 border-t border-border/60 pt-3">
									<div className="mb-2.5 flex flex-col gap-1.5">
										{(inc.updates ?? []).map((u) => (
											<div key={u.id} className="flex items-baseline gap-2.5 text-[12px]">
												<span className="shrink-0 font-mono text-[10px] uppercase tracking-wide text-text-faint">{u.status}</span>
												<span className="flex-1 text-text-dim">{u.message}</span>
												<span className="shrink-0 font-mono text-[10px] text-text-faint">
													{u.at ? new Date(u.at).toLocaleString() : ""}
												</span>
											</div>
										))}
										{(inc.updates ?? []).length === 0 && (
											<p className="text-[11.5px] text-text-faint">No updates yet.</p>
										)}
									</div>
									<div className="flex gap-2">
										<Input value={note} onChange={(e) => setNote(e.target.value)}
											placeholder="Add a note…" className="h-8 text-[12px]"
											onKeyDown={(e) => { if (e.key === "Enter") void addNote(inc.id); }} />
										<Button variant="outline" size="sm" onClick={() => void addNote(inc.id)}>Post</Button>
									</div>
								</div>
							)}
						</div>
					))}
				</div>
			)}

			<Dialog open={dlgOpen} onOpenChange={setDlgOpen}>
				<DialogContent>
					<DialogHeader>
						<DialogTitle>New incident</DialogTitle>
						<DialogDescription>Track an outage or degradation manually.</DialogDescription>
					</DialogHeader>
					<div className="flex flex-col gap-4">
						<div className="flex flex-col gap-1.5">
							<Label htmlFor="inc-title">Title</Label>
							<Input id="inc-title" value={title} onChange={(e) => setTitle(e.target.value)}
								placeholder="NAS offline" autoFocus />
						</div>
						<div className="grid grid-cols-2 gap-3">
							<div className="flex flex-col gap-1.5">
								<Label>Severity</Label>
								<Select value={sev} onValueChange={setSev}>
									<SelectTrigger><SelectValue /></SelectTrigger>
									<SelectContent>
										<SelectItem value="minor">minor</SelectItem>
										<SelectItem value="major">major</SelectItem>
										<SelectItem value="critical">critical</SelectItem>
									</SelectContent>
								</Select>
							</div>
							<div className="flex flex-col gap-1.5">
								<Label htmlFor="inc-msg">Note (optional)</Label>
								<Input id="inc-msg" value={msg} onChange={(e) => setMsg(e.target.value)} placeholder="First observed…" />
							</div>
						</div>
					</div>
					<DialogFooter>
						<Button variant="outline" onClick={() => setDlgOpen(false)} disabled={busy}>Cancel</Button>
						<Button onClick={() => void create()} disabled={busy || !title.trim()}>
							{busy ? "Creating…" : "Create"}
						</Button>
					</DialogFooter>
				</DialogContent>
			</Dialog>
		</main>
	);
}
