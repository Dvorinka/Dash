import { ChevronDown, GripVertical, Pencil, Trash2 } from "lucide-react";
import { cn } from "@/lib/utils";
import { useItemDnd, useSectionDnd } from "@/board/dnd";
import { IconImg, ItemAnchor, StatusChip, hostOf } from "@/board/primitives";
import { WidgetContent } from "@/widgets";
import type {
	BoardViewProps,
	ItemViewProps,
	RendererViews,
	SectionViewProps,
} from "@/renderers/types";

// Cards — uniform compact grid. Second renderer; proves the seam while the
// code is small. Ported from mockups/a-vercel-mono.html.

function BoardView({ children }: BoardViewProps) {
	return <div className="flex flex-col gap-10">{children}</div>;
}

function SectionView({ section, onToggle, onDelete, children }: SectionViewProps) {
	const dnd = useSectionDnd(section.id);
	return (
		<section ref={dnd.ref} style={dnd.style} className={cn(dnd.isDragging && "opacity-40")}>
			<div className="group/sec mb-3.5 flex cursor-pointer select-none items-center gap-2.5 px-1 py-1.5 text-text-dim">
				<button
					type="button"
					onClick={onToggle}
					className="flex min-w-0 flex-1 items-center gap-2.5 text-left"
				>
					<h2 className="text-[11px] font-semibold uppercase tracking-[0.14em]">{section.name}</h2>
					<span className="text-[11px] tabular-nums text-text-faint">{section.items.length}</span>
				</button>
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
					aria-label="Delete section"
					onClick={onDelete}
					className="text-text-faint opacity-0 transition-opacity hover:text-destructive group-hover/sec:opacity-100"
				>
					<Trash2 size={12} />
				</button>
				<ChevronDown
					size={13}
					className={cn("text-text-faint transition-transform duration-200", section.collapsed && "-rotate-90")}
				/>
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
						"group/card relative flex cursor-grab items-center gap-3",
						"rounded-[10px] border border-border bg-surface px-3.5 py-3",
						"transition-colors hover:border-border-strong hover:bg-surface-hover active:scale-[0.98]",
						dnd.isDragging && "opacity-40",
					)}
				>
					<IconImg item={item} size={36} />
					<div className="min-w-0">
						<div className="truncate text-[13.5px] font-medium tracking-tight">{item.name}</div>
						{host ? <div className="truncate text-[11.5px] text-text-faint">{host}</div> : null}
					</div>
					{item.urls.length > 1 && (
						<span className="shrink-0 rounded border border-border px-1 py-px text-[10px] tabular-nums text-text-faint">
							{item.urls.length}
						</span>
					)}
					{item.urls.length > 0 && <StatusChip status={status} variant="dot" />}
					<button
						type="button"
						aria-label={`Edit ${item.name}`}
						onClick={(e) => {
							e.stopPropagation();
							onEdit();
						}}
						className="absolute -right-1 -top-1 rounded-full border border-border bg-popover p-1 text-text-faint opacity-0 transition-opacity hover:text-text group-hover/card:opacity-100"
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
				"group/card relative col-span-2 flex cursor-grab items-center gap-3",
				"rounded-[10px] border border-border bg-surface px-3.5 py-3",
				"transition-colors hover:border-border-strong hover:bg-surface-hover active:scale-[0.98]",
				dnd.isDragging && "opacity-40",
			)}
		>
			<WidgetContent item={item} />
			<button
				type="button"
				aria-label={`Edit ${item.name}`}
				onClick={(e) => {
					e.stopPropagation();
					onEdit();
				}}
				className="absolute -right-1 -top-1 rounded-full border border-border bg-popover p-1 text-text-faint opacity-0 transition-opacity hover:text-text group-hover/card:opacity-100"
			>
				<Pencil size={10} />
			</button>
		</div>
	);
}

export const cards: RendererViews = {
	BoardView,
	SectionView,
	ItemView,
	WidgetView,
	itemsGridClass: "grid grid-cols-[repeat(auto-fill,minmax(200px,1fr))] gap-2.5",
};
