import { useEffect, useState } from "react";
import { useBoard } from "@/board/store";
import { api } from "@/api";
import type { Item } from "@/types";
import { Button } from "@/components/ui/button";
import {
	Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import type { components } from "@dash/api-client";
import { t } from "@/i18n";

type WidgetType = components["schemas"]["WidgetType"];

// One generic dialog for every widget type — the registry's field list
// drives the inputs, so a new integration needs no dialog code.
export function WidgetDialog({
	open,
	onOpenChange,
	item,
}: {
	open: boolean;
	onOpenChange: (o: boolean) => void;
	item?: Item | undefined;
}) {
	const board = useBoard();
	const editing = Boolean(item);

	const [types, setTypes] = useState<WidgetType[]>([]);
	const [type, setType] = useState("");
	const [name, setName] = useState("");
	const [sectionId, setSectionId] = useState("");
	const [values, setValues] = useState<Record<string, string>>({});
	const [monitors, setMonitors] = useState<{ id?: string; name?: string }[]>([]);
	const [domains, setDomains] = useState<{ id?: string; name?: string }[]>([]);
	const [systems, setSystems] = useState<{ id?: string; name?: string }[]>([]);
	const [busy, setBusy] = useState(false);
	const [err, setErr] = useState("");

	useEffect(() => {
		if (!open) return;
		setErr("");
		// SAFETY: raw fetch — the generated client types only cover typed endpoints we call through api.ts.
		fetch("/api/widgets/types")
			.then((r) => r.json() as Promise<WidgetType[]>)
			.then(setTypes)
			.catch(() => setTypes([]));
		void api.GET("/api/monitors").then(({ data }) => setMonitors(data ?? []));
		void api.GET("/api/domains").then(({ data }) => setDomains(data ?? []));
		void api.GET("/api/systems").then(({ data }) => setSystems(data ?? []));

		const cfg = item?.config ?? null;
		setType(cfg && typeof cfg.type === "string" ? cfg.type : "");
		setName(item?.name ?? "");
		setSectionId(item?.sectionId ?? board.sections[0]?.id ?? "");
		const init: Record<string, string> = {};
		if (cfg) {
			for (const [k, v] of Object.entries(cfg)) {
				if (k !== "type" && typeof v === "string") init[k] = v;
			}
		}
		setValues(init);
	}, [open, item, board.sections]);

	const meta = types.find((t) => t.type === type);

	async function submit() {
		if (!type) return setErr(t("widget.pickType")), undefined;
		if (!name.trim()) return setErr(t("service.nameRequired")), undefined;
		if (meta?.fields.some((f) => f.required && !values[f.key]?.trim())) {
			return setErr(t("widget.requiredFields")), undefined;
		}
		setBusy(true);
		try {
			const config: NonNullable<Item["config"]> = { type };
			for (const f of meta?.fields ?? []) {
				const v = values[f.key]?.trim();
				if (v) config[f.key] = v;
			}
			if (editing && item) {
				await board.patchItem(item.id, { name: name.trim(), config });
			} else {
				let sid = sectionId;
				if (!sid) {
					const sec = await board.addSection(t("widget.defaultSection"));
					if (!sec) throw new Error("could not create section");
					sid = sec.id;
				}
				await board.addItem({ sectionId: sid, kind: "widget", name: name.trim(), config, urls: [] });
			}
			onOpenChange(false);
		} catch (e) {
			setErr(e instanceof Error ? e.message : t("service.saveFailed"));
		} finally {
			setBusy(false);
		}
	}

	return (
		<Dialog open={open} onOpenChange={onOpenChange}>
			<DialogContent>
				<DialogHeader>
					<DialogTitle>{editing ? t("widget.editTitle") : t("widget.newTitle")}</DialogTitle>
					<DialogDescription>
						{editing ? t("widget.editDesc") : t("widget.newDesc")}
					</DialogDescription>
				</DialogHeader>

				<div className="flex flex-col gap-4">
					<div className="grid grid-cols-[1fr_150px] gap-3">
						<div className="flex flex-col gap-1.5">
							<Label htmlFor="wdg-name">{t("common.name")}</Label>
							<Input id="wdg-name" value={name} onChange={(e) => setName(e.target.value)} placeholder="Pi-hole" autoFocus />
						</div>
						{!editing && board.sections.length > 0 && (
							<div className="flex flex-col gap-1.5">
								<Label>{t("widget.section")}</Label>
								<Select value={sectionId} onValueChange={setSectionId}>
									<SelectTrigger><SelectValue /></SelectTrigger>
									<SelectContent>
										{board.sections.map((s) => (
											<SelectItem key={s.id} value={s.id}>{s.name}</SelectItem>
										))}
									</SelectContent>
								</Select>
							</div>
						)}
					</div>

					<div className="flex flex-col gap-1.5">
						<Label>{t("common.type")}</Label>
						<Select value={type} onValueChange={setType}>
							<SelectTrigger><SelectValue placeholder={t("widget.pickOne")} /></SelectTrigger>
							<SelectContent>
								{types.map((t) => (
									<SelectItem key={t.type} value={t.type}>{t.name}</SelectItem>
								))}
							</SelectContent>
						</Select>
						{meta?.description ? (
							<p className="text-[11px] text-text-faint">{meta.description}</p>
						) : null}
					</div>

					{meta?.fields.map((f) => (
						<div key={f.key} className="flex flex-col gap-1.5">
							<Label htmlFor={`wdg-${f.key}`}>
								{f.label}{f.required ? "" : ` (${t("common.optional")})`}
							</Label>
							{f.key === "monitorId" || f.key === "domainId" || f.key === "systemId" ? (
								<Select
									value={values[f.key] ?? ""}
									onValueChange={(v) => setValues((s) => ({ ...s, [f.key]: v }))}
								>
									<SelectTrigger><SelectValue placeholder={t("widget.pickOne")} /></SelectTrigger>
									<SelectContent>
										{(f.key === "monitorId" ? monitors : f.key === "domainId" ? domains : systems).map((m) => (
											<SelectItem key={m.id} value={m.id ?? ""}>{m.name}</SelectItem>
										))}
									</SelectContent>
								</Select>
							) : (
								<Input
									id={`wdg-${f.key}`}
									type={f.secret ? "password" : "text"}
									value={values[f.key] ?? ""}
									placeholder={f.placeholder}
									onChange={(e) => setValues((v) => ({ ...v, [f.key]: e.target.value }))}
									className="font-mono text-[12px]"
								/>
							)}
						</div>
					))}

					{err ? <p className="text-[12px] text-destructive">{err}</p> : null}
				</div>

				<DialogFooter className="items-center">
					<Button type="button" variant="outline" onClick={() => onOpenChange(false)} disabled={busy}>
						{t("common.cancel")}
					</Button>
					<Button type="button" onClick={() => void submit()} disabled={busy}>
						{busy ? t("common.saving") : editing ? t("common.save") : t("header.addWidget")}
					</Button>
				</DialogFooter>
			</DialogContent>
		</Dialog>
	);
}
