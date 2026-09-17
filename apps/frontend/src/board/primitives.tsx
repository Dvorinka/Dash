import { useState, type ReactNode } from "react";
import { cn } from "@/lib/utils";
import type { Item, Status } from "@/types";
import { Popover, PopoverAnchor, PopoverContent } from "@/components/ui/popover";
import { clickIsDragTail } from "@/board/dnd";

/** host extracts the display host from a service URL. */
export function hostOf(url: string): string {
	try {
		const u = new URL(url);
		return u.host + (u.pathname !== "/" ? u.pathname.replace(/\/$/, "") : "");
	} catch {
		return url;
	}
}

/** IconImg renders the item icon or a letter tile fallback. */
export function IconImg({ item, size = 40, className }: { item: Item; size?: number; className?: string }) {
	const [failed, setFailed] = useState(false);
	const letter = item.name.trim().charAt(0).toUpperCase() || "?";
	return (
		<div
			className={cn("relative flex shrink-0 items-center justify-center overflow-hidden bg-icon-bg text-text-dim font-semibold", className)}
			style={{ width: size, height: size, borderRadius: Math.round(size * 0.225), fontSize: size * 0.375 }}
		>
			{!item.icon || failed ? (
				letter
			) : (
				<img
					src={item.icon}
					alt=""
					className="absolute inset-0 h-full w-full bg-icon-bg object-contain p-[15%]"
					onError={() => setFailed(true)}
					loading="lazy"
				/>
			)}
		</div>
	);
}

/** StatusChip — the only color on the board. Green up, amber down, faint when
    unknown (no URLs or status not yet probed). */
export function StatusChip({ status, variant = "chip" }: { status?: Status | undefined; variant?: "chip" | "dot" }) {
	const state = status?.status;
	if (variant === "dot") {
		return (
			<span
				className={cn(
					"size-1.5 shrink-0 rounded-full",
					state === "up" ? "bg-up" : state === "down" ? "bg-down" : "bg-text-faint/40",
				)}
				title={state ?? "unknown"}
			/>
		);
	}
	return (
		<span
			className={cn(
				"ml-auto flex shrink-0 items-center gap-1.5 rounded-[5px] border border-border px-[7px] py-[3px]",
				"font-mono text-[9.5px] tracking-[0.06em] text-text-faint",
			)}
		>
			<span className={cn("size-[5px] rounded-full", state === "up" ? "bg-up" : state === "down" ? "bg-down" : "bg-text-faint/40")} />
			<span className="max-[620px]:hidden">{state ?? "…"}</span>
		</span>
	);
}

/** ItemAnchor gives a tile its launch behavior: one URL opens directly,
    multiple URLs open a label-tagged chooser popover, zero URLs open edit. */
export function ItemAnchor({
	item,
	onEdit,
	children,
}: {
	item: Item;
	onEdit: () => void;
	children: (open: () => void) => ReactNode;
}) {
	const [open, setOpen] = useState(false);
	const urls = item.urls;

	const activate = () => {
		if (clickIsDragTail()) return;
		if (urls.length === 0) onEdit();
		else if (urls.length === 1 && urls[0]) window.open(urls[0].url, "_blank", "noopener");
		else setOpen(true);
	};

	if (urls.length < 2) return <>{children(activate)}</>;

	return (
		<Popover open={open} onOpenChange={setOpen}>
			<PopoverAnchor asChild>{children(activate)}</PopoverAnchor>
			<PopoverContent onOpenAutoFocus={(e) => e.preventDefault()}>
				{urls.map((u) => (
					<a
						key={u.id}
						href={u.url}
						target="_blank"
						rel="noopener noreferrer"
						onClick={() => setOpen(false)}
						className="flex items-center gap-2.5 rounded-md px-2.5 py-2 text-[13px] text-text transition-colors hover:bg-surface-hover"
					>
						{u.label ? (
							<span className="shrink-0 rounded bg-icon-bg px-1.5 py-0.5 text-[9.5px] font-semibold uppercase tracking-[0.08em] text-text-dim">
								{u.label}
							</span>
						) : null}
						<span className="ml-auto truncate font-mono text-[11.5px] text-text-faint">{hostOf(u.url)}</span>
					</a>
				))}
			</PopoverContent>
		</Popover>
	);
}
