import { useEffect, useState } from "react";
import { LayoutGrid, Moon, Plus, Search, Settings, Sun } from "lucide-react";
import { BoardProvider, useBoard } from "@/board/store";
import { Board } from "@/board/Board";
import { ServiceDialog } from "@/components/ServiceDialog";
import { WidgetDialog } from "@/components/WidgetDialog";
import { SettingsDialog } from "@/components/SettingsDialog";
import { CommandPalette } from "@/components/CommandPalette";
import { Button } from "@/components/ui/button";
import type { Item } from "@/types";

export default function App() {
	return (
		<BoardProvider>
			<Shell />
		</BoardProvider>
	);
}

function Shell() {
	const board = useBoard();
	const [settingsOpen, setSettingsOpen] = useState(false);
	const [paletteOpen, setPaletteOpen] = useState(false);
	const [svcOpen, setSvcOpen] = useState(false);
	const [wdgOpen, setWdgOpen] = useState(false);
	const [editing, setEditing] = useState<Item | undefined>();

	// ⌘K / Ctrl+K opens the palette; / does too when not typing in a field.
	useEffect(() => {
		const onKey = (e: KeyboardEvent) => {
			const typing = e.target instanceof HTMLElement
				&& (e.target.tagName === "INPUT" || e.target.tagName === "TEXTAREA" || e.target.isContentEditable);
			if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "k") {
				e.preventDefault();
				setPaletteOpen((o) => !o);
			} else if (e.key === "/" && !typing && !e.metaKey && !e.ctrlKey) {
				e.preventDefault();
				setPaletteOpen(true);
			}
		};
		window.addEventListener("keydown", onKey);
		return () => window.removeEventListener("keydown", onKey);
	}, []);

	function openEditor(it: Item) {
		setEditing(it);
		if (it.kind === "widget") setWdgOpen(true);
		else setSvcOpen(true);
	}

	return (
		<div className="font-sans">
			<header className="sticky top-0 z-10 flex items-center gap-3.5 border-b border-border bg-bg/80 px-7 py-3.5 backdrop-blur-md">
				<div className="flex items-center gap-2.5 text-[15px] font-semibold tracking-tight">
					<svg width="18" height="18" viewBox="0 0 18 18" fill="none" aria-hidden>
						<rect x="1" y="1" width="7" height="7" rx="1.5" fill="currentColor" />
						<rect x="10" y="1" width="7" height="7" rx="1.5" fill="currentColor" opacity=".45" />
						<rect x="1" y="10" width="7" height="7" rx="1.5" fill="currentColor" opacity=".45" />
						<rect x="10" y="10" width="7" height="7" rx="1.5" fill="currentColor" />
					</svg>
					Dash
				</div>
				<div className="ml-auto flex items-center gap-2">
					<Button
						variant="outline" aria-label="Search (⌘K)"
						onClick={() => setPaletteOpen(true)}
						className="gap-1.5 text-text-faint"
					>
						<Search size={14} strokeWidth={1.8} />
						<kbd className="hidden font-mono text-[10px] min-[620px]:inline">⌘K</kbd>
					</Button>
					<Button
						variant="outline" size="icon" aria-label="Settings"
						onClick={() => setSettingsOpen(true)}
					>
						<Settings size={15} strokeWidth={1.8} />
					</Button>
					<Button
						variant="outline" size="icon" aria-label="Toggle theme"
						onClick={() => void board.setSetting("theme", board.theme === "dark" ? "light" : "dark")}
					>
						{board.theme === "dark" ? <Moon size={15} strokeWidth={1.8} /> : <Sun size={15} strokeWidth={1.8} />}
					</Button>
					<Button
						variant="outline" aria-label="Add widget"
						onClick={() => { setEditing(undefined); setWdgOpen(true); }}
					>
						<LayoutGrid size={13} strokeWidth={2} />
						Widget
					</Button>
					<Button onClick={() => { setEditing(undefined); setSvcOpen(true); }}>
						<Plus size={13} strokeWidth={2.2} />
						Add service
					</Button>
				</div>
			</header>

			<Board onEditItem={openEditor} />

			<ServiceDialog open={svcOpen} onOpenChange={setSvcOpen} item={editing?.kind === "service" ? editing : undefined} />
			<WidgetDialog open={wdgOpen} onOpenChange={setWdgOpen} item={editing?.kind === "widget" ? editing : undefined} />
			<SettingsDialog open={settingsOpen} onOpenChange={setSettingsOpen} />
			<CommandPalette
				open={paletteOpen}
				onOpenChange={setPaletteOpen}
				onAddService={() => { setEditing(undefined); setSvcOpen(true); }}
				onAddWidget={() => { setEditing(undefined); setWdgOpen(true); }}
				onSettings={() => setSettingsOpen(true)}
			/>
		</div>
	);
}
