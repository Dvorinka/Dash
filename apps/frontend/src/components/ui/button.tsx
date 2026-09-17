import * as React from "react";
import { Slot } from "@radix-ui/react-slot";
import { cva, type VariantProps } from "class-variance-authority";

import { cn } from "@/lib/utils";

const buttonVariants = cva(
	"inline-flex items-center justify-center gap-1.5 whitespace-nowrap rounded-md text-[13px] font-medium transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50 [&_svg]:size-3.5 [&_svg]:shrink-0 active:scale-[0.97]",
	{
		variants: {
			variant: {
				default: "bg-primary text-primary-foreground hover:bg-primary/90",
				outline: "border border-border bg-surface text-text-dim hover:border-border-strong hover:text-text",
				ghost: "text-text-dim hover:bg-surface-hover hover:text-text",
				destructive: "bg-destructive/10 text-destructive hover:bg-destructive/20",
			},
			size: {
				default: "h-[34px] px-3.5",
				sm: "h-8 px-3 text-xs",
				icon: "size-[34px]",
			},
		},
		defaultVariants: { variant: "default", size: "default" },
	},
);

export interface ButtonProps
	extends React.ButtonHTMLAttributes<HTMLButtonElement>,
		VariantProps<typeof buttonVariants> {
	asChild?: boolean;
}

function Button({ className, variant, size, asChild = false, ...props }: ButtonProps) {
	const Comp = asChild ? Slot : "button";
	return <Comp className={cn(buttonVariants({ variant, size, className }))} {...props} />;
}

export { Button, buttonVariants };
