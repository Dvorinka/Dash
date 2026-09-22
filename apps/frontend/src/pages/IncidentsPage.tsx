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
import { t } from "@/i18n";

const SEV_COLOR = {
	critical: "text-down",
	major: "text-down",
	minor: "text-text-faint",
} satisfies Record<NonNullable<Incident["severity"]>, string>;

type Status = NonNullable<Incident["status"]>;
const NEXT: Partial<Record<Status, { label: string; to: Status }[]>> = {
	open: [{ label: t("incidents.acknowledge"), to: "ack" }, { label: t("incidents.resolve"), to: "resolved" }],
	ack: [{ label: t("incidents.resolve"), to: "resolved" }],
	resolved: [{ label: t("incidents.close"), to: "closed" }],
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
		await api.POST("/api/incidents", {
			body: { title: title.trim(), severity, ...(msg.trim() ? { message: msg.trim() } : {}) },
		});
		setBusy(false);
		setDlgOpen(false);
		setTitle(""); setMsg("");
		void load();
	}

	async function transition(id: string | undefined, to: Status, message?: string) {
		if (!id) return;
		await api.PATCH("/api/incidents/{id}", {
			params: { path: { id } },
			body: { status: to, ...(message?.trim() ? { message } : {}) },
		});
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
					<h1 className="text-[15px] font-semibold tracking-tight">{t("incidents.title")}</h1>
					<p className="text-[12px] text-text-faint">{t("incidents.subtitle")}</p>
				</div>
				<div className="flex items-center gap-2">
					<Button variant="outline" size="sm" onClick={() => setShowAll((v) => !v)}>
						{showAll ? t("incidents.openOnly") : t("incidents.showAll")}
					</Button>
					<Button onClick={() => setDlgOpen(true)}>
						<Plus size={13} strokeWidth={2.2} /> {t("incidents.new")}
					</Button>
				</div>
			</div>

			{loaded && incidents.length === 0 ? (
				<div className="rounded-[10px] border border-dashed border-border px-6 py-14 text-center">
					<p className="text-[13px] text-text-dim">{t("incidents.empty")}</p>
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
								<Button variant="outline" size="icon" aria-label={t("common.delete")}
									onClick={() => {
										if (inc.id && confirm(t("incidents.deleteConfirm", { title: inc.title ?? "" }))) {
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
											<p className="text-[11.5px] text-text-faint">{t("incidents.noUpdates")}</p>
										)}
									</div>
									<div className="flex gap-2">
										<Input value={note} onChange={(e) => setNote(e.target.value)}
											placeholder={t("incidents.notePlaceholder")} className="h-8 text-[12px]"
											onKeyDown={(e) => { if (e.key === "Enter") void addNote(inc.id); }} />
										<Button variant="outline" size="sm" onClick={() => void addNote(inc.id)}>{t("incidents.post")}</Button>
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
						<DialogTitle>{t("incidents.newTitle")}</DialogTitle>
						<DialogDescription>{t("incidents.desc")}</DialogDescription>
					</DialogHeader>
					<div className="flex flex-col gap-4">
						<div className="flex flex-col gap-1.5">
							<Label htmlFor="inc-title">{t("common.title")}</Label>
							<Input id="inc-title" value={title} onChange={(e) => setTitle(e.target.value)}
								placeholder={t("incidents.titlePlaceholder")} autoFocus />
						</div>
						<div className="grid grid-cols-2 gap-3">
							<div className="flex flex-col gap-1.5">
								<Label>{t("incidents.severity")}</Label>
								<Select value={sev} onValueChange={setSev}>
									<SelectTrigger><SelectValue /></SelectTrigger>
									<SelectContent>
										<SelectItem value="minor">{t("incidents.sevMinor")}</SelectItem>
										<SelectItem value="major">{t("incidents.sevMajor")}</SelectItem>
										<SelectItem value="critical">{t("incidents.sevCritical")}</SelectItem>
									</SelectContent>
								</Select>
							</div>
							<div className="flex flex-col gap-1.5">
								<Label htmlFor="inc-msg">{t("incidents.noteOptional")}</Label>
								<Input id="inc-msg" value={msg} onChange={(e) => setMsg(e.target.value)} placeholder={t("incidents.noteFirst")} />
							</div>
						</div>
					</div>
					<DialogFooter>
						<Button variant="outline" onClick={() => setDlgOpen(false)} disabled={busy}>{t("common.cancel")}</Button>
						<Button onClick={() => void create()} disabled={busy || !title.trim()}>
							{busy ? t("common.creating") : t("common.create")}
						</Button>
					</DialogFooter>
				</DialogContent>
			</Dialog>
		</main>
	);
}
