import { useEffect, useRef, useState } from "react";
import { Plus, Trash2, Upload, X } from "lucide-react";
import { useBoard } from "@/board/store";
import type { Item, UrlInput } from "@/types";
import { Button } from "@/components/ui/button";
import {
	Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

// One dialog for both add and edit: item present = edit, absent = add.
export function ServiceDialog({
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

	const [name, setName] = useState("");
	const [icon, setIcon] = useState("");
	const [urls, setUrls] = useState<UrlInput[]>([{ url: "", label: "local" }]);
	const [sectionId, setSectionId] = useState("");
	const [busy, setBusy] = useState(false);
	const [err, setErr] = useState("");
	const fileRef = useRef<HTMLInputElement>(null);

	useEffect(() => {
		if (!open) return;
		setErr("");
		setName(item?.name ?? "");
		setIcon(item?.icon ?? "");
		setUrls(
			item && item.urls.length > 0
				? item.urls.map((u) => ({ url: u.url, label: u.label }))
				: [{ url: "", label: "local" }],
		);
		setSectionId(item?.sectionId ?? board.sections[0]?.id ?? "");
	}, [open, item, board.sections]);

	const setUrl = (i: number, patch: Partial<UrlInput>) =>
		setUrls((us) => us.map((u, j) => (j === i ? { ...u, ...patch } : u)));

	async function submit() {
		const cleanUrls = urls.filter((u) => u.url.trim() !== "");
		if (!name.trim()) return setErr("Name is required"), undefined;
		if (cleanUrls.some((u) => !/^https?:\/\/.+/.test(u.url))) {
			return setErr("URLs must start with http:// or https://"), undefined;
		}
		setBusy(true);
		try {
			const file = fileRef.current?.files?.[0];
			if (editing && item) {
				await board.patchItem(item.id, { name: name.trim(), icon, urls: cleanUrls });
				if (file) await board.uploadIcon(item.id, file);
			} else {
				let sid = sectionId;
				if (!sid) {
					// First service on an empty board — create the home section.
					const sec = await board.addSection("Services");
					if (!sec) throw new Error("could not create section");
					sid = sec.id;
				}
				const created = await board.addItem({
					sectionId: sid,
					name: name.trim(),
					icon,
					urls: cleanUrls,
				});
				if (created && file) await board.uploadIcon(created.id, file);
			}
			onOpenChange(false);
		} catch (e) {
			setErr(e instanceof Error ? e.message : "save failed");
		} finally {
			setBusy(false);
		}
	}

	async function remove() {
		if (!item) return;
		setBusy(true);
		await board.deleteItem(item.id);
		setBusy(false);
		onOpenChange(false);
	}

	return (
		<Dialog open={open} onOpenChange={onOpenChange}>
			<DialogContent>
				<DialogHeader>
					<DialogTitle>{editing ? "Edit service" : "Add service"}</DialogTitle>
					<DialogDescription>
						{editing ? "Update name, icon, or launch URLs." : "Name it, give it an icon, add one or more URLs."}
					</DialogDescription>
				</DialogHeader>

				<div className="flex flex-col gap-4">
					<div className="grid grid-cols-[1fr_150px] gap-3">
						<div className="flex flex-col gap-1.5">
							<Label htmlFor="svc-name">Name</Label>
							<Input id="svc-name" value={name} onChange={(e) => setName(e.target.value)} placeholder="Pi-hole" autoFocus />
						</div>
						{!editing && board.sections.length > 0 && (
							<div className="flex flex-col gap-1.5">
								<Label>Section</Label>
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
						<Label>URLs</Label>
						{urls.map((u, i) => (
							<div key={i} className="flex gap-2">
								<Input
									value={u.url}
									onChange={(e) => setUrl(i, { url: e.target.value })}
									placeholder="https://…"
									className="font-mono text-[12px]"
								/>
								<Input
									value={u.label ?? ""}
									onChange={(e) => setUrl(i, { label: e.target.value })}
									placeholder="label"
									className="w-[110px] shrink-0"
									list="url-labels"
								/>
								<Button
									type="button" variant="ghost" size="icon"
									aria-label="Remove URL"
									disabled={urls.length <= 1}
									onClick={() => setUrls((us) => us.filter((_, j) => j !== i))}
								>
									<X size={13} />
								</Button>
							</div>
						))}
						<datalist id="url-labels">
							<option value="local" /><option value="external" />
						</datalist>
						<Button
							type="button" variant="outline" size="sm"
							className="mt-1 w-fit"
							onClick={() => setUrls((us) => [...us, { url: "", label: "" }])}
						>
							<Plus size={12} /> Add URL
						</Button>
					</div>

					<div className="flex flex-col gap-1.5">
						<Label htmlFor="svc-icon">Icon URL</Label>
						<div className="flex gap-2">
							<Input id="svc-icon" value={icon} onChange={(e) => setIcon(e.target.value)} placeholder="https://…/icon.png" className="font-mono text-[12px]" />
							<Button type="button" variant="outline" size="icon" aria-label="Upload icon file" onClick={() => fileRef.current?.click()}>
								<Upload size={13} />
							</Button>
							<input
								ref={fileRef} type="file" accept="image/*" className="hidden"
								onChange={() => setErr("")}
							/>
						</div>
						<p className="text-[11px] text-text-faint">URL or uploaded file; blank shows a letter tile.</p>
					</div>

					{err ? <p className="text-[12px] text-destructive">{err}</p> : null}
				</div>

				<DialogFooter className="items-center">
					{editing ? (
						<Button type="button" variant="destructive" size="sm" className="mr-auto" onClick={() => void remove()} disabled={busy}>
							<Trash2 size={12} /> Delete
						</Button>
					) : null}
					<Button type="button" variant="outline" onClick={() => onOpenChange(false)} disabled={busy}>
						Cancel
					</Button>
					<Button type="button" onClick={() => void submit()} disabled={busy}>
						{busy ? "Saving…" : editing ? "Save" : "Add service"}
					</Button>
				</DialogFooter>
			</DialogContent>
		</Dialog>
	);
}
