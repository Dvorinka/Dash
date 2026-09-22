import type { components, paths } from "@dash/api-client";

// Board entities — generated from openapi.yaml. Never hand-edit shapes here.
export type Section = components["schemas"]["Section"];
export type Board = components["schemas"]["Board"];
export type Item = components["schemas"]["Item"];
export type BoardUrl = components["schemas"]["Url"];
export type UrlInput = components["schemas"]["UrlInput"];
export type ItemInput = components["schemas"]["ItemInput"];
export type ItemPatch = components["schemas"]["ItemPatch"];
export type Status = components["schemas"]["Status"];
export type StatusMap = Record<string, Status>;
export type ExportPayload = components["schemas"]["Export"];

// Monitoring (merge program) — same generated-schema rule applies.
export type Monitor = components["schemas"]["Monitor"];
export type MonitorInput = components["schemas"]["MonitorInput"];
export type MonitorView = components["schemas"]["MonitorView"];
export type Heartbeat = components["schemas"]["Heartbeat"];
export type Domain = components["schemas"]["Domain"];
export type DomainInput = components["schemas"]["DomainInput"];
export type DomainView = components["schemas"]["DomainView"];
export type DomainCheck = components["schemas"]["DomainCheck"];
export type Subdomain = components["schemas"]["Subdomain"];
export type System = components["schemas"]["System"];
export type StatSample = components["schemas"]["StatSample"];
/** Stats rows are StatSample plus the server-side timestamp. */
export type SystemStat = StatSample & { ts?: string };
export type Incident = components["schemas"]["Incident"];
export type IncidentInput = components["schemas"]["IncidentInput"];
export type MaintenanceWindow = components["schemas"]["MaintenanceWindow"];
export type StatusPage = components["schemas"]["StatusPage"];
export type PublicStatus = components["schemas"]["PublicStatus"];

export type Theme = "dark" | "light";
export type RendererName = "bento" | "cards" | "index" | "console";

/** Wire shape of GET /api/settings — a free-form map we narrow at the boundary. */
export type SettingsWire = NonNullable<
	paths["/api/settings"]["get"]["responses"]["200"]["content"]["application/json"]
>;

export interface Prefs {
	theme: Theme;
	renderer: RendererName;
	/** CSS color override for --accent; empty = theme default. */
	accent: string;
	/** Background image URL for the board; empty = none. */
	wallpaper: string;
	/** Raw user CSS injected into a <style> tag; empty = none. */
	customCss: string;
	/** UI language — matches a key in i18n LOCALES; empty = en. */
	locale: string;
}

/** Narrow the settings map to typed prefs; unknown/absent keys fall back. */
export function parsePrefs(raw: SettingsWire): Prefs {
	const r = raw.renderer;
	const s = (v: unknown) => (typeof v === "string" ? v : "");
	return {
		theme: raw.theme === "light" ? "light" : "dark",
		renderer: r === "cards" || r === "index" || r === "console" ? r : "bento",
		accent: s(raw.accent),
		wallpaper: s(raw.wallpaper),
		customCss: s(raw.custom_css),
		locale: s(raw.locale),
	};
}
