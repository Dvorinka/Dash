import { useRef, useState } from "react";
import { Download, Upload } from "lucide-react";
import { useBoard } from "@/board/store";
import { rendererLabels, rendererList } from "@/renderers";
import { Button } from "@/components/ui/button";
import {
	Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle,
} from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

export function SettingsDialog({ open, onOpenChange }: { open: boolean; onOpenChange: (o: boolean) => void }) {
	const board = useBoard();
	const fileRef = useRef<HTMLInputElement>(null);
	const [msg, setMsg] = useState("");

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
		const res = await fetch("/api/import", { method: "POST", body: file });
		if (res.ok) {
			location.reload(); // import replaces state; a reload re-syncs everything
		} else {
			setMsg("Import failed — not a Dash export?");
		}
	}

	return (
		<Dialog open={open} onOpenChange={onOpenChange}>
			<DialogContent className="max-w-sm">
				<DialogHeader>
					<DialogTitle>Settings</DialogTitle>
					<DialogDescription>Board layout, theme, and backup.</DialogDescription>
				</DialogHeader>

				<div className="flex flex-col gap-4">
					<div className="flex flex-col gap-1.5">
						<Label>Renderer</Label>
						<Select value={board.renderer} onValueChange={(v) => {
							if (v === "bento" || v === "cards") void board.setSetting("renderer", v);
						}}>
							<SelectTrigger><SelectValue /></SelectTrigger>
							<SelectContent>
								{rendererList.map((r) => (
									<SelectItem key={r} value={r}>{rendererLabels[r]}</SelectItem>
								))}
							</SelectContent>
						</Select>
					</div>

					<div className="flex flex-col gap-1.5">
						<Label>Theme</Label>
						<Select value={board.theme} onValueChange={(v) => {
							if (v === "dark" || v === "light") void board.setSetting("theme", v);
						}}>
							<SelectTrigger><SelectValue /></SelectTrigger>
							<SelectContent>
								<SelectItem value="dark">Dark</SelectItem>
								<SelectItem value="light">Light</SelectItem>
							</SelectContent>
						</Select>
					</div>

					<div className="flex gap-2 pt-1">
						<Button type="button" variant="outline" size="sm" onClick={() => void exportJson()}>
							<Download size={12} /> Export JSON
						</Button>
						<Button type="button" variant="outline" size="sm" onClick={() => fileRef.current?.click()}>
							<Upload size={12} /> Import
						</Button>
						<input
							ref={fileRef} type="file" accept="application/json" className="hidden"
							onChange={(e) => {
								const f = e.target.files?.[0];
								if (f) void importJson(f);
								e.target.value = "";
							}}
						/>
					</div>
					{msg ? <p className="text-[12px] text-destructive">{msg}</p> : null}
				</div>
			</DialogContent>
		</Dialog>
	);
}
