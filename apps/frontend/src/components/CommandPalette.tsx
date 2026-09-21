import { useEffect, useMemo, useRef, useState } from "react";
import { useLocation } from "wouter";
import { Activity, AlertTriangle, ArrowRight, Download, Globe, LayoutGrid, Megaphone, Moon, Plus, Server, Settings, Sun } from "lucide-react";
import { useBoard } from "@/board/store";
import { hostOf, IconImg } from "@/board/primitives";
import { rendererLabels, rendererList } from "@/renderers";
import { cn } from "@/lib/utils";
import { Dialog, DialogContent, DialogTitle } from "@/components/ui/dialog";
import type { Item } from "@/types";

// ⌘K palette — flat filtered list over actions + service endpoints.
// No cmdk dep: an input, a list, and an active-index is the whole mechanic.

interface Command {
	id: string;
	group: "Actions" | "Services";
	icon?: Item;
	glyph?: typeof LayoutGrid;
	label: string;
	hint?: string | undefined;
	run(): void;
}

export function CommandPalette({
	open,
	onOpenChange,
	onAddService,
	onAddWidget,
	onSettings,
}: {
	open: boolean;
	onOpenChange: (o: boolean) => void;
	onAddService(): void;
	onAddWidget(): void;
	onSettings(): void;
}) {
	const board = useBoard();
	const [, navigate] = useLocation();
	const [query, setQuery] = useState("");
	const [active, setActive] = useState(0);
	const listRef = useRef<HTMLDivElement>(null);

	const commands = useMemo<Command[]>(() => {
		const cmds: Command[] = [
			{ id: "nav-board", group: "Actions", glyph: LayoutGrid, label: "Go to Board", run: () => navigate("/") },
			{ id: "nav-monitors", group: "Actions", glyph: Activity, label: "Go to Monitors", run: () => navigate("/monitors") },
			{ id: "nav-domains", group: "Actions", glyph: Globe, label: "Go to Domains", run: () => navigate("/domains") },
			{ id: "nav-systems", group: "Actions", glyph: Server, label: "Go to Systems", run: () => navigate("/systems") },
			{ id: "nav-incidents", group: "Actions", glyph: AlertTriangle, label: "Go to Incidents", run: () => navigate("/incidents") },
			{ id: "nav-status", group: "Actions", glyph: Megaphone, label: "Go to Status pages", run: () => navigate("/status") },
			{ id: "add-service", group: "Actions", glyph: Plus, label: "Add service", hint: "new", run: onAddService },
			{ id: "add-widget", group: "Actions", glyph: Plus, label: "Add widget", hint: "new", run: onAddWidget },
			{ id: "settings", group: "Actions", glyph: Settings, label: "Open settings", run: onSettings },
			{
				id: "theme", group: "Actions", glyph: board.theme === "dark" ? Sun : Moon,
				label: `Switch to ${board.theme === "dark" ? "light" : "dark"} theme`,
				run: () => void board.setSetting("theme", board.theme === "dark" ? "light" : "dark"),
			},
			...rendererList.map((r): Command => ({
				id: `renderer-${r}`, group: "Actions", glyph: LayoutGrid,
				label: `Renderer: ${rendererLabels[r]}`,
				hint: board.renderer === r ? "active" : undefined,
				run: () => void board.setSetting("renderer", r),
			})),
			{
				id: "export", group: "Actions", glyph: Download, label: "Export board JSON",
				run: () => { window.open("/api/export", "_blank", "noopener"); },
			},
		];
		for (const sec of board.sections) {
			for (const it of sec.items) {
				if (it.kind === "widget") continue;
				for (const u of it.urls) {
					cmds.push({
						id: `svc-${u.id}`, group: "Services", icon: it,
						label: u.label ? `${it.name} — ${u.label}` : it.name,
						hint: hostOf(u.url),
						run: () => window.open(u.url, "_blank", "noopener"),
					});
				}
			}
		}
		return cmds;
	}, [board.sections, board.theme, board.renderer, board.setSetting, onAddService, onAddWidget, onSettings, navigate]);

	const q = query.trim().toLowerCase();
	const filtered = q
		? commands.filter((c) => `${c.label} ${c.hint ?? ""}`.toLowerCase().includes(q))
		: commands;

	const clamped = Math.min(active, Math.max(0, filtered.length - 1));

	useEffect(() => {
		if (open) { setQuery(""); setActive(0); }
	}, [open]);

	useEffect(() => {
		listRef.current?.querySelector(`[data-idx="${clamped}"]`)?.scrollIntoView({ block: "nearest" });
	}, [clamped]);

	function pick(i: number) {
		const c = filtered[i];
		if (!c) return;
		onOpenChange(false);
		c.run();
	}

	const groups: { name: string; items: { c: Command; i: number }[] }[] = [];
	filtered.forEach((c, i) => {
		const g = groups.find((g) => g.name === c.group);
		if (g) g.items.push({ c, i });
		else groups.push({ name: c.group, items: [{ c, i }] });
	});

	return (
		<Dialog open={open} onOpenChange={onOpenChange}>
			<DialogContent className="top-[18%] max-w-[520px] translate-y-0 gap-0 overflow-hidden p-0" showCloseButton={false}>
				<DialogTitle className="sr-only">Command palette</DialogTitle>
				<div className="flex items-center gap-2.5 border-b border-border px-4">
					<input
						autoFocus
						value={query}
						onChange={(e) => { setQuery(e.target.value); setActive(0); }}
						onKeyDown={(e) => {
							if (e.key === "ArrowDown") { e.preventDefault(); setActive(clamped + 1); }
							else if (e.key === "ArrowUp") { e.preventDefault(); setActive(clamped - 1); }
							else if (e.key === "Enter") { e.preventDefault(); pick(clamped); }
						}}
						placeholder="Search services and actions…"
						className="h-12 w-full bg-transparent text-[14px] outline-none placeholder:text-text-faint focus-visible:outline-none"
					/>
					<kbd className="shrink-0 rounded border border-border px-1.5 py-0.5 font-mono text-[10px] text-text-faint">esc</kbd>
				</div>
				<div ref={listRef} className="max-h-[340px] overflow-y-auto p-1.5">
					{filtered.length === 0 ? (
						<p className="px-3 py-6 text-center text-[12.5px] text-text-faint">No matches.</p>
					) : groups.map((g) => (
						<div key={g.name}>
							<div className="px-2.5 pb-1 pt-2 font-mono text-[10px] uppercase tracking-[0.1em] text-text-faint">
								{g.name}
							</div>
							{g.items.map(({ c, i }) => (
								<button
									key={c.id}
									type="button"
									data-idx={i}
									onMouseEnter={() => setActive(i)}
									onClick={() => pick(i)}
									className={cn(
										"flex w-full items-center gap-2.5 rounded-[7px] px-2.5 py-2 text-left text-[13px]",
										i === clamped && "bg-surface-hover",
									)}
								>
									{c.icon ? (
										<IconImg item={c.icon} size={22} />
									) : c.glyph ? (
										<span className="flex size-[22px] items-center justify-center text-text-faint">
											<c.glyph size={13} />
										</span>
									) : null}
									<span className="min-w-0 truncate">{c.label}</span>
									{c.hint ? <span className="ml-auto truncate font-mono text-[10.5px] text-text-faint">{c.hint}</span> : null}
									{i === clamped ? <ArrowRight size={12} className="shrink-0 text-text-faint" /> : null}
								</button>
							))}
						</div>
					))}
				</div>
			</DialogContent>
		</Dialog>
	);
}
