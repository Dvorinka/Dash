import * as React from "react";
import * as PopoverPrimitive from "@radix-ui/react-popover";

import { cn } from "@/lib/utils";

const Popover = PopoverPrimitive.Root;
const PopoverAnchor = PopoverPrimitive.Anchor;
const PopoverTrigger = PopoverPrimitive.Trigger;

function PopoverContent({
	className,
	align = "start",
	sideOffset = 6,
	...props
}: React.ComponentProps<typeof PopoverPrimitive.Content>) {
	return (
		<PopoverPrimitive.Portal>
			<PopoverPrimitive.Content
				align={align}
				sideOffset={sideOffset}
				className={cn(
					"z-30 min-w-[235px] rounded-[10px] border border-border-strong bg-popover p-[5px] text-text",
					"shadow-[0_8px_30px_rgba(0,0,0,0.35),0_1px_3px_rgba(0,0,0,0.2)]",
					"data-[state=open]:animate-in data-[state=open]:fade-in-0 data-[state=open]:zoom-in-[0.98]",
					className,
				)}
				{...props}
			/>
		</PopoverPrimitive.Portal>
	);
}

export { Popover, PopoverAnchor, PopoverTrigger, PopoverContent };
