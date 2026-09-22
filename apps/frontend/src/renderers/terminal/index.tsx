import { useEffect, useState } from "react";
import { ChevronDown, Pencil, Trash2 } from "lucide-react";
import { cn } from "@/lib/utils";
import { t } from "@/i18n";
import { useItemDnd, useSectionDnd } from "@/board/dnd";
import { useBoard } from "@/board/store";
import { IconImg, ItemAnchor, hostOf } from "@/board/primitives";
import { WidgetContent } from "@/widgets";
import type {
	BoardViewProps,
	ItemViewProps,
	RendererViews,
	SectionViewProps,
} from "@/renderers/types";

// Console — terminal renderer. Mono everything, $ banner, bordered stat
// panels, ## section heads, › prompt rows. Ported from mockups/f-console.html.

function BoardView({ children }: BoardViewProps) {
	const board = useBoard();
	const [now, setNow] = useState(() => new Date());

	useEffect(() => {
		const t = setInterval(() => setNow(new Date()), 1000);
		return () => clearInterval(t);
	}, []);

	const items = board.sections.flatMap((s) => s.items);
	const services = items.filter((i) => i.kind !== "widget");
	const widgets = items.filter((i) => i.kind === "widget");
	const endpoints = services.reduce((n, i) => n + i.urls.length, 0);
	const up = services.filter((i) => board.statuses[i.id]?.status === "up").length;
	const down = services.filter((i) => board.statuses[i.id]?.status === "down").length;
	const probed = up + down;

	const time = new Intl.DateTimeFormat(undefined, {
		hour: "2-digit", minute: "2-digit", second: "2-digit", hour12: false,
	}).format(now);
	const date = new Intl.DateTimeFormat(undefined, {
		weekday: "short", month: "short", day: "numeric",
	}).format(now);

	const panels: { head: string; v: string; s: string; live?: boolean; wide?: boolean }[] = [
		{ head: "clock --local", v: time, s: date, live: true, wide: true },
		{ head: "services --up", v: String(up), s: `of ${probed} probed` },
		{ head: "services --down", v: String(down), s: down === 0 ? t("renderer.allGreen") : t("renderer.needsAttention") },
		{ head: "widgets", v: String(widgets.length), s: "registered" },
		{ head: "endpoints", v: String(endpoints), s: "urls indexed" },
		{ head: "uptime --probes", v: probed > 0 ? `${Math.round((up / probed) * 100)}%` : "—", s: `${up}/${probed} responding` },
		{ head: "sections", v: String(board.sections.length), s: "groups" },
	];

	return (
		<div className="font-terminal">
			<div className="mb-6 whitespace-pre-wrap rounded-md border border-border px-4 py-3.5 text-[11px] leading-[1.7] text-text-faint">
				{`$ dash status --all\n`}
				<span className="font-medium text-text-dim">{services.length} services</span>
				{` indexed · `}
				<span className="text-up">{up} up</span>
				{down > 0 ? <span>{` · `}<span className="text-down">{t("renderer.downCount", { n: down })}</span></span> : ` · ${t("renderer.allGreen")}`}
				{` · `}
				<span className="font-medium text-text-dim">{widgets.length} widgets</span>
				{`\ndrag rows to reorder · click rows with multiple endpoints to choose`}
			</div>

			<div className="mb-6 grid grid-cols-2 gap-2.5 min-[760px]:grid-cols-4">
				{panels.map((p) => (
					<div
						key={p.head}
						className={cn("overflow-hidden rounded-md border border-border bg-surface", p.wide && "col-span-2")}
					>
						<div className="flex items-center gap-1.5 border-b border-border px-3 py-2 text-[10px] uppercase tracking-[0.1em] text-text-faint">
							{p.head}
							{p.live ? <span className="ml-auto text-[9px] text-up">● {t("renderer.live")}</span> : null}
						</div>
						<div className="px-3 py-3">
							<div className={cn("font-medium leading-none tracking-tight tabular-nums", p.wide ? "text-[26px]" : "text-[22px]")}>{p.v}</div>
							<div className="mt-1 truncate text-[10.5px] text-text-faint">{p.s}</div>
						</div>
					</div>
				))}
			</div>

			<div className="flex flex-col gap-6">{children}</div>
		</div>
	);
}

