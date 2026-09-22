import { useState } from "react";
import { useLocation } from "wouter";
import { Check, ChevronsUpDown, Pencil, Trash2 } from "lucide-react";
import { useBoard } from "@/board/store";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { t } from "@/i18n";

// BoardSwitcher — header control for navigating boards (/b/<slug>) and
// managing them. The store owns the data; this only routes and calls API.
export function BoardSwitcher() {
	const board = useBoard();
	const [, nav] = useLocation();
	const [manage, setManage] = useState(false);

	if (board.boards.length === 0) return null;

	return (
		<>
			<Select value={board.boardSlug} onValueChange={(slug) => nav(slug === board.boards[0]?.slug ? "/" : `/b/${slug}`)}>
				<SelectTrigger className="h-8 w-auto min-w-[110px] gap-1 border-none bg-transparent px-2 text-[13px] font-semibold shadow-none">
					<SelectValue />
					<ChevronsUpDown size={12} className="text-text-faint" />
				</SelectTrigger>
				<SelectContent>
					{board.boards.map((b) => (
						<SelectItem key={b.id} value={b.slug}>{b.name}</SelectItem>
					))}
					<button
						type="button"
						className="mt-1 w-full rounded-[6px] border-t border-border px-2 py-1.5 text-left text-[12px] text-text-faint hover:text-text"
						onClick={() => setManage(true)}
					>
						{t("boards.manage")}
					</button>
				</SelectContent>
			</Select>
			<BoardManageDialog open={manage} onOpenChange={setManage} />
		</>
	);
}

function BoardManageDialog({ open, onOpenChange }: { open: boolean; onOpenChange: (o: boolean) => void }) {
	const board = useBoard();
	const [, nav] = useLocation();
	const [name, setName] = useState("");
	const [renaming, setRenaming] = useState<string | null>(null);
	const [renameTo, setRenameTo] = useState("");

	async function create() {
		const n = name.trim();
		if (!n) return;
		const b = await board.addBoard(n);
		setName("");
		if (b) {
			onOpenChange(false);
			nav(`/b/${b.slug}`);
		}
	}

	return (
		<Dialog open={open} onOpenChange={onOpenChange}>
			<DialogContent className="max-w-sm">
				<DialogHeader>
					<DialogTitle>{t("boards.title")}</DialogTitle>
				</DialogHeader>
				<div className="grid gap-2">
					{board.boards.map((b) => (
						<div key={b.id} className="flex items-center gap-2 rounded-md border border-border px-2.5 py-1.5">
							{renaming === b.id ? (
								<>
									<Input
										value={renameTo} autoFocus
										onChange={(e) => setRenameTo(e.target.value)}
										onKeyDown={(e) => {
											if (e.key === "Enter") void board.renameBoard(b.id, renameTo.trim()).then(() => setRenaming(null));
											if (e.key === "Escape") setRenaming(null);
										}}
										className="h-7 text-[13px]"
									/>
									<Button
										variant="ghost" size="icon" className="h-7 w-7" aria-label={t("common.save")}
										onClick={() => void board.renameBoard(b.id, renameTo.trim()).then(() => setRenaming(null))}
									>
										<Check size={13} />
									</Button>
								</>
							) : (
								<>
									<span className="flex-1 text-[13px]">{b.name}</span>
									<span className="font-mono text-[10px] text-text-faint">/b/{b.slug}</span>
									<Button
										variant="ghost" size="icon" className="h-7 w-7" aria-label={t("common.edit")}
										onClick={() => { setRenaming(b.id); setRenameTo(b.name); }}
									>
										<Pencil size={12} />
									</Button>
									{board.boards.length > 1 && (
										<Button
											variant="ghost" size="icon" className="h-7 w-7 text-danger" aria-label={t("common.delete")}
											onClick={() => void board.deleteBoard(b.id).then(() => {
												if (b.slug === board.boardSlug) nav("/");
											})}
										>
											<Trash2 size={12} />
										</Button>
									)}
								</>
							)}
						</div>
					))}
					<div className="mt-1 flex gap-2">
						<Input
							value={name} placeholder={t("boards.newPlaceholder")}
							onChange={(e) => setName(e.target.value)}
							onKeyDown={(e) => { if (e.key === "Enter") void create(); }}
							className="h-8 text-[13px]"
						/>
						<Button size="sm" disabled={!name.trim()} onClick={() => void create()}>{t("common.add")}</Button>
					</div>
				</div>
			</DialogContent>
		</Dialog>
	);
}
