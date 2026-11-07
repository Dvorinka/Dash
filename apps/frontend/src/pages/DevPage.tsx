import { useCallback, useEffect, useState } from "react";
import { Link } from "wouter";
import { GitBranch, GitMerge, CircleDot, Plus, RefreshCw, Search, Trash2 } from "lucide-react";
import { api } from "@/api";
import type { DevProject, DevOverview, DevRepoRef } from "@/types";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
	Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle,
} from "@/components/ui/dialog";
import {
	Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select";
import { apiErr, cn } from "@/lib/utils";
import { t } from "@/i18n";

function ProviderBadge({ provider }: { provider: string }) {
	return (
		<span className={cn(
			"rounded-[5px] px-1.5 py-0.5 font-mono text-[9.5px] uppercase tracking-[0.08em]",
			provider === "github" ? "bg-up/15 text-up"
				: provider === "gitlab" ? "bg-accent/15 text-accent"
				: "bg-down/15 text-down",
		)}>
			{provider}
		</span>
	);
}

export function DevPage() {
	const [projects, setProjects] = useState<DevProject[]>([]);
	const [overview, setOverview] = useState<DevOverview | undefined>();
	const [loaded, setLoaded] = useState(false);
	const [dlgOpen, setDlgOpen] = useState(false);
	const [notice, setNotice] = useState("");

	const load = useCallback(async () => {
		const [p, o] = await Promise.all([
			api.GET("/api/dev/projects"),
			api.GET("/api/dev/overview"),
		]);
		if (p.data) setProjects(p.data);
		if (o.data) setOverview(o.data);
		setLoaded(true);
	}, []);

	useEffect(() => {
		void load();
		const t = setInterval(() => void load(), 30_000);
		return () => clearInterval(t);
	}, [load]);

	// GitHub App setup redirect lands here with ?installation_id=&setup_action=.
	useEffect(() => {
		const q = new URLSearchParams(window.location.search);
		const inst = q.get("installation_id");
		if (!inst) return;
		void api.PUT("/api/settings", { body: { "dev.githubAppInstallationId": inst } }).then(() => {
			setNotice(t("dev.appInstalled", { id: inst }));
			window.history.replaceState({}, "", "/dev");
		});
	}, []);

	async function sync(id: string) {
		await api.POST("/api/dev/projects/{id}/sync", { params: { path: { id } } });
		setTimeout(() => void load(), 2500);
	}

	async function syncAll() {
		await api.POST("/api/dev/sync");
		setTimeout(() => void load(), 3000);
	}

	return (
		<main className="mx-auto w-full max-w-5xl px-7 py-8">
			<div className="mb-5 flex items-center justify-between">
				<div>
					<h1 className="text-[15px] font-semibold tracking-tight">{t("dev.title")}</h1>
					<p className="text-[12px] text-text-faint">{t("dev.desc")}</p>
				</div>
				<div className="flex gap-2">
					{projects.length > 0 && (
						<Button variant="outline" onClick={() => void syncAll()}>
							<RefreshCw size={13} /> {t("dev.syncAll")}
						</Button>
					)}
					<Button onClick={() => setDlgOpen(true)}>
						<Plus size={13} strokeWidth={2.2} /> {t("dev.addProject")}
					</Button>
				</div>
			</div>

			{notice && (
				<p className="mb-4 rounded-[8px] border border-up/30 bg-up/10 px-3 py-2 text-[12px] text-up">{notice}</p>
			)}

			{overview && (
				<div className="mb-5 grid grid-cols-2 gap-3 sm:grid-cols-4">
					{[
						{ icon: GitBranch, label: t("dev.statProjects"), value: overview.projects },
						{ icon: CircleDot, label: t("dev.statIssues"), value: overview.openIssues },
						{ icon: GitMerge, label: t("dev.statPRs"), value: overview.openPRs },
						{ icon: GitMerge, label: t("dev.statMerged"), value: overview.mergedPRs },
					].map((s) => (
						<div key={s.label} className="rounded-[10px] border border-border bg-surface px-4 py-3">
							<div className="flex items-center gap-1.5 font-mono text-[10px] uppercase tracking-[0.08em] text-text-faint">
								<s.icon size={11} /> {s.label}
							</div>
							<div className="mt-1 font-mono text-[20px] font-medium">{s.value}</div>
						</div>
					))}
				</div>
			)}

			{overview && overview.activity && overview.activity.length > 0 && (
				<div className="mb-5 rounded-[10px] border border-border">
					<div className="border-b border-border px-4 py-2 font-mono text-[10px] uppercase tracking-[0.1em] text-text-faint">
						{t("dev.activity")}
					</div>
					{overview.activity.slice(0, 12).map((a, i) => (
						<Link key={i} href={`/dev/${a.projectId}`}
							className="flex items-center gap-3 border-b border-border/60 px-4 py-2 text-[12.5px] last:border-0 hover:bg-surface-hover/60">
							<ProviderBadge provider={projects.find((p) => p.id === a.projectId)?.provider ?? ""} />
							<span className={cn("font-mono text-[10px] uppercase", a.kind === "pr" ? "text-accent" : "text-up")}>
								{a.kind === "pr" ? "PR" : "issue"} #{a.number}
							</span>
							<span className="min-w-0 flex-1 truncate">{a.title}</span>
							<span className={cn("font-mono text-[10px] uppercase", a.state === "open" ? "text-up" : "text-text-faint")}>
								{a.state}
							</span>
							<span className="font-mono text-[10px] text-text-faint">{a.author}</span>
						</Link>
					))}
				</div>
			)}

			{loaded && projects.length === 0 ? (
				<div className="rounded-[10px] border border-dashed border-border px-6 py-14 text-center">
					<p className="text-[13px] text-text-dim">{t("dev.empty")}</p>
					<p className="mt-1 text-[11.5px] text-text-faint">{t("dev.emptyHint")}</p>
				</div>
			) : (
				<div className="grid gap-3 sm:grid-cols-2">
					{projects.map((p) => (
						<div key={p.id} className="rounded-[10px] border border-border bg-surface p-4">
							<div className="flex items-center gap-2">
								<ProviderBadge provider={p.provider} />
								<Link href={`/dev/${p.id}`} className="min-w-0 flex-1 truncate text-[13.5px] font-medium hover:underline">
									{p.name || p.path}
								</Link>
								{p.meta?.language && (
									<span className="font-mono text-[10px] text-text-faint">{p.meta.language}</span>
								)}
							</div>
							<div className="mt-0.5 truncate font-mono text-[11px] text-text-faint">{p.path}</div>
							{p.meta?.description && (
								<p className="mt-1.5 line-clamp-2 text-[12px] text-text-dim">{p.meta.description}</p>
							)}
							<div className="mt-3 flex items-center gap-3 font-mono text-[10.5px] text-text-faint">
								<span className="text-up">{p.openIssues} {t("dev.issuesLbl")}</span>
								<span className="text-accent">{p.openPRs} {t("dev.prsLbl")}</span>
								{p.meta?.stars ? <span>★ {p.meta.stars}</span> : null}
								{p.lastSyncError && <span className="text-down" title={p.lastSyncError}>sync err</span>}
							</div>
							<div className="mt-3 flex items-center gap-1.5">
								<Button variant="outline" size="sm" onClick={() => void sync(p.id)}>
									<RefreshCw size={11} /> {t("dev.sync")}
								</Button>
								<Button variant="outline" size="icon" aria-label={t("common.delete")}
									onClick={() => {
										if (confirm(t("dev.deleteConfirm", { name: p.name || p.path }))) {
											void api.DELETE("/api/dev/projects/{id}", { params: { path: { id: p.id } } }).then(load);
										}
									}}>
									<Trash2 size={12} />
								</Button>
								<span className="ml-auto font-mono text-[9.5px] text-text-faint">
									{p.lastSyncAt ? new Date(p.lastSyncAt).toLocaleTimeString() : "never synced"}
								</span>
							</div>
						</div>
					))}
				</div>
			)}

			<AddProjectDialog open={dlgOpen} onOpenChange={setDlgOpen} onAdded={load} />
		</main>
	);
}

function AddProjectDialog({ open, onOpenChange, onAdded }: {
	open: boolean;
	onOpenChange: (o: boolean) => void;
	onAdded: () => void;
}) {
	const [provider, setProvider] = useState<"github" | "gitlab" | "gitea">("github");
	const [host, setHost] = useState("");
	const [token, setToken] = useState("");
	const [path, setPath] = useState("");
	const [busy, setBusy] = useState(false);
	const [error, setError] = useState("");
	const [refs, setRefs] = useState<DevRepoRef[] | undefined>();
	const [picked, setPicked] = useState<Set<string>>(new Set());

	useEffect(() => {
		if (open) {
			setProvider("github"); setHost(""); setToken(""); setPath("");
			setRefs(undefined); setPicked(new Set()); setError("");
		}
	}, [open]);

	async function discover() {
		setBusy(true); setError("");
		const { data, error: err } = await api.POST("/api/dev/discover", {
			body: { provider, ...(host ? { host } : {}), ...(token ? { token } : {}) },
		});
		setBusy(false);
		if (err) { setError(apiErr(err, "discover failed")); return; }
		setRefs(data ?? []);
	}

	async function addOne(p: string) {
		setBusy(true); setError("");
		const { error: err } = await api.POST("/api/dev/projects", {
			body: { provider, ...(host ? { host } : {}), path: p, ...(token ? { token } : {}) },
		});
		setBusy(false);
		if (err) { setError(apiErr(err, "failed")); return; }
		setPicked((s) => { const n = new Set(s); n.delete(p); return n; });
		onAdded();
		if (picked.size <= 1) onOpenChange(false);
	}

	async function addManual() {
		if (!path.trim()) return;
		setBusy(true); setError("");
		const { error: err } = await api.POST("/api/dev/projects", {
			body: { provider, ...(host ? { host } : {}), path: path.trim(), ...(token ? { token } : {}) },
		});
		setBusy(false);
		if (err) { setError(apiErr(err, "failed")); return; }
		onOpenChange(false);
		onAdded();
	}

	return (
		<Dialog open={open} onOpenChange={onOpenChange}>
			<DialogContent className="max-w-lg">
				<DialogHeader>
					<DialogTitle>{t("dev.addTitle")}</DialogTitle>
					<DialogDescription>{t("dev.addDesc")}</DialogDescription>
				</DialogHeader>
				<div className="grid gap-3">
					<div className="grid grid-cols-2 gap-3">
						<div className="grid gap-1.5">
							<Label>{t("dev.provider")}</Label>
							<Select value={provider} onValueChange={(v) => setProvider(v === "gitlab" || v === "gitea" ? v : "github")}>
								<SelectTrigger><SelectValue /></SelectTrigger>
								<SelectContent>
									<SelectItem value="github">GitHub</SelectItem>
									<SelectItem value="gitlab">GitLab</SelectItem>
									<SelectItem value="gitea">Gitea / Forgejo</SelectItem>
								</SelectContent>
							</Select>
						</div>
						<div className="grid gap-1.5">
							<Label>{t("dev.host")}</Label>
							<Input value={host} onChange={(e) => setHost(e.target.value)}
								placeholder={provider === "github" ? "github.com" : provider === "gitlab" ? "gitlab.com" : "gitea.example.com"} />
						</div>
					</div>
					<div className="grid gap-1.5">
						<Label>{t("dev.token")}</Label>
						<Input type="password" value={token} onChange={(e) => setToken(e.target.value)}
							placeholder={t("dev.tokenHint")} />
					</div>
					{refs === undefined ? (
						<div className="grid gap-1.5">
							<Label>{t("dev.repoPath")}</Label>
							<div className="flex gap-2">
								<Input value={path} onChange={(e) => setPath(e.target.value)}
									placeholder="owner/repo or group/project" />
								<Button variant="outline" onClick={() => void discover()} disabled={busy}>
									<Search size={12} /> {t("dev.discover")}
								</Button>
							</div>
						</div>
					) : (
						<div className="grid max-h-56 gap-1 overflow-y-auto rounded-[8px] border border-border p-1.5">
							{refs.length === 0 && <p className="px-2 py-3 text-[12px] text-text-faint">{t("dev.noRepos")}</p>}
							{refs.map((r) => {
								const on = picked.has(r.path ?? "");
								return (
									<button key={r.path ?? ""} type="button"
										onClick={() => setPicked((s) => {
											const n = new Set(s);
											if (n.has(r.path ?? "")) n.delete(r.path ?? ""); else n.add(r.path ?? "");
											return n;
										})}
										className={cn(
											"flex items-center gap-2 rounded-[6px] px-2 py-1.5 text-left text-[12.5px] hover:bg-surface-hover",
											on && "bg-surface-hover",
										)}>
										<span className={cn("size-1.5 rounded-full", on ? "bg-up" : "bg-text-faint/30")} />
										<span className="min-w-0 flex-1 truncate font-mono">{r.path ?? ""}</span>
										{r.private && <span className="font-mono text-[9px] uppercase text-text-faint">priv</span>}
									</button>
								);
							})}
						</div>
					)}
					{error && <p className="text-[12px] text-down">{error}</p>}
				</div>
				<DialogFooter>
					{refs !== undefined && picked.size > 0 && (
						<Button variant="outline" onClick={() => { for (const p of picked) void addOne(p); }} disabled={busy}>
							{t("dev.addSelected", { n: picked.size })}
						</Button>
					)}
					<Button onClick={() => void addManual()} disabled={busy || !path.trim()}>
						{t("dev.add")}
					</Button>
				</DialogFooter>
			</DialogContent>
		</Dialog>
	);
}
