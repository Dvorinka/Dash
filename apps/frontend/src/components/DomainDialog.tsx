import { useEffect, useState } from "react";
import { api } from "@/api";
import type { Domain, DomainInput } from "@/types";
import { Button } from "@/components/ui/button";
import {
	Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { t } from "@/i18n";

// Add/edit a tracked domain. The first WHOIS/DNS/TLS lookup runs on create —
// expect a few seconds of spinner.
export function DomainDialog({
	open,
	onOpenChange,
	domain,
	onSaved,
}: {
	open: boolean;
	onOpenChange: (o: boolean) => void;
	domain?: Domain | undefined;
	onSaved: () => void;
}) {
	const editing = Boolean(domain);
	const [name, setName] = useState("");
	const [intervalH, setIntervalH] = useState("24");
	const [alertDays, setAlertDays] = useState("30");
	const [autoRenew, setAutoRenew] = useState(false);
	const [certDays, setCertDays] = useState("");
	const [mute, setMute] = useState(false);
	const [notes, setNotes] = useState("");
	const [busy, setBusy] = useState(false);
	const [err, setErr] = useState("");

	useEffect(() => {
		if (!open) return;
		setErr("");
		setName(domain?.name ?? "");
		setIntervalH(String(domain?.intervalH ?? 24));
		setAlertDays(String(domain?.alertDaysBefore ?? 30));
		setAutoRenew(domain?.autoRenew ?? false);
		setCertDays(domain?.alerts?.certDays ? String(domain.alerts.certDays) : "");
		setMute(domain?.alerts?.mute ?? false);
		setNotes(domain?.notes ?? "");
	}, [open, domain]);

	async function submit() {
		if (!name.trim()) return setErr(t("domains.nameRequired")), undefined;
		setBusy(true);
		const body: DomainInput = {
			name: name.trim(),
			intervalH: Math.max(1, parseInt(intervalH) || 24),
			alertDaysBefore: Math.max(0, parseInt(alertDays) || 30),
			autoRenew,
			notes,
			alerts: {
				certDays: Math.max(0, parseInt(certDays) || 0),
				mute,
			},
		};
		try {
			if (editing && domain) {
				await api.PATCH("/api/domains/{id}", { params: { path: { id: domain.id! } }, body });
			} else {
				const { error } = await api.POST("/api/domains", { body });
				if (error) throw new Error("create failed");
			}
			onSaved();
			onOpenChange(false);
		} catch (e) {
			setErr(e instanceof Error ? e.message : t("domains.saveFailed"));
		} finally {
			setBusy(false);
		}
	}

	return (
		<Dialog open={open} onOpenChange={onOpenChange}>
			<DialogContent className="max-w-md">
				<DialogHeader>
					<DialogTitle>{editing ? t("domains.editTitle") : t("domains.addTitle")}</DialogTitle>
					<DialogDescription>
						{t("domains.addDesc")}
					</DialogDescription>
				</DialogHeader>

				<div className="flex flex-col gap-4">
					<div className="flex flex-col gap-1.5">
						<Label htmlFor="dom-name">{t("domains.domain")}</Label>
						<Input id="dom-name" value={name} onChange={(e) => setName(e.target.value)}
							placeholder={t("domains.namePlaceholder")} className="font-mono text-[12px]" autoFocus />
					</div>

					<div className="grid grid-cols-2 gap-3">
						<div className="flex flex-col gap-1.5">
							<Label htmlFor="dom-int">{t("domains.refreshEvery")}</Label>
							<Input id="dom-int" value={intervalH} onChange={(e) => setIntervalH(e.target.value)} className="font-mono text-[12px]" />
						</div>
						<div className="flex flex-col gap-1.5">
							<Label htmlFor="dom-alert">{t("domains.alertDays")}</Label>
							<Input id="dom-alert" value={alertDays} onChange={(e) => setAlertDays(e.target.value)} className="font-mono text-[12px]" />
						</div>
					</div>

					<div className="grid grid-cols-2 gap-3">
						<div className="flex flex-col gap-1.5">
							<Label htmlFor="dom-cert">{t("domains.certDaysLbl")}</Label>
							<Input id="dom-cert" value={certDays} onChange={(e) => setCertDays(e.target.value)}
								placeholder={t("domains.certDaysPlaceholder")} className="font-mono text-[12px]" />
						</div>
						<div className="flex items-end pb-2">
							<label className="flex items-center gap-2 text-[12px] text-text-dim">
								<input type="checkbox" checked={mute} onChange={(e) => setMute(e.target.checked)} />
								{t("monitors.mute")}
							</label>
						</div>
					</div>

					<label className="flex items-center gap-2 text-[12.5px] text-text-dim">
						<input type="checkbox" checked={autoRenew} onChange={(e) => setAutoRenew(e.target.checked)} />
						{t("domains.autoRenewLbl")}
					</label>

					<div className="flex flex-col gap-1.5">
						<Label htmlFor="dom-notes">{t("domains.notesOptional")}</Label>
						<Input id="dom-notes" value={notes} onChange={(e) => setNotes(e.target.value)} />
					</div>

					{err ? <p className="text-[12px] text-destructive">{err}</p> : null}
				</div>

				<DialogFooter className="items-center">
					<Button type="button" variant="outline" onClick={() => onOpenChange(false)} disabled={busy}>
						{t("common.cancel")}
					</Button>
					<Button type="button" onClick={() => void submit()} disabled={busy}>
						{busy ? t("domains.lookingUp") : editing ? t("common.save") : t("domains.new")}
					</Button>
				</DialogFooter>
			</DialogContent>
		</Dialog>
	);
}
