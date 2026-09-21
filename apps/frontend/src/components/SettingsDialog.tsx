import { useRef, useState } from "react";
import { Download, Upload } from "lucide-react";
import { useBoard } from "@/board/store";
import { rendererDescriptions, rendererLabels, rendererList } from "@/renderers";
import { cn } from "@/lib/utils";
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
			setMsg(body?.error ?? "Import failed");
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
						<div className="grid grid-cols-2 gap-1.5" role="radiogroup" aria-label="Renderer">
							{rendererList.map((r) => (
								<button
									key={r}
									type="button"
									role="radio"
									aria-checked={board.renderer === r}
									onClick={() => void board.setSetting("renderer", r)}
									className={cn(
										"rounded-[8px] border px-3 py-2.5 text-left transition-colors",
										board.renderer === r
											? "border-border-strong bg-surface-hover"
											: "border-border hover:border-border-strong hover:bg-surface-hover",
									)}
								>
									<span className="block text-[12.5px] font-medium">{rendererLabels[r]}</span>
									<span className="block font-mono text-[10px] text-text-faint">{rendererDescriptions[r]}</span>
								</button>
							))}
						</div>
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
							ref={fileRef} type="file" accept=".json,.yml,.yaml" className="hidden"
							onChange={(e) => {
								const f = e.target.files?.[0];
								if (f) void importJson(f);
								e.target.value = "";
							}}
						/>
					</div>
					<p className="text-[11px] text-text-faint">Import accepts Dash exports plus Homepage, Homarr, and Dashy configs.</p>
					{msg ? <p className="text-[12px] text-destructive">{msg}</p> : null}
				</div>
			</DialogContent>
		</Dialog>
	);
}
