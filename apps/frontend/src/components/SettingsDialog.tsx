import { useEffect, useRef, useState } from "react";
import { Bell, Download, HardDrive, Image, LayoutGrid, Lock, Moon, Palette, Sun, Upload, type LucideIcon } from "lucide-react";
import { useBoard } from "@/board/store";
import { api } from "@/api";
import { rendererDescriptions, rendererIcons, rendererLabels, rendererList } from "@/renderers";
import { cn } from "@/lib/utils";
import { LOCALES, t } from "@/i18n";
import { Button } from "@/components/ui/button";
import {
	Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

// Notification transports — mirrors the backend registry. Config is stored
// under notify_<name> as a JSON object; legacy notify_webhook strings map to
// {url}.
const TRANSPORTS = {
	webhook:  { label: "Webhook",  fields: [{ k: "url", l: "URL", ph: "https://…" }] },
	slack:    { label: "Slack",    fields: [{ k: "url", l: "Webhook URL", ph: "https://hooks.slack.com/…" }] },
	discord:  { label: "Discord",  fields: [{ k: "url", l: "Webhook URL", ph: "https://discord.com/api/webhooks/…" }] },
	telegram: { label: "Telegram", fields: [
		{ k: "token", l: "Bot token", ph: "123456:ABC…", secret: true },
		{ k: "chat_id", l: "Chat ID", ph: "-100…" },
		{ k: "server", l: "API server (optional)", ph: "https://api.telegram.org" },
	] },
	gotify:   { label: "Gotify",   fields: [
		{ k: "server", l: "Server", ph: "https://gotify.local" },
		{ k: "token", l: "App token", secret: true },
	] },
	ntfy:     { label: "ntfy",     fields: [
		{ k: "server", l: "Server", ph: "https://ntfy.sh" },
		{ k: "topic", l: "Topic", ph: "dash-alerts" },
	] },
	smtp:     { label: "SMTP",     fields: [
		{ k: "host", l: "Host", ph: "smtp.local" },
		{ k: "port", l: "Port", ph: "587" },
		{ k: "user", l: "User (optional)" },
		{ k: "pass", l: "Password", secret: true },
		{ k: "from", l: "From", ph: "dash@local" },
		{ k: "to", l: "To", ph: "me@local, oncall@local" },
	] },
} as const;
type TransportName = keyof typeof TRANSPORTS;

// Group — a labelled settings section. Icon + mono header keeps the dialog
// scannable now that it holds five distinct concerns.
function Group({ icon: Icon, title, children }: { icon: LucideIcon; title: string; children: React.ReactNode }) {
	return (
		<section className="flex flex-col gap-2.5 border-t border-border/60 pt-4 first:border-0 first:pt-0">
			<h3 className="flex items-center gap-1.5 font-mono text-[10.5px] uppercase tracking-[0.12em] text-text-faint">
				<Icon size={11} /> {title}
			</h3>
			{children}
		</section>
	);
}

export function SettingsDialog({ open, onOpenChange }: { open: boolean; onOpenChange: (o: boolean) => void }) {
	const board = useBoard();
	const fileRef = useRef<HTMLInputElement>(null);
	const [msg, setMsg] = useState("");
	const [transport, setTransport] = useState<TransportName>("webhook");
	const [notifyCfg, setNotifyCfg] = useState<Record<string, Record<string, string>>>({});
	const [notifyMsg, setNotifyMsg] = useState("");
	const [auth, setAuth] = useState<{ enabled?: boolean; authed?: boolean; username?: string }>({});

	useEffect(() => {
		if (!open) return;
		void api.GET("/api/auth/session").then(({ data }) => setAuth(data ?? {}));
		void api.GET("/api/settings").then(({ data }) => {
			const cfg: Record<string, Record<string, string>> = {};
			for (const name of Object.keys(TRANSPORTS)) {
				const v = data?.["notify_" + name];
				if (typeof v === "string") cfg[name] = { url: v }; // legacy webhook
				else if (v !== null && typeof v === "object" && !Array.isArray(v)) {
					const obj: Record<string, string> = {};
					for (const [k, val] of Object.entries(v)) if (typeof val === "string") obj[k] = val;
					cfg[name] = obj;
				}
			}
			setNotifyCfg(cfg);
			setNotifyMsg("");
		});
	}, [open]);

	const fields = TRANSPORTS[transport].fields;
	const cfg = notifyCfg[transport] ?? {};
	const setField = (k: string, v: string) => {
		setNotifyCfg((s) => ({ ...s, [transport]: { ...s[transport], [k]: v } }));
		setNotifyMsg("");
	};

	async function saveNotify() {
		setNotifyMsg("");
		await api.PUT("/api/settings", { body: { ["notify_" + transport]: cfg } });
		setNotifyMsg(t("settings.saved"));
	}

	async function testNotify() {
		setNotifyMsg("");
		const { data, error } = await api.POST("/api/notify/test", { body: { transport } });
		const r = data?.results?.[transport];
		setNotifyMsg(error ? t("settings.failed") : r === "ok" ? t("settings.sent") : (r ?? t("settings.sent")));
	}

	// Enabling auth flips the whole app to the setup screen — reload so the
	// gate picks it up. Same on sign-out/disable.
	async function enableAuth() {
		await api.PUT("/api/settings", { body: { auth_enabled: true } });
		location.reload();
	}

	async function signOut() {
		await api.POST("/api/auth/logout");
		location.reload();
	}

	async function disableAuth() {
		await api.POST("/api/auth/disable");
		location.reload();
	}

	async function exportJson() {
		const res = await fetch("/api/export");
		if (!res.ok) return;
		const blob = await res.blob();
		const a = document.createElement("a");
		a.href = URL.createObjectURL(blob);
		a.download = "dash-export.json";
		a.click();
		URL.revokeObjectURL(a.href);
	}

	async function importJson(file: File) {
		setMsg("");
		// Dash exports are JSON with {version, sections}; anything else
		// (Homepage/Homarr/Dashy) goes to the external importer for sniffing.
		let target = "/api/import/external";
		try {
			const j: unknown = JSON.parse(await file.text());
			if (j !== null && typeof j === "object" && "version" in j && "sections" in j) {
				target = "/api/import";
			}
		} catch { /* not JSON — external importer handles YAML */ }
		const res = await fetch(target, { method: "POST", body: file });
		if (res.ok) {
			location.reload(); // import replaces state; a reload re-syncs everything
		} else {
			// SAFETY: error responses always carry {error: string} per openapi.yaml Error schema.
			const body = (await res.json().catch(() => null)) as { error?: string } | null;
			setMsg(body?.error ?? t("settings.importFailed"));
		}
	}

	return (
		<Dialog open={open} onOpenChange={onOpenChange}>
			<DialogContent className="max-w-xl">
				<DialogHeader>
					<DialogTitle>{t("settings.title")}</DialogTitle>
					<DialogDescription>{t("settings.desc")}</DialogDescription>
				</DialogHeader>

				<div className="flex max-h-[72vh] flex-col gap-5 overflow-y-auto pr-1">
					<Group icon={LayoutGrid} title={t("settings.renderer")}>
						<div className="grid grid-cols-2 gap-2" role="radiogroup" aria-label={t("settings.renderer")}>
							{rendererList.map((r) => {
								const Icon = rendererIcons[r];
								const active = board.renderer === r;
								return (
									<button
										key={r}
										type="button"
										role="radio"
										aria-checked={active}
										onClick={() => void board.setSetting("renderer", r)}
										className={cn(
											"flex items-center gap-3 rounded-[9px] border px-3 py-2.5 text-left transition-colors",
											active
												? "border-accent bg-accent/10"
												: "border-border hover:border-border-strong hover:bg-surface-hover",
										)}
									>
										<span className={cn(
											"flex size-8 shrink-0 items-center justify-center rounded-[7px] border",
											active ? "border-accent/40 bg-accent/15 text-accent" : "border-border bg-bg text-text-faint",
										)}>
											<Icon size={15} />
										</span>
										<span className="min-w-0">
											<span className="block text-[12.5px] font-medium">{rendererLabels[r]}</span>
											<span className="block truncate font-mono text-[10px] text-text-faint">{rendererDescriptions[r]}</span>
										</span>
									</button>
								);
							})}
						</div>
					</Group>

					<Group icon={Palette} title={t("settings.appearance")}>
						<div className="grid grid-cols-2 gap-3 max-[560px]:grid-cols-1">
							<div className="flex flex-col gap-1.5">
								<Label>{t("settings.theme")}</Label>
								<div className="flex overflow-hidden rounded-[7px] border border-border">
									{(["dark", "light"] as const).map((th) => (
										<button
											key={th}
											type="button"
											aria-pressed={board.theme === th}
											onClick={() => void board.setSetting("theme", th)}
											className={cn(
												"flex flex-1 items-center justify-center gap-1.5 px-3 py-1.5 text-[12px] transition-colors",
												board.theme === th ? "bg-surface-hover text-text" : "text-text-faint hover:text-text",
											)}
										>
											{th === "dark" ? <Moon size={12} /> : <Sun size={12} />}
											{th === "dark" ? t("settings.dark") : t("settings.light")}
										</button>
									))}
								</div>
							</div>
							<div className="flex flex-col gap-1.5">
								<Label>{t("settings.language")}</Label>
								<Select
									value={board.locale || "en"}
									onValueChange={(v) => void board.setSetting("locale", v)}
								>
									<SelectTrigger><SelectValue /></SelectTrigger>
									<SelectContent>
										{Object.entries(LOCALES).map(([k, name]) => (
											<SelectItem key={k} value={k}>{name}</SelectItem>
										))}
									</SelectContent>
								</Select>
							</div>
						</div>
						<div className="flex flex-col gap-1.5">
							<Label htmlFor="accent">{t("settings.accent")}</Label>
							<div className="flex items-center gap-2">
								<input
									id="accent" type="color" aria-label={t("settings.accent")}
									value={/^#[0-9a-fA-F]{6}$/.test(board.accent) ? board.accent : "#3b82f6"}
									onChange={(e) => void board.setSetting("accent", e.target.value)}
									className="h-8 w-12 cursor-pointer rounded-[6px] border border-border bg-surface p-0.5"
								/>
								<span className="font-mono text-[11px] text-text-faint">{board.accent || "—"}</span>
								<Button type="button" variant="outline" size="sm" onClick={() => void board.setSetting("accent", "")}>
									{t("common.reset")}
								</Button>
							</div>
						</div>
					</Group>

					<Group icon={Image} title={t("settings.wallpaper")}>
						<Input
							id="wallpaper"
							value={board.wallpaper}
							onChange={(e) => void board.setSetting("wallpaper", e.target.value)}
							placeholder={t("settings.wallpaperPh")}
							className="font-mono text-[12px]"
						/>
						<textarea
							id="custom-css"
							aria-label={t("settings.customCss")}
							value={board.customCss}
							onChange={(e) => void board.setSetting("customCss", e.target.value)}
							placeholder=".item-tile { border-radius: 0; }"
							rows={3}
							className="w-full resize-y rounded-[7px] border border-border bg-bg px-2.5 py-2 font-mono text-[11.5px] outline-none focus:border-border-strong"
						/>
						<p className="text-[11px] text-text-faint">{t("settings.customCssHint")}</p>
					</Group>

					<Group icon={Bell} title={t("settings.notifications")}>
						<div className="flex gap-2">
							<Select value={transport} onValueChange={(v) => {
								// SAFETY: Select only yields TRANSPORTS keys.
								setTransport(v as TransportName); setNotifyMsg("");
							}}>
								<SelectTrigger><SelectValue /></SelectTrigger>
								<SelectContent>
									{Object.entries(TRANSPORTS).map(([k, v]) => (
										<SelectItem key={k} value={k}>
											{v.label}{notifyCfg[k] ? " ·" : ""}
										</SelectItem>
									))}
								</SelectContent>
							</Select>
							<Button type="button" variant="outline" size="sm" onClick={() => void saveNotify()}>{t("common.save")}</Button>
							<Button
								type="button" variant="outline" size="icon" aria-label={t("settings.testNotify")}
								onClick={() => void testNotify()}
							>
								<Bell size={12} />
							</Button>
						</div>
						{fields.map((f) => (
							<div key={f.k} className="flex flex-col gap-1">
								<Label htmlFor={`ntf-${f.k}`} className="text-[11px] text-text-faint">{f.l}</Label>
								<Input
									id={`ntf-${f.k}`}
									type={"secret" in f && f.secret ? "password" : "text"}
									value={cfg[f.k] ?? ""}
									onChange={(e) => setField(f.k, e.target.value)}
									placeholder={"ph" in f ? f.ph : ""}
									className="font-mono text-[12px]"
								/>
							</div>
						))}
						<p className="text-[11px] text-text-faint">
							{t("settings.notifyFanout")}
						</p>
						{notifyMsg ? <p className="text-[11px] text-text-faint">{notifyMsg}</p> : null}
					</Group>

					<Group icon={Lock} title={t("auth.section")}>
						<div className="flex items-center justify-between rounded-lg border border-border px-3 py-2.5">
							<p className="text-[11px] text-text-faint">
								{auth.enabled ? `${t("auth.onHint")} ${auth.username ?? ""}` : t("auth.offHint")}
							</p>
							{auth.enabled ? (
								<div className="flex gap-2">
									<Button type="button" variant="outline" size="sm" onClick={() => void signOut()}>
										{t("auth.signOut")}
									</Button>
									<Button type="button" variant="outline" size="sm" onClick={() => void disableAuth()}>
										{t("auth.disable")}
									</Button>
								</div>
							) : (
								<Button type="button" variant="outline" size="sm" onClick={() => void enableAuth()}>
									{t("auth.enable")}
								</Button>
							)}
						</div>
					</Group>

					<Group icon={HardDrive} title={t("settings.backup")}>
						<div className="flex gap-2">
							<Button type="button" variant="outline" size="sm" onClick={() => void exportJson()}>
								<Download size={12} /> {t("settings.exportJson")}
							</Button>
							<Button type="button" variant="outline" size="sm" onClick={() => fileRef.current?.click()}>
								<Upload size={12} /> {t("settings.import")}
							</Button>
							<input
								ref={fileRef} type="file" accept=".json,.yml,.yaml" className="hidden"
								onChange={(e) => {
									const f = e.target.files?.[0];
									if (f) void importJson(f);
									e.target.value = "";
								}}
							/>
						</div>
						<p className="text-[11px] text-text-faint">{t("settings.importHint")}</p>
						{msg ? <p className="text-[12px] text-destructive">{msg}</p> : null}
					</Group>
				</div>
			</DialogContent>
		</Dialog>
	);
}
