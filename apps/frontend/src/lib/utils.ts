import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
	return twMerge(clsx(inputs));
}

/** Extract the {error} message from an openapi-fetch error payload. */
export function apiErr(err: unknown, fallback: string): string {
	// SAFETY: error responses always carry {error: string} per openapi.yaml Error schema.
	const e = err as { error?: string } | undefined;
	return e?.error ?? fallback;
}
