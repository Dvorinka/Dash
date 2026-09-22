import { en, type Key } from "@/i18n/en";
import { cs } from "@/i18n/cs";

export type { Key } from "@/i18n/en";

export const LOCALES = { en: "English", cs: "Čeština" } as const;
export type Locale = keyof typeof LOCALES;

// i18n seam: dictionaries are plain objects keyed by dotted names. `t` is a
// typed lookup that falls back to English, then to the key itself — a missing
// translation is loud in dev, never a crash. The store calls setLocale when
// the `locale` setting loads/changes; components re-render on prefs updates.
const dicts: Record<Locale, Partial<Record<Key, string>>> = { en, cs };
let active: Locale = "en";

export function setLocale(l: string) {
	active = l === "cs" ? "cs" : "en";
	document.documentElement.lang = active;
}

export function t(key: Key, params?: Record<string, string | number>): string {
	let s: string = dicts[active][key] ?? en[key] ?? key;
	if (params) {
		for (const [k, v] of Object.entries(params)) s = s.replaceAll(`{${k}}`, String(v));
	}
	return s;
}