function SectionView({ section, onToggle, onDelete, children }: SectionViewProps) {
	const dnd = useSectionDnd(section.id);
	return (
		<section ref={dnd.ref} style={dnd.style} className={cn(dnd.isDragging && "opacity-35")}>
			<div className="group/sec mb-2 flex cursor-pointer select-none items-center gap-2.5 px-0.5 py-1 text-[12px]">
				<button
					type="button"
					onClick={onToggle}
					className="flex min-w-0 flex-1 items-center gap-2.5 text-left"
				>
					<span className="text-text-faint">##</span>
					<h2 className="truncate font-bold lowercase tracking-[0.02em]">{section.name}</h2>
					<span className="text-[11px] text-text-faint">[{section.items.length}]</span>
					<span className="flex-1 overflow-hidden whitespace-nowrap text-text-faint/60" aria-hidden>
						{"─".repeat(120)}
					</span>
					<ChevronDown
						size={12}
						className={cn("text-text-faint transition-transform duration-150", section.collapsed && "-rotate-90")}
					/>
				</button>
				<button
					type="button"
					aria-label={t("renderer.dragSection")}
					className="cursor-grab text-text-faint opacity-0 transition-opacity group-hover/sec:opacity-100 active:cursor-grabbing"
					{...dnd.attributes}
					{...dnd.listeners}
				>
					≡
				</button>
				<button
					type="button"
					aria-label={t("renderer.deleteSection")}
					onClick={onDelete}
					className="text-text-faint opacity-0 transition-opacity hover:text-destructive group-hover/sec:opacity-100"
				>
					<Trash2 size={11} />
				</button>
			</div>
			{!section.collapsed && (
				<div className="overflow-hidden rounded-md border border-border bg-surface">{children}</div>
			)}
		</section>
	);
}

function ItemView({ item, sectionId, status, onEdit }: ItemViewProps) {
	const dnd = useItemDnd(sectionId, item.id);
	const host = item.urls[0] ? hostOf(item.urls[0].url) : "";
	const state = status?.status;
	return (
		<ItemAnchor item={item} onEdit={onEdit}>
			{(open) => (
				<div
					ref={dnd.ref}
					style={dnd.style}
					{...dnd.attributes}
					{...dnd.listeners}
					onClick={open}
					className={cn(
						"group/row relative grid cursor-grab grid-cols-[16px_24px_minmax(0,1fr)_auto] items-center gap-3",
						"border-b border-border px-3 py-2 text-[12.5px] transition-colors last:border-b-0 hover:bg-surface-hover",
						"min-[760px]:grid-cols-[20px_26px_minmax(0,1.2fr)_minmax(0,1fr)_auto]",
						dnd.isDragging && "opacity-35",
					)}
				>
					<span className="text-text-faint">›</span>
					<IconImg item={item} size={22} />
					<span className="min-w-0">
						<span className="block truncate font-medium lowercase">{item.name}</span>
						{host ? <span className="block truncate text-[10.5px] text-text-faint min-[760px]:hidden">{host}</span> : null}
					</span>
					<span className="hidden min-w-0 truncate text-[11px] text-text-faint min-[760px]:block">
						{host}{item.urls.length > 1 ? ` · ${item.urls.length} endpoints` : ""}
					</span>
					{item.urls.length > 0 ? (
						<span
							className={cn(
								"justify-self-end text-[10.5px] tracking-[0.04em] transition-opacity group-hover/row:opacity-0",
								state === "up" ? "text-up" : state === "down" ? "text-down" : "text-text-faint",
							)}
						>
							[ {state ?? "…"} ]
						</span>
					) : <span />}
					<button
						type="button"
						aria-label={`Edit ${item.name}`}
						onClick={(e) => {
							e.stopPropagation();
							onEdit();
						}}
						className="absolute right-2.5 top-1/2 -translate-y-1/2 rounded bg-popover p-1 text-text-faint opacity-0 transition-opacity hover:text-text group-hover/row:opacity-100"
					>
						<Pencil size={10} />
					</button>
				</div>
			)}
		</ItemAnchor>
	);
}

function WidgetView({ item, sectionId, onEdit }: ItemViewProps) {
	const dnd = useItemDnd(sectionId, item.id);
	return (
		<div
			ref={dnd.ref}
			style={dnd.style}
			{...dnd.attributes}
			{...dnd.listeners}
			className={cn(
				"group/row relative grid cursor-grab grid-cols-[16px_24px_minmax(0,1fr)] items-center gap-3",
				"border-b border-border px-3 py-2 text-[12.5px] transition-colors last:border-b-0 hover:bg-surface-hover",
				"min-[760px]:grid-cols-[20px_26px_minmax(0,0.8fr)_minmax(0,1.2fr)]",
				dnd.isDragging && "opacity-35",
			)}
		>
			<span className="text-text-faint">›</span>
			<IconImg item={item} size={22} />
			<span className="min-w-0 truncate font-medium lowercase">{item.name}</span>
			<div className="min-w-0 max-[759px]:col-start-3 max-[759px]:py-1">
				<WidgetContent item={item} />
			</div>
			<button
				type="button"
				aria-label={`Edit ${item.name}`}
				onClick={(e) => {
					e.stopPropagation();
					onEdit();
				}}
				className="absolute right-2.5 top-2 rounded bg-popover p-1 text-text-faint opacity-0 transition-opacity hover:text-text group-hover/row:opacity-100"
			>
				<Pencil size={10} />
			</button>
		</div>
	);
}

export const terminal: RendererViews = {
	BoardView,
	SectionView,
	ItemView,
	WidgetView,
	itemsGridClass: "flex flex-col",
};
