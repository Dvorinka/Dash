import { useEffect, useState } from "react";
import { api } from "@/api";
import type { Monitor, MonitorInput } from "@/types";
import { Button } from "@/components/ui/button";
import {
	Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { t } from "@/i18n";

const TYPES = ["http", "tcp", "ping", "dns", "keyword", "json", "push"] as const;
type MonType = (typeof TYPES)[number];

// One dialog for add + edit. The type picker drives which fields matter —
// same pattern as ServiceDialog.
export function MonitorDialog({
	open,
	onOpenChange,
	monitor,
	onSaved,
}: {
	open: boolean;
	onOpenChange: (o: boolean) => void;
	monitor?: Monitor | undefined;
	onSaved: () => void;
}) {
	const editing = Boolean(monitor);
	const [f, setF] = useState<Record<string, string>>({});
	const [busy, setBusy] = useState(false);
	const [err, setErr] = useState("");

	useEffect(() => {
		if (!open) return;
		setErr("");
		setF({
			name: monitor?.name ?? "",
			type: monitor?.type ?? "http",
			url: monitor?.url ?? "",
			hostname: monitor?.hostname ?? "",
			port: monitor?.port ? String(monitor.port) : "",
			method: monitor?.method ?? "GET",
			keyword: monitor?.keyword ?? "",
			keywordInvert: monitor?.keywordInvert ? "1" : "",
			jsonQuery: monitor?.jsonQuery ?? "",
			expected: monitor?.expected ?? "",
			dnsType: monitor?.dnsType ?? "A",
			intervalS: String(monitor?.intervalS ?? 60),
			timeoutS: String(monitor?.timeoutS ?? 10),
			retries: String(monitor?.retries ?? 0),
			consecFails: String(monitor?.alerts?.consecutiveFailures ?? ""),
			latencyWarn: String(monitor?.alerts?.latencyWarnMs ?? ""),
			mute: monitor?.alerts?.mute ? "1" : "",
			notes: monitor?.notes ?? "",
		});
	}, [open, monitor]);

	const set = (k: string) => (e: React.ChangeEvent<HTMLInputElement>) =>
		setF((s) => ({ ...s, [k]: e.target.value }));
	// SAFETY: f.type only ever receives values from the TYPES select or a
	// persisted monitor row, both constrained to MonType.
	const type = (f.type || "http") as MonType;

	async function submit() {
		if (!f.name?.trim()) return setErr(t("monitors.nameRequired")), undefined;
		if (type !== "push" && type !== "tcp" && type !== "ping" && type !== "dns" && !f.url?.trim())
			return setErr(t("monitors.urlRequired")), undefined;
		if ((type === "tcp" || type === "ping" || type === "dns") && !f.hostname?.trim() && !f.url?.trim())
			return setErr(t("monitors.hostnameRequired")), undefined;
		setBusy(true);
		const body: MonitorInput = {
			name: f.name.trim(),
			type,
			url: f.url?.trim() ?? "",
			hostname: f.hostname?.trim() ?? "",
			port: parseInt(f.port ?? "0") || 0,
			method: f.method || "GET",
			keyword: f.keyword ?? "",
			keywordInvert: f.keywordInvert === "1",
			jsonQuery: f.jsonQuery ?? "",
			expected: f.expected ?? "",
			dnsType: f.dnsType || "A",
			intervalS: Math.max(10, parseInt(f.intervalS ?? "60") || 60),
			timeoutS: Math.max(1, parseInt(f.timeoutS ?? "10") || 10),
			retries: Math.max(0, parseInt(f.retries ?? "0") || 0),
			notes: f.notes ?? "",
			alerts: {
				consecutiveFailures: Math.max(1, parseInt(f.consecFails ?? "") || 1),
				latencyWarnMs: Math.max(0, parseInt(f.latencyWarn ?? "") || 0),
				mute: f.mute === "1",
			},
		};
		try {
			if (editing && monitor) {
				await api.PATCH("/api/monitors/{id}", { params: { path: { id: monitor.id! } }, body });
			} else {
				await api.POST("/api/monitors", { body });
			}
			onSaved();
			onOpenChange(false);
		} catch {
			setErr(t("monitors.saveFailed"));
		} finally {
			setBusy(false);
		}
	}

	return (
		<Dialog open={open} onOpenChange={onOpenChange}>
			<DialogContent className="max-w-md">
				<DialogHeader>
					<DialogTitle>{editing ? t("monitors.editTitle") : t("monitors.newTitle")}</DialogTitle>
					<DialogDescription>{t("monitors.desc")}</DialogDescription>
				</DialogHeader>

				<div className="flex max-h-[60vh] flex-col gap-4 overflow-y-auto pr-1">
					<div className="grid grid-cols-[1fr_130px] gap-3">
						<div className="flex flex-col gap-1.5">
							<Label htmlFor="mon-name">{t("common.name")}</Label>
							<Input id="mon-name" value={f.name ?? ""} onChange={set("name")} placeholder="Plex" autoFocus />
						</div>
						<div className="flex flex-col gap-1.5">
							<Label>{t("common.type")}</Label>
							<Select value={type} onValueChange={(v) => setF((s) => ({ ...s, type: v }))}>
								<SelectTrigger><SelectValue /></SelectTrigger>
								<SelectContent>
									{TYPES.map((t) => <SelectItem key={t} value={t}>{t}</SelectItem>)}
								</SelectContent>
							</Select>
						</div>
					</div>

					{(type === "http" || type === "keyword" || type === "json") && (
						<>
							<div className="grid grid-cols-[1fr_110px] gap-3">
								<div className="flex flex-col gap-1.5">
									<Label htmlFor="mon-url">{t("common.url")}</Label>
									<Input id="mon-url" value={f.url ?? ""} onChange={set("url")} placeholder="https://plex.local" className="font-mono text-[12px]" />
								</div>
								<div className="flex flex-col gap-1.5">
									<Label>{t("monitors.method")}</Label>
									<Select value={f.method || "GET"} onValueChange={(v) => setF((s) => ({ ...s, method: v }))}>
										<SelectTrigger><SelectValue /></SelectTrigger>
										<SelectContent>
											{["GET", "HEAD", "POST"].map((m) => <SelectItem key={m} value={m}>{m}</SelectItem>)}
										</SelectContent>
									</Select>
								</div>
							</div>
							{type === "keyword" && (
								<div className="flex flex-col gap-1.5">
									<Label htmlFor="mon-kw">{t("monitors.keyword")}</Label>
									<Input id="mon-kw" value={f.keyword ?? ""} onChange={set("keyword")} placeholder="text that must appear" className="font-mono text-[12px]" />
									<label className="flex items-center gap-2 text-[12px] text-text-dim">
										<input type="checkbox" checked={f.keywordInvert === "1"}
											onChange={(e) => setF((s) => ({ ...s, keywordInvert: e.target.checked ? "1" : "" }))} />
										{t("monitors.keywordInvert")}
									</label>
								</div>
							)}
							{type === "json" && (
								<div className="grid grid-cols-2 gap-3">
									<div className="flex flex-col gap-1.5">
										<Label htmlFor="mon-jq">{t("monitors.jsonPath")}</Label>
										<Input id="mon-jq" value={f.jsonQuery ?? ""} onChange={set("jsonQuery")} placeholder="data.status" className="font-mono text-[12px]" />
									</div>
									<div className="flex flex-col gap-1.5">
										<Label htmlFor="mon-exp">{t("monitors.expected")}</Label>
										<Input id="mon-exp" value={f.expected ?? ""} onChange={set("expected")} placeholder="ok" className="font-mono text-[12px]" />
									</div>
								</div>
							)}
						</>
					)}

					{(type === "tcp" || type === "ping" || type === "dns") && (
						<div className="grid grid-cols-[1fr_110px] gap-3">
							<div className="flex flex-col gap-1.5">
								<Label htmlFor="mon-host">{t("common.hostname")}</Label>
								<Input id="mon-host" value={f.hostname ?? ""} onChange={set("hostname")} placeholder="nas.local" className="font-mono text-[12px]" />
							</div>
							{type === "tcp" ? (
								<div className="flex flex-col gap-1.5">
									<Label htmlFor="mon-port">{t("common.port")}</Label>
									<Input id="mon-port" value={f.port ?? ""} onChange={set("port")} placeholder="443" className="font-mono text-[12px]" />
								</div>
							) : type === "dns" ? (
								<div className="flex flex-col gap-1.5">
									<Label>{t("monitors.record")}</Label>
									<Select value={f.dnsType || "A"} onValueChange={(v) => setF((s) => ({ ...s, dnsType: v }))}>
										<SelectTrigger><SelectValue /></SelectTrigger>
										<SelectContent>
											{["A", "AAAA", "CNAME", "MX", "NS", "TXT"].map((t) => <SelectItem key={t} value={t}>{t}</SelectItem>)}
										</SelectContent>
									</Select>
								</div>
							) : <div />}
						</div>
					)}
					{type === "dns" && (
						<div className="flex flex-col gap-1.5">
							<Label htmlFor="mon-dnsexp">{t("monitors.dnsExpected")}</Label>
							<Input id="mon-dnsexp" value={f.expected ?? ""} onChange={set("expected")} className="font-mono text-[12px]" />
						</div>
					)}

					{type === "push" && (
						<p className="rounded-[8px] border border-border bg-surface px-3 py-2 text-[12px] text-text-dim">
							{t("monitors.pushHint")}
						</p>
					)}

					<div className="grid grid-cols-3 gap-3">
						<div className="flex flex-col gap-1.5">
							<Label htmlFor="mon-int">{t("monitors.interval")}</Label>
							<Input id="mon-int" value={f.intervalS ?? "60"} onChange={set("intervalS")} className="font-mono text-[12px]" />
						</div>
						<div className="flex flex-col gap-1.5">
							<Label htmlFor="mon-to">{t("monitors.timeout")}</Label>
							<Input id="mon-to" value={f.timeoutS ?? "10"} onChange={set("timeoutS")} className="font-mono text-[12px]" />
						</div>
						<div className="flex flex-col gap-1.5">
							<Label htmlFor="mon-re">{t("monitors.retries")}</Label>
							<Input id="mon-re" value={f.retries ?? "0"} onChange={set("retries")} className="font-mono text-[12px]" />
						</div>
					</div>

					<div className="grid grid-cols-3 gap-3">
						<div className="flex flex-col gap-1.5">
							<Label htmlFor="mon-cf">{t("monitors.failsBeforeDown")}</Label>
							<Input id="mon-cf" value={f.consecFails ?? ""} onChange={set("consecFails")} placeholder="1" className="font-mono text-[12px]" />
						</div>
						<div className="flex flex-col gap-1.5">
							<Label htmlFor="mon-lw">{t("monitors.latencyWarn")}</Label>
							<Input id="mon-lw" value={f.latencyWarn ?? ""} onChange={set("latencyWarn")} placeholder="off" className="font-mono text-[12px]" />
						</div>
						<div className="flex items-end pb-2">
							<label className="flex items-center gap-2 text-[12px] text-text-dim">
								<input type="checkbox" checked={f.mute === "1"}
									onChange={(e) => setF((s) => ({ ...s, mute: e.target.checked ? "1" : "" }))} />
								{t("monitors.mute")}
							</label>
						</div>
					</div>

					{monitor?.pushToken ? (
						<div className="flex flex-col gap-1.5">
							<Label>{t("monitors.pushUrl")}</Label>
							<code className="rounded-[8px] border border-border bg-surface px-3 py-2 font-mono text-[11px] text-text-dim">
								{location.origin}/api/push/{monitor.pushToken}
							</code>
						</div>
					) : null}

					{err ? <p className="text-[12px] text-destructive">{err}</p> : null}
				</div>

				<DialogFooter className="items-center">
					<Button type="button" variant="outline" onClick={() => onOpenChange(false)} disabled={busy}>
						{t("common.cancel")}
					</Button>
					<Button type="button" onClick={() => void submit()} disabled={busy}>
						{busy ? t("common.saving") : editing ? t("common.save") : t("monitors.addMonitor")}
					</Button>
				</DialogFooter>
			</DialogContent>
		</Dialog>
	);
}
