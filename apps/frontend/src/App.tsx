import { useState } from "react";
import { Moon, Plus, Settings, Sun } from "lucide-react";
import { BoardProvider, useBoard } from "@/board/store";
import { Board } from "@/board/Board";
import { ServiceDialog } from "@/components/ServiceDialog";
import { SettingsDialog } from "@/components/SettingsDialog";
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
	const [svcOpen, setSvcOpen] = useState(false);
	const [editing, setEditing] = useState<Item | undefined>();

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
					<Button onClick={() => { setEditing(undefined); setSvcOpen(true); }}>
						<Plus size={13} strokeWidth={2.2} />
						Add service
					</Button>
				</div>
			</header>

			<Board onEditItem={(it) => { setEditing(it); setSvcOpen(true); }} />

			<ServiceDialog open={svcOpen} onOpenChange={setSvcOpen} item={editing} />
			<SettingsDialog open={settingsOpen} onOpenChange={setSettingsOpen} />
		</div>
	);
}
