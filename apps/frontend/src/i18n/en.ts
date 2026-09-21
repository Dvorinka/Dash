// English source strings. Flat keys, dot-separated namespaces.
// Add a locale by copying this file — keys must match exactly.
export const en = {
	"nav.board": "Board",
	"nav.monitors": "Monitors",
	"nav.domains": "Domains",
	"nav.systems": "Systems",
	"nav.incidents": "Incidents",
	"nav.status": "Status",

	"header.search": "Search (⌘K)",
	"header.settings": "Settings",
	"header.theme": "Toggle theme",
	"header.addWidget": "Add widget",
	"header.addService": "Add service",
	"header.widget": "Widget",

	"common.add": "Add",
	"common.edit": "Edit",
	"common.delete": "Delete",
	"common.cancel": "Cancel",
	"common.save": "Save",
	"common.test": "Test",
	"common.loading": "Loading…",
} as const;

export type Key = keyof typeof en;
