import { useState, type FormEvent } from "react";
import { Plus } from "lucide-react";
import { useBoard } from "./store";
import { BoardDnd, ItemSortZone, SectionSortList } from "./dnd";
import { renderers } from "@/renderers";
import type { Item } from "@/types";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

// Board is the renderer-agnostic glue: state + dnd + sort contexts wired once,
// the active renderer supplies the chrome.
export function Board({ onEditItem }: { onEditItem: (item: Item) => void }) {
	const board = useBoard();
	const views = renderers[board.renderer];
	const [addingSection, setAddingSection] = useState(false);
	const [sectionName, setSectionName] = useState("");

	async function submitSection(e: FormEvent) {
		e.preventDefault();
		const name = sectionName.trim();
		if (!name) return;
		await board.addSection(name);
		setSectionName("");
		setAddingSection(false);
	}

	if (!board.loaded) {
		return <main className="mx-auto max-w-[1140px] px-7 pb-24 pt-9" />;
	}

	if (board.sections.length === 0) {
		return (
			<main className="mx-auto flex max-w-[1140px] flex-col items-center gap-4 px-7 pb-24 pt-24">
				<p className="text-[13px] text-text-faint">No sections yet.</p>
				<AddSectionRow
					open={addingSection} setOpen={setAddingSection}
					name={sectionName} setName={setSectionName} onSubmit={submitSection}
				/>
			</main>
		);
	}

	return (
		<main className="mx-auto max-w-[1140px] px-7 pb-24 pt-9">
			<BoardDnd>
				<SectionSortList ids={board.sections.map((s) => s.id)}>
					<views.BoardView>
						{board.sections.map((sec) => (
							<views.SectionView
								key={sec.id}
								section={sec}
								onToggle={() => void board.patchSection(sec.id, { collapsed: !sec.collapsed })}
								onDelete={() => {
									if (window.confirm(`Delete section "${sec.name}" and its ${sec.items.length} items?`)) {
										void board.deleteSection(sec.id);
									}
								}}
							>
								<ItemSortZone section={sec} className={views.itemsGridClass}>
									{sec.items.map((it) =>
										it.kind === "widget" ? (
											<views.WidgetView key={it.id} item={it} sectionId={sec.id} onEdit={() => onEditItem(it)} />
										) : (
											<views.ItemView
												key={it.id}
												item={it}
												sectionId={sec.id}
												status={board.statuses[it.id]}
												onEdit={() => onEditItem(it)}
											/>
										),
									)}
								</ItemSortZone>
							</views.SectionView>
						))}
					</views.BoardView>
				</SectionSortList>
			</BoardDnd>

			<div className="mt-8 flex justify-center">
				<AddSectionRow
					open={addingSection} setOpen={setAddingSection}
					name={sectionName} setName={setSectionName} onSubmit={submitSection}
				/>
			</div>
		</main>
	);
}

// Inline add-section row — one field doesn't warrant a dialog.
function AddSectionRow({
	open, setOpen, name, setName, onSubmit,
}: {
	open: boolean; setOpen: (o: boolean) => void;
	name: string; setName: (n: string) => void;
	onSubmit: (e: FormEvent) => void;
}) {
	if (!open) {
		return (
			<Button type="button" variant="outline" size="sm" onClick={() => setOpen(true)}>
				<Plus size={12} /> Add section
			</Button>
		);
	}
	return (
		<form onSubmit={onSubmit} className="flex gap-2">
			<Input
				autoFocus
				value={name}
				onChange={(e) => setName(e.target.value)}
				placeholder="Section name"
				className="w-[200px]"
				onBlur={() => !name.trim() && setOpen(false)}
				onKeyDown={(e) => e.key === "Escape" && setOpen(false)}
			/>
			<Button type="submit" size="sm">Add</Button>
		</form>
	);
}
