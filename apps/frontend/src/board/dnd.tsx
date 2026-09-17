import {
	useState,
	type CSSProperties,
	type ReactNode,
} from "react";
import {
	DndContext,
	DragOverlay,
	PointerSensor,
	pointerWithin,
	rectIntersection,
	useDroppable,
	useSensor,
	useSensors,
	type CollisionDetection,
	type DraggableAttributes,
	type DraggableSyntheticListeners,
	type DragEndEvent,
	type DragOverEvent,
	type DragStartEvent,
} from "@dnd-kit/core";
import {
	SortableContext,
	arrayMove,
	rectSortingStrategy,
	useSortable,
	verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import type { Section } from "@/types";
import { useBoard } from "./store";

// Shared drag-drop primitives. Renderers consume the hooks; they never
// wire sensors, contexts, or persistence themselves.

interface DragData {
	type: "section" | "item" | "sectionBody";
	sectionId?: string;
}

/** dnd-kit types data.current as any; our hooks below only ever attach DragData. */
function dragData(d: { data: { current: unknown } }): DragData | undefined {
	// SAFETY: payloads are set exclusively by useItemDnd/useSectionDnd/ItemSortZone.
	return d.data.current as DragData | undefined;
}

const collision: CollisionDetection = (args) => {
	const p = pointerWithin(args);
	return p.length > 0 ? p : rectIntersection(args);
};

// Pointerup fires a click on whatever sits under the cursor, which would open
// a tile's URL popover right after a drop. Stamp the last real drag and let
// ItemAnchor swallow clicks that arrive immediately after it.
let lastDropAt = 0;
export function clickIsDragTail(): boolean {
	return performance.now() - lastDropAt < 200;
}

/** Wraps the whole board: sensors, drag lifecycle, overlay, persistence. */
export function BoardDnd({ children }: { children: ReactNode }) {
	const board = useBoard();
	const [activeLabel, setActiveLabel] = useState<string | null>(null);

	const sensors = useSensors(
		useSensor(PointerSensor, { activationConstraint: { distance: 6 } }),
	);

	function onDragStart(e: DragStartEvent) {
		const data = dragData(e.active);
		if (data?.type === "item") {
			const item = board.sections
				.flatMap((s) => s.items)
				.find((i) => i.id === e.active.id);
			setActiveLabel(item?.name ?? null);
		} else {
			const sec = board.sections.find((s) => s.id === e.active.id);
			setActiveLabel(sec?.name ?? null);
		}
	}

	function onDragOver(e: DragOverEvent) {
		const active = dragData(e.active);
		const over = e.over ? dragData(e.over) : undefined;
		if (!active || active.type !== "item" || !over || !e.over) return;

		const activeId = String(e.active.id);
		if (over.type === "item" && over.sectionId && over.sectionId !== active.sectionId) {
			const items = board.sections.find((s) => s.id === over.sectionId)?.items ?? [];
			const idx = items.findIndex((i) => i.id === String(e.over?.id));
			board.moveItemLocal(activeId, over.sectionId, idx < 0 ? items.length : idx);
		} else if (over.type === "sectionBody" && over.sectionId && over.sectionId !== active.sectionId) {
			const len = board.sections.find((s) => s.id === over.sectionId)?.items.length ?? 0;
			board.moveItemLocal(activeId, over.sectionId, len);
		} else if (over.type === "section" && over.sectionId !== active.sectionId) {
			const len = board.sections.find((s) => s.id === String(e.over?.id))?.items.length ?? 0;
			board.moveItemLocal(activeId, String(e.over?.id), len);
		}
	}

	function onDragEnd(e: DragEndEvent) {
		setActiveLabel(null);
		lastDropAt = performance.now();
		const active = dragData(e.active);
		const overId = e.over ? String(e.over.id) : null;
		if (!active || !overId) return;

		if (active.type === "section") {
			const over = e.over ? dragData(e.over) : undefined;
			// Dropping anywhere inside another section — its body or an item —
			// counts as reordering onto that section.
			const targetId =
				over?.type === "section" ? overId :
				over?.sectionId ?? null;
			if (!targetId) return;
			const ids = board.sections.map((s) => s.id);
			const from = ids.indexOf(String(e.active.id));
			const to = ids.indexOf(targetId);
			if (from < 0 || to < 0 || from === to) return;
			const next = arrayMove(board.sections, from, to);
			board.moveSectionLocal(String(e.active.id), to);
			const idx = next.findIndex((s) => s.id === e.active.id);
			void board.persistSectionMove(
				String(e.active.id),
				idx > 0 ? next[idx - 1]?.id : undefined,
				idx < next.length - 1 ? next[idx + 1]?.id : undefined,
			);
			return;
		}

		if (active.type === "item") {
			const id = String(e.active.id);
			const over = e.over ? dragData(e.over) : undefined;
			const sec = board.sections.find((s) => s.items.some((i) => i.id === id));
			if (!sec) return;

			// onDragOver only mutates local state for cross-section moves; for a
			// same-section drop the order is still pre-drag, so apply it now.
			let items = sec.items;
			const from = items.findIndex((i) => i.id === id);
			if (over?.type === "item" && over.sectionId === sec.id) {
				const to = items.findIndex((i) => i.id === overId);
				if (to >= 0 && to !== from) {
					items = arrayMove(items, from, to);
					board.moveItemLocal(id, sec.id, to);
				}
			} else if (over?.type === "sectionBody" && over.sectionId === sec.id) {
				items = arrayMove(items, from, items.length - 1);
				board.moveItemLocal(id, sec.id, items.length - 1);
			}

			const idx = items.findIndex((i) => i.id === id);
			void board.persistItemMove(
				id,
				sec.id,
				idx > 0 ? items[idx - 1]?.id : undefined,
				idx < items.length - 1 ? items[idx + 1]?.id : undefined,
			);
		}
	}

	function onDragCancel() {
		setActiveLabel(null);
		void board.refresh();
	}

	return (
		<DndContext
			sensors={sensors}
			collisionDetection={collision}
			onDragStart={onDragStart}
			onDragOver={onDragOver}
			onDragEnd={onDragEnd}
			onDragCancel={onDragCancel}
		>
			{children}
			<DragOverlay dropAnimation={null}>
				{activeLabel ? (
					<div className="rounded-[10px] border border-border-strong bg-popover px-4 py-2.5 text-[13px] font-medium shadow-xl">
						{activeLabel}
					</div>
				) : null}
			</DragOverlay>
		</DndContext>
	);
}

/** Sortable context for the section list (vertical). Shared, not per-renderer. */
export function SectionSortList({ ids, children }: { ids: string[]; children: ReactNode }) {
	return (
		<SortableContext items={ids} strategy={verticalListSortingStrategy}>
			{children}
		</SortableContext>
	);
}

/** Sortable context + droppable body for one section's items.
    The div carries the renderer's grid classes so the drop zone has a real box. */
export function ItemSortZone({ section, className, children }: { section: Section; className?: string; children: ReactNode }) {
	const { setNodeRef } = useDroppable({
		id: `body:${section.id}`,
		data: { type: "sectionBody", sectionId: section.id } satisfies DragData,
	});
	return (
		<SortableContext items={section.items.map((i) => i.id)} strategy={rectSortingStrategy}>
			<div ref={setNodeRef} className={className} style={{ minHeight: 8 }}>
				{children}
			</div>
		</SortableContext>
	);
}

export interface SortableBind {
	ref: (el: HTMLElement | null) => void;
	style: CSSProperties;
	isDragging: boolean;
	attributes: DraggableAttributes;
	listeners: DraggableSyntheticListeners;
}

/** Item-level sortable binding. ItemView applies ref/style and spreads
    attributes/listeners on the tile (whole-tile drag; clicks pass through
    because the pointer sensor requires 6px of movement). */
export function useItemDnd(sectionId: string, itemId: string): SortableBind {
	const { setNodeRef, attributes, listeners, transform, transition, isDragging } = useSortable({
		id: itemId,
		data: { type: "item", sectionId } satisfies DragData,
	});
	return {
		ref: setNodeRef,
		style: { transform: CSS.Transform.toString(transform), transition },
		isDragging,
		attributes,
		listeners,
	};
}

/** Section-level sortable binding. Handle props go on the grip button. */
export function useSectionDnd(sectionId: string): SortableBind {
	const { setNodeRef, attributes, listeners, transform, transition, isDragging } = useSortable({
		id: sectionId,
		data: { type: "section", sectionId } satisfies DragData,
	});
	return {
		ref: setNodeRef,
		style: { transform: CSS.Transform.toString(transform), transition },
		isDragging,
		attributes,
		listeners,
	};
}
