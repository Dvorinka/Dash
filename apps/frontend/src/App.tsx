import { useEffect, useState } from "react";
import { Moon, Plus, Sun } from "lucide-react";

type Theme = "dark" | "light";

function initialTheme(): Theme {
	const saved = localStorage.getItem("dash.theme");
	return saved === "light" ? "light" : "dark";
}

export default function App() {
	const [theme, setTheme] = useState<Theme>(initialTheme);

	useEffect(() => {
		document.documentElement.dataset.theme = theme;
		localStorage.setItem("dash.theme", theme);
	}, [theme]);

	return (
		<div className="font-sans">
			<header className="sticky top-0 z-10 flex items-center gap-3.5 border-b border-border bg-bg/80 px-7 py-3.5 backdrop-blur-md">
				<div className="flex items-center gap-2.5 text-[15px] font-semibold tracking-tight">
					<svg width="18" height="18" viewBox="0 0 18 18" fill="none" aria-hidden>
						<rect x="1" y="1" width="7" height="7" rx="1.5" fill="currentColor" />
						<rect x="10" y="1" width="7" height="7" rx="1.5" fill="currentColor" opacity=".45" />
						<rect x="1" y="10" width="7" height="7" rx="1.5" fill="currentColor" opacity=".45" />
						<rect x="10" y="10" width="7" height="7" rx="1.5" fill="currentColor" />
					</svg>
					Dash
				</div>
				<button
					type="button"
					onClick={() => setTheme(theme === "dark" ? "light" : "dark")}
					aria-label="Toggle theme"
					className="ml-auto flex size-[34px] items-center justify-center rounded-lg border border-border bg-surface text-text-dim transition-colors hover:border-border-strong hover:text-text active:scale-95"
				>
					{theme === "dark" ? <Moon size={15} strokeWidth={1.8} /> : <Sun size={15} strokeWidth={1.8} />}
				</button>
				<button
					type="button"
					className="flex h-[34px] items-center gap-1.5 rounded-lg bg-text px-3.5 text-[13px] font-medium text-bg transition-transform active:scale-95"
				>
					<Plus size={13} strokeWidth={2.2} />
					Add service
				</button>
			</header>
			<main className="mx-auto max-w-[1140px] px-7 pb-24 pt-9">
				<p className="text-center text-[13px] text-text-faint">
					No sections yet. Add a service to get started.
				</p>
			</main>
		</div>
	);
}
