import { useEffect, useState } from "react";
import { ChevronDown, GripVertical, Pencil, Trash2 } from "lucide-react";
import { cn } from "@/lib/utils";
import { t } from "@/i18n";
import { useItemDnd, useSectionDnd } from "@/board/dnd";
import { useBoard } from "@/board/store";
import { IconImg, ItemAnchor, formatLatency, hostOf, probeCounts } from "@/board/primitives";
import { WidgetContent } from "@/widgets";
import type {
	BoardViewProps,
	ItemViewProps,
	RendererViews,
	SectionViewProps,
} from "@/renderers/types";

// Index — editorial/list renderer. Serif masthead + section heads, numbered
// full-width rows, stat strip. Ported from mockups/e-editorial.html.

function BoardView({ children }: BoardViewProps) {
	const board = useBoard();
	const [now, setNow] = useState(() => new Date());

	useEffect(() => {
		const t = setInterval(() => setNow(new Date()), 1000);
		return () => clearInterval(t);
	}, []);

	const services = board.sections.flatMap((s) => s.items).filter((i) => i.kind !== "widget");
	const widgets = board.sections.flatMap((s) => s.items).filter((i) => i.kind === "widget");
	const { up, down, pending } = probeCounts(services, board.statuses);
	const settled = (n: number) => (pending > 0 && n === 0 ? "-" : String(n));

	const time = new Intl.DateTimeFormat(undefined, {
		hour: "2-digit", minute: "2-digit", second: "2-digit", hour12: false,
	}).format(now);
	const date = new Intl.DateTimeFormat(undefined, {
		weekday: "long", day: "numeric", month: "long", year: "numeric",
	}).format(now);

	const cells: [string, string, string][] = [
		[t("renderer.localTime"), time, date.split(",")[0] ?? ""],
		[t("renderer.services"), String(services.length), t(board.sections.length === 1 ? "renderer.section" : "renderer.sections", { n: board.sections.length })],
		[t("renderer.up"), settled(up), t("renderer.responding")],
		[t("renderer.down"), settled(down), down > 0 ? t("renderer.offline") : pending > 0 ? t("common.loading") : t("renderer.allGreen")],
		[t("renderer.widgets"), String(widgets.length), t("renderer.registered")],
	];

	return (
		<div>
			<header className="mb-7 flex items-end justify-between gap-6 border-b border-border-strong pb-5">
				<h1 className="font-serif text-[52px] leading-none tracking-tight max-[620px]:text-[38px]">
					Dash.
				</h1>
				<span className="pb-1 text-right font-mono text-[11px] uppercase tracking-[0.08em] text-text-faint">
					{date}
				</span>
			</header>

			<div className="mb-10 grid grid-cols-2 gap-px overflow-hidden rounded-[10px] border border-border bg-border min-[760px]:grid-cols-5">
				{cells.map(([k, v, s], i) => (
					<div key={k} className={cn("bg-surface px-4 py-3.5", i === cells.length - 1 && "col-span-2 min-[760px]:col-span-1")}>
						<div className="font-mono text-[10px] uppercase tracking-[0.1em] text-text-faint">{k}</div>
						<div className="mt-1 font-mono text-[24px] font-medium leading-none tracking-tight tabular-nums">{v}</div>
						<div className="mt-1.5 truncate text-[10.5px] text-text-faint">{s}</div>
					</div>
				))}
			</div>

			<div className="flex flex-col gap-9">{children}</div>
		</div>
	);
}

function SectionView({ section, index, onToggle, onDelete, children }: SectionViewProps) {
	const dnd = useSectionDnd(section.id);
	return (
		<section ref={dnd.ref} style={dnd.style} className={cn(dnd.isDragging && "opacity-35")}>
			<div className="group/sec mb-0.5 flex cursor-pointer select-none items-baseline gap-3.5 border-b border-border-strong pb-2.5">
				<button
					type="button"
					aria-label={t("renderer.dragSection")}
					className="show-on-hover cursor-grab self-center text-text-faint transition-opacity active:cursor-grabbing"
					{...dnd.attributes}
					{...dnd.listeners}
				>
					<GripVertical size={13} />
				</button>
				<button
					type="button"
					onClick={onToggle}
					className="flex min-w-0 flex-1 items-baseline gap-3.5 text-left"
				>
					<span className="font-serif text-[30px] leading-none text-text-faint tabular-nums">
						{String(index + 1).padStart(2, "0")}
					</span>
					<h2 className="truncate font-serif text-[30px] leading-none tracking-tight">
						{section.name}
					</h2>
					<span className="font-mono text-[11px] text-text-faint">
						{section.items.length} {t(section.items.length === 1 ? "renderer.entry" : "renderer.entries")}
					</span>
					<ChevronDown
						size={14}
						className={cn("ml-auto self-center text-text-faint transition-transform duration-200", section.collapsed && "-rotate-90")}
					/>
				</button>
				<button
					type="button"
					aria-label={t("renderer.deleteSection")}
					onClick={onDelete}
					className="show-on-hover self-center text-text-faint transition-opacity hover:text-destructive"
				>
					<Trash2 size={12} />
				</button>
			</div>
			{!section.collapsed && children}
		</section>
	);
}

