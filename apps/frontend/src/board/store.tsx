import {
	createContext,
	useCallback,
	useContext,
	useEffect,
	useMemo,
	useState,
	type ReactNode,
} from "react";
import { api } from "@/api";
import type {
	Board, Item, ItemInput, ItemPatch, Prefs, RendererName, Section, StatusMap, Theme,
} from "@/types";
import { parsePrefs } from "@/types";
import { setLocale } from "@/i18n";

// Board store — single renderer-agnostic source of truth for the board tree,
// reachability statuses, and persisted settings. Renderers consume via useBoard.

interface BoardState {
	sections: Section[];
	boards: Board[];
	boardId: string;
	boardSlug: string;
	statuses: StatusMap;
	theme: Theme;
	renderer: RendererName;
	accent: string;
	wallpaper: string;
	customCss: string;
	locale: string;
	loaded: boolean;
	refresh(): Promise<void>;
	// slug is the URL form; "" means the default (first) board.
	setActiveSlug(slug: string): void;
	addBoard(name: string): Promise<Board | null>;
	renameBoard(id: string, name: string): Promise<void>;
	deleteBoard(id: string): Promise<void>;
	addSection(name: string): Promise<Section | null>;
	patchSection(id: string, patch: { name?: string; collapsed?: boolean }): Promise<void>;
	deleteSection(id: string): Promise<void>;
	addItem(input: ItemInput): Promise<Item | null>;
	patchItem(id: string, patch: ItemPatch): Promise<void>;
	deleteItem(id: string): Promise<void>;
	uploadIcon(id: string, file: File): Promise<void>;
	moveItemLocal(itemId: string, toSectionId: string, index: number): void;
	moveSectionLocal(id: string, index: number): void;
	persistItemMove(itemId: string, sectionId: string, beforeId?: string, afterId?: string): Promise<void>;
	persistSectionMove(id: string, beforeId?: string, afterId?: string): Promise<void>;
	setSetting(key: keyof Prefs, value: string): Promise<void>;
}

const Ctx = createContext<BoardState | null>(null);

