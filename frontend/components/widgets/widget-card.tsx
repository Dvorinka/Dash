"use client";

import type { WidgetInstance, WidgetData } from "@/lib/api/schema";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { MoreVertical, RefreshCw, Pencil, Trash2, GripVertical, Clock, Shield, ImageIcon, StickyNote, Camera, Activity } from "lucide-react";
import { useWidgetData, useRefreshWidget } from "@/lib/api/hooks";
import { cn } from "@/lib/utils";
import { useTheme } from "@/components/providers";

const widgetTypeIcons: Record<string, React.ReactNode> = {
  clock: <Clock className="h-3.5 w-3.5" />,
  pihole: <Shield className="h-3.5 w-3.5" />,
  image: <ImageIcon className="h-3.5 w-3.5" />,
  memos: <StickyNote className="h-3.5 w-3.5" />,
  immich: <Camera className="h-3.5 w-3.5" />,
};

const widgetTypeColors: Record<string, string> = {
  clock: "from-blue-500/20 to-cyan-500/20",
  pihole: "from-emerald-500/20 to-teal-500/20",
  image: "from-purple-500/20 to-pink-500/20",
  memos: "from-amber-500/20 to-orange-500/20",
  immich: "from-rose-500/20 to-red-500/20",
};

export function WidgetCard({
  widget,
  onEdit,
  onDelete,
  dragHandleProps,
}: {
  widget: WidgetInstance;
  onEdit: (w: WidgetInstance) => void;
  onDelete: (id: string) => void;
  dragHandleProps?: React.HTMLAttributes<HTMLDivElement>;
}) {
  const { data, isLoading, error } = useWidgetData(widget.id);
  const refreshMut = useRefreshWidget();
  const { theme } = useTheme();
  const isCasaOS = theme === "casaos";

  const handleRefresh = () => refreshMut.mutate(widget.id);

  const statusLabel = data?.status === "stale" ? "stale" : data?.status === "error" ? "error" : "";
  const typeIcon = widgetTypeIcons[widget.type] || <Activity className="h-3.5 w-3.5" />;
  const typeGradient = widgetTypeColors[widget.type] || "from-muted to-muted";

  return (
    <Card className={cn(
      "group relative border-0 overflow-hidden",
      isCasaOS
        ? "rounded-[20px] bg-card border border-border shadow-[0_4px_16px_rgba(0,0,0,0.15)] hover:shadow-[0_8px_32px_rgba(0,0,0,0.25)] hover:-translate-y-[2px] transition-all duration-300"
        : "rounded-2xl shadow-[0px_0px_0px_1px_var(--color-border)] hover:shadow-border-hover transition-all duration-200"
    )}>
      <div className={cn(
        "absolute top-0 left-0 right-0 h-1 opacity-60",
        isCasaOS ? `bg-gradient-to-r ${typeGradient}` : "bg-gradient-to-r from-ring/40 to-transparent"
      )} />
      <CardHeader className={cn("flex flex-row items-center justify-between pt-4 pb-2", isCasaOS ? "px-5" : "px-4")}>
        <div className="flex items-center gap-2.5 min-w-0">
          {dragHandleProps && (
            <div {...dragHandleProps} className="cursor-grab opacity-0 group-hover:opacity-60 transition-opacity rounded-md p-0.5 hover:bg-accent">
              <GripVertical className={cn("text-muted-foreground", isCasaOS ? "h-5 w-5" : "h-4 w-4")} />
            </div>
          )}
          <div className={cn("flex h-6 w-6 items-center justify-center rounded-md shrink-0", isCasaOS ? "bg-white/10" : "bg-accent")}>
            {typeIcon}
          </div>
          <div className="flex items-center gap-1.5 min-w-0">
            <CardTitle className="text-xs font-semibold uppercase tracking-wide truncate">
              {widget.title}
            </CardTitle>
            {statusLabel && (
              <span className={cn(
                "text-[9px] px-1.5 py-0.5 rounded-full font-medium uppercase shrink-0",
                statusLabel === "stale" ? "bg-amber-500/15 text-amber-400" : "bg-destructive/15 text-destructive"
              )}>
                {statusLabel}
              </span>
            )}
          </div>
        </div>
        <div className="flex items-center gap-0.5 shrink-0">
          <Button variant="ghost" size="icon" className={cn("relative z-10 pointer-events-auto rounded-lg h-7 w-7", isCasaOS ? "text-white/50 hover:text-white hover:bg-white/10" : "hover:bg-accent")} onClick={handleRefresh} disabled={refreshMut.isPending}>
            <RefreshCw className={cn(refreshMut.isPending && "animate-spin", isCasaOS ? "h-4 w-4" : "h-3.5 w-3.5")} />
          </Button>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="icon" className={cn("rounded-lg h-7 w-7", isCasaOS ? "text-white/50 hover:text-white hover:bg-white/10" : "hover:bg-accent")}>
                <MoreVertical className="h-3.5 w-3.5" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-40 rounded-xl">
              <DropdownMenuItem onClick={() => onEdit(widget)} className="gap-2 text-xs">
                <Pencil className="h-3.5 w-3.5" /> Edit
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem className="gap-2 text-xs text-destructive" onClick={() => onDelete(widget.id)}>
                <Trash2 className="h-3.5 w-3.5" /> Delete
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </CardHeader>
      <CardContent className={cn(isCasaOS ? "px-5 pb-5 pt-1" : "px-4 pb-4 pt-1")}>
        {isLoading ? (
          <span className="font-mono text-xs text-muted-foreground">[LOADING...]</span>
        ) : error || data?.status === "error" ? (
          <span className="font-mono text-xs text-destructive">[ERROR: {data?.error || "Failed to load"}]</span>
        ) : (
          <WidgetContent widget={widget} data={data} />
        )}
      </CardContent>
    </Card>
  );
}

