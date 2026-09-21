import { useEffect, useState } from "react";
import type { Item } from "@/types";
import { ClockWidget } from "@/widgets/clock";
import { PiholeWidget, AdguardWidget } from "@/widgets/blocker";
import { ImmichWidget } from "@/widgets/immich";
import { MonitorWidget } from "@/widgets/monitor";

// WidgetHost seam: renderers own the tile chrome; this maps config.type to
// the component that fills it. Adding a widget = one fetcher in Go + one
// entry here.

export type WidgetComponent = (p: { item: Item; data: WidgetData | undefined; error: string | undefined }) => React.ReactNode;

export interface WidgetData {
	[key: string]: string | number | boolean | undefined;
}

// Widget payloads are free-form per type; components read fields defensively.
function isData(v: unknown): v is WidgetData {
	return typeof v === "object" && v !== null;
}

interface WidgetRegistry {
	[type: string]: { local?: boolean; view: WidgetComponent };
}

const registry: WidgetRegistry = {
	clock: { local: true, view: ClockWidget },
	pihole: { view: PiholeWidget },
	adguard: { view: AdguardWidget },
	immich: { view: ImmichWidget },
	monitor: { view: MonitorWidget },
};

export function cfgStr(item: Item, key: string): string {
	const cfg = item.config;
	if (!cfg) return "";
	const v = cfg[key];
	return typeof v === "string" ? v : "";
}

export function cfgType(item: Item): string {
	return cfgStr(item, "type");
}

/** Polls /api/widgets/:id/data every 30s for server-backed widgets. */
export function useWidgetData(item: Item): { data: WidgetData | undefined; error: string | undefined } {
	const type = cfgType(item);
	const local = registry[type]?.local ?? false;
	const [state, setState] = useState<{ data: WidgetData | undefined; error: string | undefined }>({
		data: undefined, error: undefined,
	});

	useEffect(() => {
		if (local || !type) return;
		let dead = false;
		let timer: ReturnType<typeof setTimeout>;

		const tick = async () => {
			try {
				const res = await fetch(`/api/widgets/${item.id}/data`);
				const body: unknown = await res.json();
				if (dead) return;
				if (!res.ok) {
					let msg = "fetch failed";
					if (typeof body === "object" && body !== null && "error" in body) {
						msg = String(body.error);
					}
					setState((s) => ({ data: s.data, error: msg }));
				} else {
					let data: WidgetData = {};
					if (typeof body === "object" && body !== null && "data" in body && isData(body.data)) {
						data = body.data;
					}
					setState({ data, error: undefined });
				}
			} catch {
				if (!dead) setState((s) => ({ data: s.data, error: "offline" }));
			}
			if (!dead) timer = setTimeout(tick, 30_000);
		};
		void tick();
		return () => {
			dead = true;
			clearTimeout(timer);
		};
	}, [item.id, type, local]);

	return state;
}

/** WidgetContent renders inside a renderer's tile chrome. */
export function WidgetContent({ item }: { item: Item }) {
	const { data, error } = useWidgetData(item);
	const type = cfgType(item);
	const entry = registry[type];
	if (!entry) {
		return <span className="text-[11px] uppercase tracking-[0.1em] text-text-faint">{item.name || "widget"}</span>;
	}
	return <>{entry.view({ item, data, error })}</>;
}
