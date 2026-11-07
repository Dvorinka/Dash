import { useCallback, useEffect, useState } from "react";
import { Link } from "wouter";
import { ArrowLeft, ExternalLink, GitMerge, GitPullRequest, MessageSquare, RefreshCw, RotateCcw, Tag } from "lucide-react";
import { api } from "@/api";
import type { DevProject, DevIssue, DevPR, DevDayCount } from "@/types";
import { Heatmap } from "@/components/Heatmap";
import { Button } from "@/components/ui/button";
import {
	Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select";
import { cn } from "@/lib/utils";
import { t } from "@/i18n";

type Tab = "prs" | "issues";
const TABS: Tab[] = ["prs", "issues"];

function timeAgo(s?: string | null): string {
	if (!s) return "";
	const d = Date.now() - new Date(s).getTime();
	const m = Math.floor(d / 60000);
	if (m < 60) return `${m}m`;
	const h = Math.floor(m / 60);
	if (h < 48) return `${h}h`;
	return `${Math.floor(h / 24)}d`;
}

function StateChip({ state }: { state: string }) {
	return (
		<span className={cn(
			"rounded-[5px] px-1.5 py-0.5 font-mono text-[9.5px] uppercase tracking-[0.06em]",
			state === "open" ? "bg-up/15 text-up"
				: state === "merged" ? "bg-accent/15 text-accent"
				: "bg-down/15 text-down",
		)}>
			{state}
		</span>
	);
}

export function DevProjectPage({ id }: { id: string }) {
	const [project, setProject] = useState<DevProject | undefined>();
	const [tab, setTab] = useState<Tab>("prs");
	const [state, setState] = useState<"open" | "merged" | "closed" | "all">("open");
	const [prs, setPrs] = useState<DevPR[]>([]);
	const [issues, setIssues] = useState<DevIssue[]>([]);
	const [days, setDays] = useState<DevDayCount[]>([]);
	const [loaded, setLoaded] = useState(false);
	const [commentFor, setCommentFor] = useState<{ kind: "issue" | "pr"; number: number } | null>(null);
	const [commentBody, setCommentBody] = useState("");

	const loadMeta = useCallback(async () => {
		const { data } = await api.GET("/api/dev/projects/{id}", { params: { path: { id } } });
		if (data) setProject(data);
		const { data: d } = await api.GET("/api/dev/projects/{id}/contributions", { params: { path: { id } } });
		if (d) setDays(d);
	}, [id]);

	const loadList = useCallback(async () => {
		if (tab === "prs") {
			const { data } = await api.GET("/api/dev/projects/{id}/pullrequests", {
				params: { path: { id }, query: { state, limit: 200 } },
			});
			if (data) setPrs(data);
		} else {
			const { data } = await api.GET("/api/dev/projects/{id}/issues", {
				params: { path: { id }, query: { state, limit: 200 } },
			});
			if (data) setIssues(data);
		}
		setLoaded(true);
	}, [id, tab, state]);

	useEffect(() => { void loadMeta(); }, [loadMeta]);
	useEffect(() => { void loadList(); }, [loadList]);

	async function act(fn: () => Promise<unknown>) {
		await fn();
		await loadList();
		await loadMeta();
	}

	function comment() {
		const body = commentBody.trim();
		if (!commentFor || !body) return;
		const { kind, number } = commentFor;
		const path = kind === "pr"
			? "/api/dev/projects/{id}/pullrequests/{number}/comment" as const
			: "/api/dev/projects/{id}/issues/{number}/comment" as const;
		void act(() => api.POST(path, { params: { path: { id, number } }, body: { body } }));
		setCommentFor(null);
		setCommentBody("");
	}

	function CommentForm({ kind, number }: { kind: "issue" | "pr"; number: number }) {
		const open = commentFor?.kind === kind && commentFor.number === number;
		if (!open) return null;
		return (
			<div className="mt-2 flex gap-2">
				<textarea
					value={commentBody}
					onChange={(e) => setCommentBody(e.target.value)}
					placeholder={t("dev.commentPh")}
					rows={2}
					className="min-w-0 flex-1 resize-y rounded-[7px] border border-border bg-bg px-2.5 py-1.5 text-[12px] outline-none focus:border-border-strong"
				/>
				<Button size="sm" className="h-8 self-end" onClick={comment}>
					{t("dev.commentSend")}
				</Button>
			</div>
		);
	}

	const states: ("open" | "merged" | "closed" | "all")[] = tab === "prs" ? ["open", "merged", "closed", "all"] : ["open", "closed", "all"];

	return (
		<main className="mx-auto w-full max-w-5xl px-7 py-8">
			<Link href="/dev" className="mb-3 inline-flex items-center gap-1.5 text-[12px] text-text-faint hover:text-text">
				<ArrowLeft size={12} /> {t("dev.back")}
			</Link>

			{project && (
				<div className="mb-5 rounded-[10px] border border-border bg-surface p-4">
					<div className="flex items-center gap-2">
						<span className={cn("rounded-[5px] px-1.5 py-0.5 font-mono text-[9.5px] uppercase tracking-[0.08em]",
							project.provider === "github" ? "bg-up/15 text-up" : "bg-accent/15 text-accent")}>
							{project.provider}
						</span>
						<h1 className="min-w-0 flex-1 truncate text-[15px] font-semibold">{project.name || project.path}</h1>
						{project.meta?.defaultBranch && (
							<span className="font-mono text-[10.5px] text-text-faint">{project.meta.defaultBranch}</span>
						)}
						<Button variant="outline" size="icon" aria-label={t("dev.sync")}
							onClick={() => void api.POST("/api/dev/projects/{id}/sync", { params: { path: { id } } }).then(() => setTimeout(loadMeta, 2500))}>
							<RefreshCw size={12} />
						</Button>
						{project.meta?.homepage && (
							<a href={project.meta.homepage} target="_blank" rel="noreferrer"
								className="text-text-faint hover:text-text" aria-label="open repo">
								<ExternalLink size={13} />
							</a>
						)}
					</div>
					{project.meta?.description && (
						<p className="mt-1.5 text-[12.5px] text-text-dim">{project.meta.description}</p>
					)}
					<div className="mt-3">
						<Heatmap days={days} />
					</div>
					{project.lastSyncError && (
						<p className="mt-2 font-mono text-[10.5px] text-down">{project.lastSyncError}</p>
					)}
				</div>
			)}

			<div className="mb-3 flex items-center gap-2">
				<div className="flex rounded-[8px] border border-border p-0.5">
					{TABS.map((tb) => (
						<button key={tb} onClick={() => setTab(tb)}
							className={cn(
								"flex items-center gap-1.5 rounded-[6px] px-3 py-1.5 text-[12px]",
								tab === tb ? "bg-surface-hover text-text" : "text-text-faint hover:text-text",
							)}>
							{tb === "prs" ? <GitPullRequest size={12} /> : <Tag size={12} />}
							{tb === "prs" ? t("dev.tabPRs") : t("dev.tabIssues")}
						</button>
					))}
				</div>
				<Select value={state} onValueChange={(v) => setState(v === "merged" || v === "closed" || v === "all" ? v : "open")}>
					<SelectTrigger className="h-8 w-32"><SelectValue /></SelectTrigger>
					<SelectContent>
						{states.map((s) => <SelectItem key={s} value={s}>{s}</SelectItem>)}
					</SelectContent>
				</Select>
			</div>

			{tab === "prs" ? (
				<div className="overflow-hidden rounded-[10px] border border-border">
					{prs.length === 0 && loaded && (
						<p className="px-4 py-8 text-center text-[12.5px] text-text-faint">{t("dev.none")}</p>
					)}
					{prs.map((pr) => (
						<div key={pr.number ?? 0} className="border-b border-border/60 px-4 py-3 last:border-0 hover:bg-surface-hover/50">
							<div className="flex items-center gap-2.5">
								<StateChip state={pr.state ?? ""} />
								<a href={pr.url ?? "#"} target="_blank" rel="noreferrer"
									className="min-w-0 flex-1 truncate text-[13px] font-medium hover:underline">
									{pr.title}
								</a>
								{pr.draft && <span className="font-mono text-[9.5px] uppercase text-text-faint">draft</span>}
								{pr.reviewDecision && pr.reviewDecision !== "" && (
									<span className="font-mono text-[9.5px] uppercase text-text-faint">{pr.reviewDecision.replaceAll("_", " ")}</span>
								)}
							</div>
							<div className="mt-1 flex flex-wrap items-center gap-x-3 gap-y-0.5 font-mono text-[10.5px] text-text-faint">
								<span>#{pr.number}</span>
								<span>{pr.author}</span>
								<span className="text-up">+{pr.additions}</span>
								<span className="text-down">−{pr.deletions}</span>
								<span>{pr.sourceBranch} → {pr.targetBranch}</span>
								<span>{timeAgo(pr.updatedAt)}</span>
								<span className="ml-auto flex gap-1">
									{pr.state === "open" && (
										<>
											<Button variant="outline" size="sm" className="h-6 px-2 text-[10.5px]"
												onClick={() => {
													const method = confirm(t("dev.mergeSquashConfirm")) ? "squash" : "merge";
													void act(() => api.POST("/api/dev/projects/{id}/pullrequests/{number}/merge",
														{ params: { path: { id, number: pr.number ?? 0 } }, body: { method } }));
												}}>
												<GitMerge size={10} /> {t("dev.merge")}
											</Button>
											<Button variant="outline" size="sm" className="h-6 px-2 text-[10.5px]"
												onClick={() => {
													if (confirm(t("dev.closeConfirm", { n: pr.number ?? 0 }))) {
														void act(() => api.POST("/api/dev/projects/{id}/pullrequests/{number}/close",
															{ params: { path: { id, number: pr.number ?? 0 } } }));
													}
												}}>
												{t("dev.close")}
											</Button>
										</>
									)}
									{pr.state === "closed" && (
										<Button variant="outline" size="sm" className="h-6 px-2 text-[10.5px]"
											onClick={() => {
												if (confirm(t("dev.reopenConfirm", { n: pr.number ?? 0 }))) {
													void act(() => api.POST("/api/dev/projects/{id}/pullrequests/{number}/reopen",
														{ params: { path: { id, number: pr.number ?? 0 } } }));
												}
											}}>
											<RotateCcw size={10} /> {t("dev.reopen")}
										</Button>
									)}
									<Button variant="outline" size="sm" className="h-6 px-2 text-[10.5px]"
										onClick={() => { setCommentFor({ kind: "pr", number: pr.number ?? 0 }); setCommentBody(""); }}>
										<MessageSquare size={10} /> {t("dev.comment")}
									</Button>
								</span>
							</div>
							<CommentForm kind="pr" number={pr.number ?? 0} />
						</div>
					))}
				</div>
			) : (
				<div className="overflow-hidden rounded-[10px] border border-border">
					{issues.length === 0 && loaded && (
						<p className="px-4 py-8 text-center text-[12.5px] text-text-faint">{t("dev.none")}</p>
					)}
					{issues.map((i) => (
						<div key={i.number ?? 0} className="border-b border-border/60 px-4 py-3 last:border-0 hover:bg-surface-hover/50">
							<div className="flex items-center gap-2.5">
								<StateChip state={i.state ?? ""} />
								<a href={i.url ?? "#"} target="_blank" rel="noreferrer"
									className="min-w-0 flex-1 truncate text-[13px] font-medium hover:underline">
									{i.title}
								</a>
								{i.labels?.map((l) => (
									<span key={l} className="rounded-[5px] bg-surface-hover px-1.5 py-0.5 font-mono text-[9.5px] text-text-faint">{l}</span>
								))}
							</div>
							<div className="mt-1 flex items-center gap-3 font-mono text-[10.5px] text-text-faint">
								<span>#{i.number}</span>
								<span>{i.author}</span>
								{i.comments ? <span>{i.comments} comments</span> : null}
								<span>{timeAgo(i.updatedAt)}</span>
								<span className="ml-auto flex gap-1">
									{i.state === "open" && (
										<Button variant="outline" size="sm" className="h-6 px-2 text-[10.5px]"
											onClick={() => {
												if (confirm(t("dev.closeConfirm", { n: i.number ?? 0 }))) {
													void act(() => api.POST("/api/dev/projects/{id}/issues/{number}/close",
														{ params: { path: { id, number: i.number ?? 0 } } }));
												}
											}}>
											{t("dev.close")}
										</Button>
									)}
									{i.state === "closed" && (
										<Button variant="outline" size="sm" className="h-6 px-2 text-[10.5px]"
											onClick={() => {
												if (confirm(t("dev.reopenConfirm", { n: i.number ?? 0 }))) {
													void act(() => api.POST("/api/dev/projects/{id}/issues/{number}/reopen",
														{ params: { path: { id, number: i.number ?? 0 } } }));
												}
											}}>
											<RotateCcw size={10} /> {t("dev.reopen")}
										</Button>
									)}
									<Button variant="outline" size="sm" className="h-6 px-2 text-[10.5px]"
										onClick={() => { setCommentFor({ kind: "issue", number: i.number ?? 0 }); setCommentBody(""); }}>
										<MessageSquare size={10} /> {t("dev.comment")}
									</Button>
								</span>
							</div>
							<CommentForm kind="issue" number={i.number ?? 0} />
						</div>
					))}
				</div>
			)}
		</main>
	);
}