function WidgetContent({ widget, data }: { widget: WidgetInstance; data?: WidgetData }) {
  switch (widget.type) {
    case "clock":
      return <ClockContent config={widget.config} data={data} />;
    case "image":
      return <ImageContent config={widget.config} />;
    case "pihole":
      return <PiHoleContent data={data} />;
    case "memos":
      return <MemosContent data={data} />;
    case "immich":
      return <ImmichContent data={data} />;
    default:
      return <span className="font-mono text-xs text-muted-foreground">Unknown widget type</span>;
  }
}

function ClockContent({ config }: { config: Record<string, unknown>; data?: WidgetData }) {
  const timezones = (config.timezones as string[]) || [];
  const now = new Date();
  const localTime = now.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit" });
  const localDate = now.toLocaleDateString([], { weekday: "long", month: "long", day: "numeric" });

  return (
    <div className="flex flex-col gap-2">
      <div className="font-mono text-3xl tabular-nums tracking-tight text-foreground">{localTime}</div>
      <div className="text-xs text-muted-foreground font-medium">{localDate}</div>
      {timezones.length > 0 && (
        <div className="mt-2 flex flex-col gap-1.5 border-t border-border/30 pt-2">
          {timezones.map((tz) => {
            try {
              const t = new Date().toLocaleTimeString([], { timeZone: tz, hour: "2-digit", minute: "2-digit" });
              return (
                <div key={tz} className="flex items-center justify-between text-xs">
                  <span className="text-muted-foreground text-[11px]">{tz.split("/").pop()?.replace("_", " ")}</span>
                  <span className="font-mono tabular-nums text-foreground">{t}</span>
                </div>
              );
            } catch {
              return null;
            }
          })}
        </div>
      )}
    </div>
  );
}

function ImageContent({ config }: { config: Record<string, unknown> }) {
  const imageUrl = config.imageUrl as string;
  const linkUrl = config.linkUrl as string | null;

  const img = (
    <img
      src={imageUrl}
      alt="Widget image"
      className="max-h-48 w-full rounded-xl object-cover border border-border/20 shadow-sm"
      onError={(e) => {
        (e.target as HTMLImageElement).style.display = "none";
      }}
    />
  );

  if (linkUrl) {
    return <a href={linkUrl} target="_blank" rel="noopener noreferrer" className="block rounded-xl overflow-hidden">{img}</a>;
  }
  return img;
}

function PiHoleContent({ data }: { data?: WidgetData }) {
  const d = data?.data as Record<string, unknown> | undefined;
  if (!d) return <span className="font-mono text-xs text-muted-foreground">No data</span>;

  return (
    <div className="grid grid-cols-2 gap-3">
      <div className="rounded-lg bg-emerald-500/10 p-2.5">
        <div className="text-[10px] uppercase tracking-wider text-emerald-400 font-medium mb-0.5">Status</div>
        <div className={cn("text-sm font-semibold", d.status === "enabled" ? "text-emerald-400" : "text-destructive")}>
          {String(d.status || "unknown")}
        </div>
      </div>
      <div className="rounded-lg bg-blue-500/10 p-2.5">
        <div className="text-[10px] uppercase tracking-wider text-blue-400 font-medium mb-0.5">Blocked</div>
        <div className="font-mono text-sm font-semibold text-foreground">{String(d.ads_blocked_today || "0")}</div>
      </div>
      <div className="rounded-lg bg-purple-500/10 p-2.5">
        <div className="text-[10px] uppercase tracking-wider text-purple-400 font-medium mb-0.5">Queries</div>
        <div className="font-mono text-sm font-semibold text-foreground">{String(d.dns_queries_today || "0")}</div>
      </div>
      <div className="rounded-lg bg-amber-500/10 p-2.5">
        <div className="text-[10px] uppercase tracking-wider text-amber-400 font-medium mb-0.5">% Blocked</div>
        <div className="font-mono text-sm font-semibold text-foreground">{String(d.ads_percentage_today || "0")}%</div>
      </div>
    </div>
  );
}

function MemosContent({ data }: { data?: WidgetData }) {
  const d = data?.data as Record<string, unknown> | undefined;
  const memos = (d?.memos as Array<Record<string, unknown>>) || [];
  if (memos.length === 0) return <span className="font-mono text-xs text-muted-foreground">No memos</span>;

  return (
    <div className="flex flex-col gap-2 max-h-40 overflow-y-auto pr-1">
      {memos.slice(0, 5).map((m, i) => (
        <div key={i} className="rounded-lg bg-amber-500/10 p-2.5 border border-amber-500/10">
          <div className="text-[11px] leading-relaxed line-clamp-2 text-foreground/90">
            {String(m.content || m.snippet || "")}
          </div>
        </div>
      ))}
    </div>
  );
}

function ImmichContent({ data }: { data?: WidgetData }) {
  const d = data?.data as Record<string, unknown> | undefined;
  if (!d) return <span className="font-mono text-xs text-muted-foreground">No data</span>;

  return (
    <div className="grid grid-cols-2 gap-3">
      <div className="rounded-lg bg-blue-500/10 p-2.5">
        <div className="text-[10px] uppercase tracking-wider text-blue-400 font-medium mb-0.5">Photos</div>
        <div className="font-mono text-sm font-semibold text-foreground">{String(d.photos || "0")}</div>
      </div>
      <div className="rounded-lg bg-rose-500/10 p-2.5">
        <div className="text-[10px] uppercase tracking-wider text-rose-400 font-medium mb-0.5">Videos</div>
        <div className="font-mono text-sm font-semibold text-foreground">{String(d.videos || "0")}</div>
      </div>
    </div>
  );
}
