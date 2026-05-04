"use client";

import { useTheme } from "@/components/providers";
import { themeLabels, type Theme } from "@/lib/theme/themes";
import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { Sun, Moon, Sparkles, Check } from "lucide-react";
import { cn } from "@/lib/utils";

const themeIcons: Record<Theme, React.ReactNode> = {
  light: <Sun className="h-4 w-4" />,
  dark: <Moon className="h-4 w-4" />,
  casaos: <Sparkles className="h-4 w-4" />,
};

const themeDot: Record<Theme, string> = {
  light: "bg-amber-400",
  dark: "bg-indigo-400",
  casaos: "bg-pink-400",
};

export function ThemeToggle() {
  const { theme, setTheme } = useTheme();

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="icon" className="rounded-lg hover:bg-accent relative" aria-label="Toggle theme">
          <div className="relative">
            {themeIcons[theme]}
            <span className={cn("absolute -bottom-0.5 -right-0.5 h-1.5 w-1.5 rounded-full border-2 border-background", themeDot[theme])} />
          </div>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-40 rounded-xl">
        {(["light", "dark", "casaos"] as Theme[]).map((t) => (
          <DropdownMenuItem key={t} onClick={() => setTheme(t)} className={cn("gap-2.5 rounded-lg cursor-pointer", theme === t && "bg-accent")}>
            <span className={cn("flex h-5 w-5 items-center justify-center rounded-md", theme === t ? "text-foreground" : "text-muted-foreground")}>
              {themeIcons[t]}
            </span>
            <span className="text-sm">{themeLabels[t]}</span>
            {theme === t && <Check className="ml-auto h-3.5 w-3.5 text-foreground" />}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
