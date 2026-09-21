import { en, type Key } from "@/i18n/en";

export type { Key } from "@/i18n/en";

// i18n seam: dictionaries are plain objects keyed by dotted names. `t` is a
// typed lookup that falls back to the key itself, so a missing translation
// renders the key — loud in dev, never a crash. Locales beyond `en` plug in
// here later (settings key + lazy import); nothing else changes.
const dict: Record<Key, string> = en;

export function t(key: Key): string {
	return dict[key] ?? key;
}
