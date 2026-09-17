import { ChevronDown, GripVertical, Pencil, Trash2 } from "lucide-react";
import { cn } from "@/lib/utils";
import { useItemDnd, useSectionDnd } from "@/board/dnd";
import { IconImg, ItemAnchor, StatusChip, hostOf } from "@/board/primitives";
import type {
	BoardViewProps,
	ItemViewProps,
	RendererViews,
	SectionViewProps,
} from "@/renderers/types";

// Bento — default renderer. Mixed-size grid, hairline section headers,
// status chips carry the only color. Ported from mockups/d-bento-mono.html.

function BoardView({ children }: BoardViewProps) {
	return <div className="flex flex-col gap-5">{children}</div>;
}

function SectionView({ section, onToggle, onDelete, children }: SectionViewProps) {
	const dnd = useSectionDnd(section.id);
	return (
		<section
			ref={dnd.ref}
			style={dnd.style}
			className={cn(dnd.isDragging && "opacity-35")}
		>
			<div className="group/sec flex cursor-pointer select-none items-center gap-3 px-0.5 py-1">
				<button
					type="button"
					aria-label="Drag section"
					className="cursor-grab text-text-faint opacity-0 transition-opacity group-hover/sec:opacity-100 active:cursor-grabbing"
					{...dnd.attributes}
					{...dnd.listeners}
				>
					<GripVertical size={13} />
				</button>
				<button
					type="button"
					onClick={onToggle}
					className="flex min-w-0 flex-1 items-center gap-3 text-left"
				>
					<h2 className="text-[11px] font-semibold uppercase tracking-[0.14em] text-text-dim">
						{section.name}
					</h2>
					<span className="font-mono text-[11px] text-text-faint">
						{String(section.items.length).padStart(2, "0")}
					</span>
					<span className="h-px flex-1 bg-border" />
					<ChevronDown
						size={13}
						className={cn("text-text-faint transition-transform duration-200", section.collapsed && "-rotate-90")}
					/>
				</button>
				<button
					type="button"
					aria-label="Delete section"
					onClick={onDelete}
					className="text-text-faint opacity-0 transition-opacity hover:text-destructive group-hover/sec:opacity-100"
				>
					<Trash2 size={12} />
				</button>
			</div>
			{!section.collapsed && children}
		</section>
	);
}

function ItemView({ item, sectionId, status, onEdit }: ItemViewProps) {
	const dnd = useItemDnd(sectionId, item.id);
	const host = item.urls[0] ? hostOf(item.urls[0].url) : "";
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
						"group/tile relative col-span-1 flex min-h-[70px] cursor-grab items-center gap-3",
						"rounded-[10px] border border-border bg-surface px-[15px] py-[13px]",
						"transition-colors hover:border-border-strong hover:bg-surface-hover active:scale-[0.98]",
						"min-[620px]:col-span-2",
						dnd.isDragging && "opacity-35",
					)}
				>
					<IconImg item={item} size={40} />
					<div className="min-w-0">
						<div className="truncate text-[13.5px] font-medium tracking-tight">{item.name}</div>
						{host ? <div className="truncate font-mono text-[10.5px] text-text-faint">{host}</div> : null}
					</div>
					{item.urls.length > 0 && <StatusChip status={status} />}
					<button
						type="button"
						aria-label={`Edit ${item.name}`}
						onClick={(e) => {
							e.stopPropagation();
							onEdit();
						}}
						className="absolute right-2 top-2 rounded p-1 text-text-faint opacity-0 transition-opacity hover:text-text group-hover/tile:opacity-100"
					>
						<Pencil size={11} />
					</button>
				</div>
			)}
		</ItemAnchor>
	);
}

// Widget tiles land in Phase 2; render a quiet placeholder tile meanwhile.
function WidgetView({ item, sectionId }: ItemViewProps) {
	const dnd = useItemDnd(sectionId, item.id);
	return (
		<div
			ref={dnd.ref}
			style={dnd.style}
			{...dnd.attributes}
			{...dnd.listeners}
			className={cn(
				"col-span-1 flex min-h-[70px] items-center rounded-[10px] border border-dashed border-border-strong px-[15px] py-[13px]",
				"min-[620px]:col-span-2",
				dnd.isDragging && "opacity-35",
			)}
		>
			<span className="text-[11px] font-semibold uppercase tracking-[0.12em] text-text-faint">
				{item.name}
			</span>
		</div>
	);
}

export const bento: RendererViews = {
	BoardView,
	SectionView,
	ItemView,
	WidgetView,
	itemsGridClass: "grid grid-cols-2 min-[620px]:grid-cols-4 min-[900px]:grid-cols-6 gap-2.5",
};
