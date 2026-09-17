import * as React from "react";
import * as SelectPrimitive from "@radix-ui/react-select";
import { Check, ChevronDown } from "lucide-react";

import { cn } from "@/lib/utils";

const Select = SelectPrimitive.Root;
const SelectValue = SelectPrimitive.Value;

function SelectTrigger({ className, children, ...props }: React.ComponentProps<typeof SelectPrimitive.Trigger>) {
	return (
		<SelectPrimitive.Trigger
			className={cn(
				"flex h-9 w-full items-center justify-between rounded-md border border-border bg-surface px-3",
				"text-[13px] text-text focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring",
				"[&>span]:truncate",
				className,
			)}
			{...props}
		>
			{children}
			<SelectPrimitive.Icon asChild>
				<ChevronDown size={13} className="text-text-faint" />
			</SelectPrimitive.Icon>
		</SelectPrimitive.Trigger>
	);
}

function SelectContent({ className, children, ...props }: React.ComponentProps<typeof SelectPrimitive.Content>) {
	return (
		<SelectPrimitive.Portal>
			<SelectPrimitive.Content
				position="popper"
				sideOffset={6}
				className={cn(
					"z-50 min-w-[var(--radix-select-trigger-width)] overflow-hidden rounded-[10px]",
					"border border-border-strong bg-popover p-[5px] text-text shadow-xl",
					"data-[state=open]:animate-in data-[state=open]:fade-in-0 data-[state=open]:zoom-in-[0.98]",
					className,
				)}
				{...props}
			>
				<SelectPrimitive.Viewport>{children}</SelectPrimitive.Viewport>
			</SelectPrimitive.Content>
		</SelectPrimitive.Portal>
	);
}

function SelectItem({ className, children, ...props }: React.ComponentProps<typeof SelectPrimitive.Item>) {
	return (
		<SelectPrimitive.Item
			className={cn(
				"relative flex cursor-pointer select-none items-center rounded-md py-1.5 pl-8 pr-3 text-[13px]",
				"outline-none data-[highlighted]:bg-surface-hover",
				className,
			)}
			{...props}
		>
			<span className="absolute left-2 flex items-center">
				<SelectPrimitive.ItemIndicator>
					<Check size={13} />
				</SelectPrimitive.ItemIndicator>
			</span>
			<SelectPrimitive.ItemText>{children}</SelectPrimitive.ItemText>
		</SelectPrimitive.Item>
	);
}

export { Select, SelectTrigger, SelectValue, SelectContent, SelectItem };
