import type { ComponentType, ReactNode } from "react";
import type { Item, Section, Status } from "@/types";

// Renderer contract — same board state, shared dnd/popover/dialog primitives,
// different structure per style. New renderers implement this shape only.
// Dnd stays shared: views call useSectionDnd / useItemDnd hooks internally.

export interface SectionViewProps {
	section: Section;
	index: number;
	onToggle(): void;
	onDelete(): void;
	children: ReactNode; // rendered items grid (ItemSortZone output)
}

export interface ItemViewProps {
	item: Item;
	index: number;
	sectionId: string;
	status?: Status | undefined;
	onEdit(): void;
}

export interface BoardViewProps {
	children: ReactNode; // rendered sections
}

export interface RendererViews {
	/** Arrangement wrapper around the section list. */
	BoardView: ComponentType<BoardViewProps>;
	/** Section chrome: label, count, collapse, drag grip. */
	SectionView: ComponentType<SectionViewProps>;
	/** Service tile anatomy: icon, name, host, status, launch anchor. */
	ItemView: ComponentType<ItemViewProps>;
	/** Widget tile anatomy (Phase 2 fills in). */
	WidgetView: ComponentType<ItemViewProps>;
	/** Tailwind classes for the items grid inside a section. */
	itemsGridClass: string;
}