export function BoardProvider({ children }: { children: ReactNode }) {
	const [sections, setSections] = useState<Section[]>([]);
	const [boards, setBoards] = useState<Board[]>([]);
	const [activeSlug, setActiveSlug] = useState("");
	const [statuses, setStatuses] = useState<StatusMap>({});
	const [prefs, setPrefs] = useState<Prefs>({ theme: "dark", renderer: "bento", accent: "", wallpaper: "", customCss: "", locale: "" });
	const [loaded, setLoaded] = useState(false);

	// Resolve slug → id; unknown/empty slugs fall back to the first board.
	const board = boards.find((b) => b.slug === activeSlug) ?? boards[0];
	const boardId = board?.id ?? "";
	const boardSlug = board?.slug ?? "";

	const refresh = useCallback(async () => {
		if (!boardId) return;
		const { data } = await api.GET("/api/sections", { params: { query: { board: boardId } } });
		if (data) setSections(data);
	}, [boardId]);

	const refreshStatus = useCallback(async () => {
		const { data } = await api.GET("/api/status");
		if (data) setStatuses(data);
	}, []);

	useEffect(() => {
		void (async () => {
			const [{ data: bs }, { data: cfg }] = await Promise.all([
				api.GET("/api/boards"),
				api.GET("/api/settings"),
			]);
			if (bs) setBoards(bs);
			if (cfg) setPrefs(parsePrefs(cfg));
			setLoaded(true);
			void refreshStatus();
		})();
		const t = setInterval(() => void refreshStatus(), 60_000);
		return () => clearInterval(t);
	}, [refreshStatus]);

	// Sections reload whenever the active board changes.
	useEffect(() => {
		void refresh();
	}, [refresh]);

	const addBoard = useCallback(async (name: string) => {
		const { data } = await api.POST("/api/boards", { body: { name } });
		if (data) setBoards((bs) => [...bs, data]);
		return data ?? null;
	}, []);

	const renameBoard = useCallback(async (id: string, name: string) => {
		const { data } = await api.PATCH("/api/boards/{id}", { params: { path: { id } }, body: { name } });
		if (data) setBoards((bs) => bs.map((b) => (b.id === id ? data : b)));
	}, []);

	const deleteBoard = useCallback(async (id: string) => {
		const { error } = await api.DELETE("/api/boards/{id}", { params: { path: { id } } });
		if (!error) {
			setBoards((bs) => bs.filter((b) => b.id !== id));
			setActiveSlug("");
		}
	}, []);

	const { theme, renderer, accent, wallpaper, customCss, locale } = prefs;

	// t() reads a module-level dict; this effect swaps it when the pref lands.
	useEffect(() => setLocale(locale), [locale]);

	// Apply theme: <html data-theme> drives all CSS vars. localStorage mirrors
	// the server setting so index.html can pre-paint without a flash.
	useEffect(() => {
		if (!loaded) return;
		document.documentElement.dataset.theme = theme;
		localStorage.setItem("dash.theme", theme);
	}, [theme, loaded]);

	// Appearance overrides: accent is a CSS var the whole theme references,
	// wallpaper hangs off <body>, customCss lands in a managed <style> tag.
	useEffect(() => {
		if (!loaded) return;
		const root = document.documentElement;
		if (accent) root.style.setProperty("--accent", accent);
		else root.style.removeProperty("--accent");
		// Quotes/backslashes stripped so the URL can't break out of url("…").
		const safeWallpaper = wallpaper.replace(/["\\()]/g, "");
		document.body.style.backgroundImage = safeWallpaper ? `url("${safeWallpaper}")` : "";
		document.body.style.backgroundSize = wallpaper ? "cover" : "";
		document.body.style.backgroundAttachment = wallpaper ? "fixed" : "";
		let tag = document.getElementById("dash-custom-css");
		if (customCss) {
			if (!tag) {
				tag = document.createElement("style");
				tag.id = "dash-custom-css";
				document.head.appendChild(tag);
			}
			tag.textContent = customCss;
		} else {
			tag?.remove();
		}
	}, [accent, wallpaper, customCss, loaded]);

	const addSection = useCallback(async (name: string) => {
		const { data } = await api.POST("/api/sections", { body: { name, boardId } });
		if (data) setSections((s) => [...s, data]);
		return data ?? null;
	}, [boardId]);

	const patchSection = useCallback(async (id: string, patch: { name?: string; collapsed?: boolean }) => {
		setSections((ss) => ss.map((s) => (s.id === id ? { ...s, ...patch } : s)));
		const { error } = await api.PATCH("/api/sections/{id}", { params: { path: { id } }, body: patch });
		if (error) await refresh();
	}, [refresh]);

	const deleteSection = useCallback(async (id: string) => {
		setSections((ss) => ss.filter((s) => s.id !== id));
		const { error } = await api.DELETE("/api/sections/{id}", { params: { path: { id } } });
		if (error) await refresh();
	}, [refresh]);

	const addItem = useCallback(async (input: ItemInput) => {
		const { data } = await api.POST("/api/items", { body: input });
		if (data) {
			setSections((ss) => ss.map((s) => (s.id === data.sectionId ? { ...s, items: [...s.items, data] } : s)));
			void refreshStatus();
		}
		return data ?? null;
	}, [refreshStatus]);

	const patchItem = useCallback(async (id: string, patch: ItemPatch) => {
		const { data } = await api.PATCH("/api/items/{id}", { params: { path: { id } }, body: patch });
		if (data) {
			setSections((ss) => ss.map((s) => ({
				...s,
				items: s.items.map((i) => (i.id === id ? data : i)),
			})));
			void refreshStatus();
		}
	}, [refreshStatus]);

	const deleteItem = useCallback(async (id: string) => {
		setSections((ss) => ss.map((s) => ({ ...s, items: s.items.filter((i) => i.id !== id) })));
		const { error } = await api.DELETE("/api/items/{id}", { params: { path: { id } } });
		if (error) await refresh();
	}, [refresh]);

	const uploadIcon = useCallback(async (id: string, file: File) => {
		const form = new FormData();
		form.append("file", file);
		const res = await fetch(`/api/items/${id}/icon`, { method: "POST", body: form });
		if (res.ok) {
			// SAFETY: /api/items/:id/icon returns Item per openapi.yaml; fetch isn't typed by the client.
			const it = (await res.json()) as Item;
			setSections((ss) => ss.map((s) => ({
				...s,
				items: s.items.map((i) => (i.id === id ? it : i)),
			})));
		}
	}, []);

	// Local moves run during drag (optimistic); persist* fire the API on drop.
	const moveItemLocal = useCallback((itemId: string, toSectionId: string, index: number) => {
		setSections((ss) => {
			const next = ss.map((s) => ({ ...s, items: [...s.items] }));
			let moved: Item | undefined;
			for (const s of next) {
				const i = s.items.findIndex((it) => it.id === itemId);
				if (i >= 0) {
					[moved] = s.items.splice(i, 1);
					break;
				}
			}
			if (!moved) return ss;
			moved = { ...moved, sectionId: toSectionId };
			const target = next.find((s) => s.id === toSectionId);
			if (!target) return ss;
			target.items.splice(Math.min(index, target.items.length), 0, moved);
			return next;
		});
	}, []);

	const moveSectionLocal = useCallback((id: string, index: number) => {
		setSections((ss) => {
			const next = [...ss];
			const from = next.findIndex((s) => s.id === id);
			if (from < 0) return ss;
			const [moved] = next.splice(from, 1);
			if (!moved) return ss;
			next.splice(Math.min(index, next.length), 0, moved);
			return next;
		});
	}, []);

	const persistItemMove = useCallback(async (itemId: string, sectionId: string, beforeId?: string, afterId?: string) => {
		const { error } = await api.POST("/api/items/reorder", {
			body: {
				id: itemId, sectionId,
				...(beforeId ? { beforeId } : {}),
				...(afterId ? { afterId } : {}),
			},
		});
		if (error) await refresh();
	}, [refresh]);

	const persistSectionMove = useCallback(async (id: string, beforeId?: string, afterId?: string) => {
		const { error } = await api.POST("/api/sections/reorder", {
			body: {
				id,
				...(beforeId ? { beforeId } : {}),
				...(afterId ? { afterId } : {}),
			},
		});
		if (error) await refresh();
	}, [refresh]);

	const setSetting = useCallback(async (key: keyof Prefs, value: string) => {
		setPrefs((p) => ({ ...p, [key]: value }));
		// Prefs uses camelCase; wire keys keep the snake_case convention.
		const wireKey = key === "customCss" ? "custom_css" : key;
		const { data, error } = await api.PUT("/api/settings", {
			body: { [wireKey]: value },
		});
		if (error) await refresh();
		else if (data) setPrefs(parsePrefs(data));
	}, [refresh]);

	const value = useMemo<BoardState>(() => ({
		sections, boards, boardId, boardSlug, statuses, theme, renderer, accent, wallpaper, customCss, locale, loaded,
		refresh, setActiveSlug, addBoard, renameBoard, deleteBoard,
		addSection, patchSection, deleteSection,
		addItem, patchItem, deleteItem, uploadIcon,
		moveItemLocal, moveSectionLocal, persistItemMove, persistSectionMove,
		setSetting,
	}), [
		sections, boards, boardId, boardSlug, statuses, theme, renderer, accent, wallpaper, customCss, locale, loaded,
		refresh, addBoard, renameBoard, deleteBoard,
		addSection, patchSection, deleteSection,
		addItem, patchItem, deleteItem, uploadIcon,
		moveItemLocal, moveSectionLocal, persistItemMove, persistSectionMove,
		setSetting,
	]);

	return <Ctx.Provider value={value}>{children}</Ctx.Provider>;
}

export function useBoard(): BoardState {
	const v = useContext(Ctx);
	if (!v) throw new Error("useBoard outside BoardProvider");
	return v;
}
