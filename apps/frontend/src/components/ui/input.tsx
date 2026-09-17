import * as React from "react";

import { cn } from "@/lib/utils";

function Input({ className, type, ...props }: React.ComponentProps<"input">) {
	return (
		<input
			type={type}
			className={cn(
				"flex h-9 w-full rounded-md border border-border bg-surface px-3 text-[13px] text-text",
				"placeholder:text-text-faint focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring",
				"disabled:cursor-not-allowed disabled:opacity-50",
				className,
			)}
			{...props}
		/>
	);
}

export { Input };
