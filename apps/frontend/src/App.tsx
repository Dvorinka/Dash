import { useEffect, useState } from "react";
import { Link, Route, useLocation } from "wouter";
import { Activity, AlertTriangle, Code2, Gauge, Globe, LayoutGrid, Megaphone, Moon, Plus, Search, Server, Settings, Sun } from "lucide-react";
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
import { OverviewPage } from "@/pages/OverviewPage";
import { DevPage } from "@/pages/DevPage";
import { DevProjectPage } from "@/pages/DevProjectPage";
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
	{ href: "/overview", labelKey: "nav.overview", icon: Gauge },
	{ href: "/monitors", labelKey: "nav.monitors", icon: Activity },
	{ href: "/domains", labelKey: "nav.domains", icon: Globe },
	{ href: "/systems", labelKey: "nav.systems", icon: Server },
	{ href: "/incidents", labelKey: "nav.incidents", icon: AlertTriangle },
	{ href: "/status", labelKey: "nav.status", icon: Megaphone },
	{ href: "/dev", labelKey: "nav.dev", icon: Code2 },
];

function Mark() {
	return (
		<svg width="18" height="18" viewBox="0 0 18 18" fill="none" aria-hidden>
			<rect x="1" y="1" width="7" height="7" rx="1.5" fill="currentColor" />
			<rect x="10" y="1" width="7" height="7" rx="1.5" fill="currentColor" opacity=".45" />
			<rect x="1" y="10" width="7" height="7" rx="1.5" fill="currentColor" opacity=".45" />
			<rect x="10" y="10" width="7" height="7" rx="1.5" fill="currentColor" />
		</svg>
	);
}

function ChromeTools({ onSearch, onTheme, onSettings, dark }: {
	onSearch(): void;
	onTheme(): void;
	onSettings(): void;
	dark: boolean;
}) {
	return (
		<>
			<Button variant="ghost" size="icon" aria-label={t("header.search")} className="size-10" onClick={onSearch}>
				<Search size={15} strokeWidth={1.8} />
			</Button>
			<Button variant="ghost" size="icon" aria-label={t("header.theme")} className="size-10" onClick={onTheme}>
				{dark ? <Moon size={15} strokeWidth={1.8} /> : <Sun size={15} strokeWidth={1.8} />}
			</Button>
			<Button variant="ghost" size="icon" aria-label={t("header.settings")} className="size-10" onClick={onSettings}>
				<Settings size={15} strokeWidth={1.8} />
			</Button>
		</>
	);
}

function navActive(href: string, loc: string) {
	if (href === "/") return loc === "/" || loc.startsWith("/b/");
	return loc === href || loc.startsWith(href + "/");
}

function NavItems({ loc, testid }: { loc: string; testid?: boolean }) {
	return NAV.map((n) => {
		const active = navActive(n.href, loc);
		return (
			<Link
				key={n.href}
				href={n.href}
				data-testid={testid ? `nav-${n.labelKey}` : undefined}
				aria-current={active ? "page" : undefined}
				className={cn(
					"flex min-h-10 shrink-0 items-center gap-2 rounded-[7px] px-2.5 text-[12.5px] transition-colors",
					active ? "bg-surface-hover text-text" : "text-text-dim hover:bg-surface-hover hover:text-text",
				)}
			>
				<n.icon size={14} strokeWidth={1.8} />
				{t(n.labelKey)}
			</Link>
		);
	});
}

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

	// Public status pages stay reachable when auth is on, and they render
	// without the admin shell — the URL is meant to be shared.
	const publicPage = loc.startsWith("/status/");
	if (publicPage) {
		return <StatusPublicPage slug={decodeURIComponent(loc.slice("/status/".length).split("/")[0] ?? "")} />;
	}
	if (auth === null) {
		return <div className="min-h-dvh bg-bg" />;
	}
	if (auth.enabled && !auth.authed) {
		return <LoginPage setup={!!auth.setup} onDone={() => location.reload()} />;
	}

	const onBoard = loc === "/" || loc.startsWith("/b/");
	const toolProps = {
		dark: board.theme === "dark",
		onSearch: () => setPaletteOpen(true),
		onTheme: () => void board.setSetting("theme", board.theme === "dark" ? "light" : "dark"),
		onSettings: () => setSettingsOpen(true),
	};

	return (
		<div className="min-h-dvh bg-bg font-sans md:grid md:grid-cols-[208px_minmax(0,1fr)]">
			<aside className="sticky top-0 hidden h-dvh flex-col border-r border-border px-3 py-4 md:flex">
				<Link href="/" className="mb-5 flex shrink-0 items-center gap-2.5 px-2 text-[15px] font-semibold tracking-tight">
					<Mark />
					Dash
				</Link>
				<nav className="flex min-h-0 flex-1 flex-col gap-0.5 overflow-y-auto" aria-label="Primary">
					<NavItems loc={loc} testid />
				</nav>
				<div className="mt-3 flex shrink-0 items-center gap-1 border-t border-border pt-3">
					<ChromeTools {...toolProps} />
				</div>
			</aside>

			<div className="min-w-0">
			<header className="sticky top-0 z-10 border-b border-border bg-bg/85 backdrop-blur-md md:static md:border-0 md:bg-transparent md:backdrop-blur-none">
				<div className="flex items-center gap-2 px-4 py-2.5 md:hidden">
					<Link href="/" className="flex items-center gap-2 text-[14px] font-semibold tracking-tight">
						<Mark />
						Dash
					</Link>
					<div className="ml-auto flex items-center gap-1"><ChromeTools {...toolProps} /></div>
				</div>
				<nav className="flex gap-1 overflow-x-auto px-3 pb-2 [mask-image:linear-gradient(to_right,black_calc(100%-28px),transparent)] md:hidden" aria-label="Primary">
					<NavItems loc={loc} />
				</nav>
				{onBoard && (
					<div className="mx-auto flex w-full max-w-[1140px] items-center gap-2 px-4 py-3 md:px-7 md:pt-5">
						<BoardSwitcher />
						<div className="ml-auto flex items-center gap-2">
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
					</div>
				)}
			</header>

			<Route path="/">
				<Board onEditItem={openEditor} />
			</Route>
			<Route path="/b/:slug">
				<Board onEditItem={openEditor} />
			</Route>
			<Route path="/overview">
				<OverviewPage />
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
			<Route path="/dev">
				<DevPage />
			</Route>
			<Route path="/dev/:id">
				{(p) => <DevProjectPage id={p.id} />}
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
		</div>
	);
}
