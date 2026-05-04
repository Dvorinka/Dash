"use client";

import { ThemeToggle } from "./theme-toggle";
import { Button } from "@/components/ui/button";
import { Plus, LayoutGrid, AppWindow, Puzzle } from "lucide-react";
import { useState, useEffect } from "react";

export function Header({
  onAddService,
  onAddWidget,
  onAddGroup,
}: {
  onAddService: () => void;
  onAddWidget: () => void;
  onAddGroup: () => void;
}) {
  const [now, setNow] = useState(new Date());

  useEffect(() => {
    const id = setInterval(() => setNow(new Date()), 1000);
    return () => clearInterval(id);
  }, []);

  const timeStr = now.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
  const dateStr = now.toLocaleDateString([], { weekday: "short", month: "short", day: "numeric" });

  return (
    <header className="sticky top-0 z-40 w-full border-b border-border bg-background">
      <div className="mx-auto flex h-14 max-w-6xl items-center justify-between px-4">
        <div className="flex items-center gap-4">
          <div className="flex items-center gap-2">
            <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-secondary">
              <LayoutGrid className="h-4 w-4 text-primary" />
            </div>
            <span className="text-sm font-semibold tracking-tight text-foreground">
              Dash
            </span>
          </div>
          <div className="hidden h-4 w-px bg-border sm:block" />
          <div className="hidden items-center gap-2 sm:flex">
            <span className="text-xs text-muted-foreground">
              {dateStr}
            </span>
            <span className="font-mono text-xs tabular-nums text-muted-foreground">
              {timeStr}
            </span>
          </div>
        </div>
        <div className="flex items-center gap-1">
          <Button variant="ghost" size="sm" onClick={onAddWidget} className="gap-1.5 text-xs text-muted-foreground hover:text-foreground">
            <Puzzle className="h-3.5 w-3.5" />
            <span className="hidden sm:inline">Widget</span>
          </Button>
          <Button variant="ghost" size="sm" onClick={onAddGroup} className="gap-1.5 text-xs text-muted-foreground hover:text-foreground">
            <AppWindow className="h-3.5 w-3.5" />
            <span className="hidden sm:inline">Group</span>
          </Button>
          <Button variant="default" size="sm" onClick={onAddService} className="gap-1.5 text-xs">
            <Plus className="h-3.5 w-3.5" />
            <span className="hidden sm:inline">App</span>
          </Button>
          <div className="ml-1 h-4 w-px bg-border" />
          <ThemeToggle />
        </div>
      </div>
    </header>
  );
}