function ItemView({ item, index, sectionId, status, onEdit }: ItemViewProps) {
	const dnd = useItemDnd(sectionId, item.id);
	const host = item.urls[0] ? hostOf(item.urls[0].url) : "";
	const state = status?.status;
	const latency = state ? formatLatency(status?.latencyMs) : "";
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
						"group/row touch-pad-action relative grid cursor-grab grid-cols-[26px_30px_minmax(0,1fr)_64px] items-center gap-3.5",
						"border-b border-border px-1.5 py-[11px] transition-colors hover:bg-surface-hover",
						"min-[760px]:grid-cols-[30px_34px_minmax(0,1.4fr)_minmax(0,1fr)_auto_108px_22px]",
						dnd.isDragging && "opacity-35",
					)}
				>
					<span className="font-mono text-[11px] text-text-faint tabular-nums">
						{String(index + 1).padStart(2, "0")}
					</span>
					<IconImg item={item} size={30} />
					<span className="min-w-0">
						<span className="block truncate text-[14px] font-medium tracking-tight">{item.name}</span>
						{host ? <span className="block truncate font-mono text-[10.5px] text-text-faint min-[760px]:hidden">{host}</span> : null}
					</span>
					<span className="hidden min-w-0 truncate font-mono text-[11px] text-text-faint min-[760px]:block">{host}</span>
					<span className="hidden justify-end gap-1.5 min-[760px]:flex">
						{item.urls.slice(0, 2).map((u) => u.label ? (
							<span key={u.id} className="rounded border border-border px-[5px] py-0.5 font-mono text-[9px] uppercase tracking-[0.08em] text-text-faint">
								{u.label}
							</span>
						) : null)}
					</span>
					{item.urls.length > 0 ? (
						<span className="hide-on-hover flex items-center gap-1.5 justify-self-end font-mono text-[10.5px] tracking-[0.05em] text-text-dim transition-opacity">
							<span className={cn("size-[5px] rounded-full", state === "up" ? "bg-up" : state === "down" ? "bg-down" : "bg-text-faint/40")} />
							{state ?? "…"}
							{latency ? <span className="hidden text-text-faint min-[760px]:inline">{latency}</span> : null}
						</span>
					) : <span />}
					<span className="show-on-hover hidden justify-self-end text-text-faint transition-all min-[760px]:block">→</span>
					<button
						type="button"
						aria-label={`Edit ${item.name}`}
						onClick={(e) => {
							e.stopPropagation();
							onEdit();
						}}
						className="show-on-hover absolute right-2.5 top-1/2 -translate-y-1/2 rounded bg-popover p-1 text-text-faint transition-opacity hover:text-text"
					>
						<Pencil size={11} />
					</button>
				</div>
			)}
		</ItemAnchor>
	);
}

function WidgetView({ item, index, sectionId, onEdit }: ItemViewProps) {
	const dnd = useItemDnd(sectionId, item.id);
	return (
		<div
			ref={dnd.ref}
			style={dnd.style}
			{...dnd.attributes}
			{...dnd.listeners}
			className={cn(
				"group/row touch-pad-action relative grid cursor-grab grid-cols-[26px_30px_minmax(0,1fr)] items-center gap-3.5",
				"border-b border-border px-1.5 py-[11px] transition-colors hover:bg-surface-hover",
				"min-[760px]:grid-cols-[30px_34px_minmax(0,0.8fr)_minmax(0,1.2fr)]",
				dnd.isDragging && "opacity-35",
			)}
		>
			<span className="font-mono text-[11px] text-text-faint tabular-nums">
				{String(index + 1).padStart(2, "0")}
			</span>
			<IconImg item={item} size={30} />
			<span className="min-w-0 truncate text-[14px] font-medium tracking-tight">{item.name}</span>
			<div className="min-w-0 max-[759px]:col-start-3">
				<WidgetContent item={item} />
			</div>
			<button
				type="button"
				aria-label={`Edit ${item.name}`}
				onClick={(e) => {
					e.stopPropagation();
					onEdit();
				}}
				className="show-on-hover absolute right-2.5 top-2 rounded bg-popover p-1 text-text-faint transition-opacity hover:text-text"
			>
				<Pencil size={11} />
			</button>
		</div>
	);
}

export const editorial: RendererViews = {
	BoardView,
	SectionView,
	ItemView,
	WidgetView,
	itemsGridClass: "flex flex-col",
};
