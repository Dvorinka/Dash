import { useEffect, useState } from "react";
import { Link, Route, useLocation } from "wouter";
import { Activity, AlertTriangle, Globe, LayoutGrid, Megaphone, Moon, Plus, Search, Server, Settings, Sun } from "lucide-react";
import { BoardProvider, useBoard } from "@/board/store";
import { BoardSwitcher } from "@/components/BoardSwitcher";
import { Board } from "@/board/Board";
import { ServiceDialog } from "@/components/ServiceDialog";
import { WidgetDialog } from "@/components/WidgetDialog";
import { SettingsDialog } from "@/components/SettingsDialog";
import { CommandPalette } from "@/components/CommandPalette";
import { MonitorsPage } from "@/pages/MonitorsPage";
import { MonitorDetailPage } from "@/pages/MonitorDetailPage";
import { DomainsPage } from "@/pages/DomainsPage";
import { DomainDetailPage } from "@/pages/DomainDetailPage";
import { SystemsPage } from "@/pages/SystemsPage";
import { SystemDetailPage } from "@/pages/SystemDetailPage";
import { IncidentsPage } from "@/pages/IncidentsPage";
import { StatusPagesPage } from "@/pages/StatusPagesPage";
import { StatusPublicPage } from "@/pages/StatusPublicPage";
import { LoginPage } from "@/pages/LoginPage";
import { api } from "@/api";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { t, type Key } from "@/i18n";
import type { Item } from "@/types";

export default function App() {
	return (
		<BoardProvider>
			<Shell />
		</BoardProvider>
	);
}

const NAV: { href: string; labelKey: Key; icon: typeof LayoutGrid }[] = [
	{ href: "/", labelKey: "nav.board", icon: LayoutGrid },
	{ href: "/monitors", labelKey: "nav.monitors", icon: Activity },
	{ href: "/domains", labelKey: "nav.domains", icon: Globe },
	{ href: "/systems", labelKey: "nav.systems", icon: Server },
	{ href: "/incidents", labelKey: "nav.incidents", icon: AlertTriangle },
	{ href: "/status", labelKey: "nav.status", icon: Megaphone },
];

function Shell() {
	const board = useBoard();
	const [loc] = useLocation();
	// null = still checking; gate blocks everything except public status pages.
	const [auth, setAuth] = useState<{ enabled?: boolean; setup?: boolean; authed?: boolean } | null>(null);
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

	// Auth gate: check the session once, and again whenever the client sees a
	// 401 (middleware in api.ts dispatches dash:unauthorized).
	useEffect(() => {
		const load = () => api.GET("/api/auth/session").then(({ data }) => setAuth(data ?? {}));
		load();
		window.addEventListener("dash:unauthorized", load);
		return () => window.removeEventListener("dash:unauthorized", load);
	}, []);

	// URL is the board source of truth: / -> default, /b/<slug> -> that board.
	const setActiveSlug = board.setActiveSlug;
	useEffect(() => {
		setActiveSlug(loc.startsWith("/b/") ? decodeURIComponent(loc.slice(3)) : "");
	}, [loc, setActiveSlug]);

	function openEditor(it: Item) {
		setEditing(it);
		if (it.kind === "widget") setWdgOpen(true);
		else setSvcOpen(true);
	}

	// Public status pages stay reachable when auth is on — the backend serves
	// their API unauthenticated, so the SPA must let the route through too.
	const publicPage = loc.startsWith("/status/");
	if (auth?.enabled && !auth.authed && !publicPage) {
		return <LoginPage setup={!!auth.setup} onDone={() => location.reload()} />;
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
				<BoardSwitcher />
				<nav className="ml-4 flex items-center gap-1" aria-label="Primary">
					{NAV.map((n) => {
						const active = n.href === "/" ? loc === "/" || loc.startsWith("/b/") : loc.startsWith(n.href);
						return (
							<Link
								key={n.href}
								href={n.href}
								className={cn(
									"flex items-center gap-1.5 rounded-[7px] px-2.5 py-1.5 text-[12.5px] transition-colors",
									active ? "bg-surface-hover text-text" : "text-text-faint hover:text-text",
								)}
							>
								<n.icon size={13} strokeWidth={1.8} />
								{t(n.labelKey)}
							</Link>
						);
					})}
				</nav>
				<div className="ml-auto flex items-center gap-2">
					<Button
						variant="outline" aria-label={t("header.search")}
						onClick={() => setPaletteOpen(true)}
						className="gap-1.5 text-text-faint"
					>
						<Search size={14} strokeWidth={1.8} />
						<kbd className="hidden font-mono text-[10px] min-[620px]:inline">⌘K</kbd>
					</Button>
					<Button
						variant="outline" size="icon" aria-label={t("header.settings")}
						onClick={() => setSettingsOpen(true)}
					>
						<Settings size={15} strokeWidth={1.8} />
					</Button>
					<Button
						variant="outline" size="icon" aria-label={t("header.theme")}
						onClick={() => void board.setSetting("theme", board.theme === "dark" ? "light" : "dark")}
					>
						{board.theme === "dark" ? <Moon size={15} strokeWidth={1.8} /> : <Sun size={15} strokeWidth={1.8} />}
					</Button>
					<Button
						variant="outline" aria-label={t("header.addWidget")}
						onClick={() => { setEditing(undefined); setWdgOpen(true); }}
					>
						<LayoutGrid size={13} strokeWidth={2} />
						{t("header.widget")}
					</Button>
					<Button onClick={() => { setEditing(undefined); setSvcOpen(true); }}>
						<Plus size={13} strokeWidth={2.2} />
						{t("header.addService")}
					</Button>
				</div>
			</header>

			<Route path="/">
				<Board onEditItem={openEditor} />
			</Route>
			<Route path="/b/:slug">
				<Board onEditItem={openEditor} />
			</Route>
			<Route path="/monitors">
				<MonitorsPage />
			</Route>
			<Route path="/monitors/:id">
				{(p) => <MonitorDetailPage id={p.id} />}
			</Route>
			<Route path="/domains">
				<DomainsPage />
			</Route>
			<Route path="/domains/:id">
				{(p) => <DomainDetailPage id={p.id} />}
			</Route>
			<Route path="/systems">
				<SystemsPage />
			</Route>
			<Route path="/systems/:id">
				{(p) => <SystemDetailPage id={p.id} />}
			</Route>
			<Route path="/incidents">
				<IncidentsPage />
			</Route>
			<Route path="/status">
				<StatusPagesPage />
			</Route>
			<Route path="/status/:slug">
				{(p) => <StatusPublicPage slug={p.slug} />}
			</Route>

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
