"use client";

import { useState, useEffect } from "react";
import type { Service, ServiceUrl } from "@/lib/api/schema";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger, DropdownMenuSeparator } from "@/components/ui/dropdown-menu";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { MoreVertical, ExternalLink, Pencil, Trash2, Globe, Home, Settings } from "lucide-react";
import { cn } from "@/lib/utils";

function getInitials(name: string) {
  const words = name.trim().split(/\s+/);
  if (words.length >= 2) return (words[0][0] + words[1][0]).toUpperCase();
  return name.slice(0, 2).toUpperCase();
}

function extractHost(url: string) {
  try {
    return new URL(url).hostname;
  } catch {
    return url;
  }
}

function getIconUrl(service: Service) {
  if (service.iconUrl) return service.iconUrl;
  if (service.iconAssetId) return `/uploads/icons/${service.iconAssetId}`;
  return null;
}

function kindIcon(kind: string) {
  switch (kind) {
    case "local": return <Home className="h-3 w-3" />;
    case "external": return <Globe className="h-3 w-3" />;
    default: return <Settings className="h-3 w-3" />;
  }
}

function kindBadgeClass(kind: string) {
  switch (kind) {
    case "local": return "badge-local";
    case "external": return "badge-external";
    default: return "badge-custom";
  }
}

function useServicePing(url: string | undefined) {
  const [status, setStatus] = useState<"up" | "down" | "unknown">("unknown");

  useEffect(() => {
    if (!url) return;
    let cancelled = false;
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), 5000);

    fetch(url, { method: "HEAD", mode: "no-cors", signal: controller.signal })
      .then(() => { if (!cancelled) setStatus("up"); })
      .catch(() => { if (!cancelled) setStatus("down"); })
      .finally(() => clearTimeout(timer));

    return () => { cancelled = true; controller.abort(); };
  }, [url]);

  return status;
}

function StatusDot({ status }: { status: "up" | "down" | "unknown" }) {
  if (status === "unknown") return null;
  return (
    <span
      className={cn(
        "absolute -bottom-0.5 -right-0.5 h-2.5 w-2.5 rounded-full border-2 border-card",
        status === "up" && "bg-emerald-500",
        status === "down" && "bg-red-500"
      )}
      title={status === "up" ? "Online" : "Offline"}
    />
  );
}

function UrlPickerDialog({
  urls,
  open,
  onOpenChange,
}: {
  urls: ServiceUrl[];
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Open App</DialogTitle>
          <DialogDescription>Choose which URL to open</DialogDescription>
        </DialogHeader>
        <div className="flex flex-col gap-2">
          {urls.map((u) => (
            <a
              key={u.id}
              href={u.url}
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center justify-between rounded-xl border border-border bg-card px-4 py-3 text-sm transition-all hover:bg-accent hover:border-border"
              onClick={() => onOpenChange(false)}
            >
              <div className="flex items-center gap-3 min-w-0">
                <Badge variant="secondary" className={cn("gap-1 text-[10px] px-2 py-0.5 font-medium uppercase", kindBadgeClass(u.kind))}>
                  {kindIcon(u.kind)}
                  {u.kind}
                </Badge>
                <span className="font-medium truncate">{u.label}</span>
              </div>
              <div className="flex items-center gap-2 shrink-0">
                <span className="text-xs text-muted-foreground hidden sm:inline">{extractHost(u.url)}</span>
                <ExternalLink className="h-3.5 w-3.5 text-muted-foreground" />
              </div>
            </a>
          ))}
        </div>
      </DialogContent>
    </Dialog>
  );
}

export function ServiceCard({
  service,
  onEdit,
  onDelete,
}: {
  service: Service;
  onEdit: (s: Service) => void;
  onDelete: (id: string) => void;
}) {
  const [pickerOpen, setPickerOpen] = useState(false);

  const handleClick = () => {
    if (service.urls.length === 1) {
      window.open(service.urls[0].url, "_blank", "noopener,noreferrer");
    } else {
      setPickerOpen(true);
    }
  };

  const iconSrc = getIconUrl(service);
  const primaryUrl = service.urls.find((u) => u.isPrimary) || service.urls[0];
  const status = useServicePing(primaryUrl?.url);

  return (
    <>
      <Card
        className={cn(
          "service-card group relative cursor-pointer overflow-hidden",
          "aspect-square rounded-2xl border border-border bg-card shadow-[0px_0px_0px_1px_var(--color-border)] hover:bg-accent hover:shadow-border-hover",
        )}
        onClick={handleClick}
      >
        {/* Accent line at top */}
        <div className="absolute top-0 left-0 right-0 h-[2px] opacity-0 group-hover:opacity-100 transition-opacity bg-muted-foreground" />

        <div className="flex h-full flex-col items-center justify-center gap-2.5 p-4">

          {/* Icon container */}
          <div className="relative flex items-center justify-center transition-transform duration-300 group-hover:scale-110 h-12 w-12">
            {iconSrc ? (
              <img
                src={iconSrc}
                alt={service.name}
                className="h-full w-full object-contain drop-shadow-lg rounded-xl"
                onError={(e) => {
                  (e.target as HTMLImageElement).style.display = "none";
                  (e.target as HTMLImageElement).nextElementSibling?.classList.remove("hidden");
                }}
              />
            ) : null}
            <div
              className={cn(
                "flex h-full w-full items-center justify-center rounded-xl font-mono font-bold text-secondary-foreground bg-secondary text-sm",
                iconSrc && "hidden",
              )}
            >
              {getInitials(service.name)}
            </div>
            <StatusDot status={status} />
          </div>

          {/* App name */}
          <span className="max-w-full truncate text-center font-semibold leading-tight text-xs text-foreground">
            {service.name}
          </span>

          {/* URL indicator */}
          {primaryUrl && (
            <span className="text-[10px] text-muted-foreground truncate max-w-full hidden sm:block">
              {extractHost(primaryUrl.url)}
            </span>
          )}

          {/* URL kind badges */}
          {service.urls.length > 1 && (
            <div className="flex gap-1">
              {service.urls.slice(0, 3).map((u) => (
                <span
                  key={u.id}
                  className={cn(
                    "text-[9px] px-1.5 py-0.5 rounded-full font-medium uppercase tracking-wider",
                    kindBadgeClass(u.kind)
                  )}
                >
                  {u.kind}
                </span>
              ))}
            </div>
          )}
        </div>

        {/* Actions */}
        <div
          className="absolute right-2 top-2 opacity-0 transition-all group-hover:opacity-100"
          onClick={(e) => e.stopPropagation()}
        >
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                variant="ghost"
                size="icon"
                className="h-7 w-7 rounded-lg hover:bg-accent"
              >
                <MoreVertical className="h-3.5 w-3.5" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-40 rounded-xl">
              <DropdownMenuItem onClick={() => onEdit(service)} className="gap-2 text-xs">
                <Pencil className="h-3.5 w-3.5" /> Edit
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem className="gap-2 text-xs text-destructive" onClick={() => onDelete(service.id)}>
                <Trash2 className="h-3.5 w-3.5" /> Delete
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </Card>
      {service.urls.length > 1 && (
        <UrlPickerDialog urls={service.urls} open={pickerOpen} onOpenChange={setPickerOpen} />
      )}
    </>
  );
}
