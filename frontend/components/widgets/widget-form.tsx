"use client";

import { useState } from "react";
import type { WidgetInstance, WidgetRequest } from "@/lib/api/schema";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { useCreateWidget, useUpdateWidget } from "@/lib/api/hooks";
import { Command, CommandEmpty, CommandGroup, CommandInput, CommandItem, CommandList } from "@/components/ui/command";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Badge } from "@/components/ui/badge";
import { Check, ChevronsUpDown, X } from "lucide-react";
import { cn } from "@/lib/utils";

const POPULAR_TIMEZONES = [
  "America/New_York", "America/Chicago", "America/Denver", "America/Los_Angeles",
  "America/Anchorage", "Pacific/Honolulu", "America/Sao_Paulo", "America/Argentina/Buenos_Aires",
  "Europe/London", "Europe/Paris", "Europe/Berlin", "Europe/Prague", "Europe/Moscow",
  "Asia/Dubai", "Asia/Kolkata", "Asia/Bangkok", "Asia/Shanghai", "Asia/Tokyo", "Asia/Seoul",
  "Australia/Sydney", "Australia/Melbourne", "Pacific/Auckland", "UTC",
];

const WIDGET_TYPES = ["clock", "image", "pihole", "memos", "immich"] as const;

interface WidgetFormProps {
  widget?: WidgetInstance | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function WidgetForm({ widget, open, onOpenChange }: WidgetFormProps) {
  const isEdit = !!widget;
  const createMut = useCreateWidget();
  const updateMut = useUpdateWidget();

  const [type, setType] = useState<string>(widget?.type || "clock");
  const [title, setTitle] = useState(widget?.title || "");
  const [enabled, setEnabled] = useState(widget?.enabled ?? true);
  const [selectedTzs, setSelectedTzs] = useState<string[]>(
    (widget?.config?.timezones as string[]) || [],
  );
  const [tzPopoverOpen, setTzPopoverOpen] = useState(false);
  const [imageUrl, setImageUrl] = useState((widget?.config?.imageUrl as string) || "");
  const [linkUrl, setLinkUrl] = useState((widget?.config?.linkUrl as string) || "");
  const [piholeBaseUrl, setPiholeBaseUrl] = useState((widget?.config?.baseUrl as string) || "");
  const [piholeApiToken, setPiholeApiToken] = useState((widget?.config?.apiToken as string) || "");
  const [memosBaseUrl, setMemosBaseUrl] = useState((widget?.config?.baseUrl as string) || "");
  const [memosApiToken, setMemosApiToken] = useState((widget?.config?.apiToken as string) || "");
  const [memosPageSize, setMemosPageSize] = useState(String((widget?.config?.pageSize as number) || 5));
  const [immichBaseUrl, setImmichBaseUrl] = useState((widget?.config?.baseUrl as string) || "");
  const [immichApiKey, setImmichApiKey] = useState((widget?.config?.apiKey as string) || "");
  const [error, setError] = useState("");

  const buildConfig = (): Record<string, unknown> => {
    switch (type) {
      case "clock":
        return { timezones: selectedTzs };
      case "image":
        return { imageUrl, linkUrl: linkUrl || null };
      case "pihole":
        return { baseUrl: piholeBaseUrl, apiToken: piholeApiToken };
      case "memos":
        return { baseUrl: memosBaseUrl, apiToken: memosApiToken, pageSize: parseInt(memosPageSize) || 5 };
      case "immich":
        return { baseUrl: immichBaseUrl, apiKey: immichApiKey };
      default:
        return {};
    }
  };

  const handleSubmit = async () => {
    if (!title.trim()) { setError("Title is required"); return; }
    if ((type === "pihole" || type === "memos") && !piholeBaseUrl && !memosBaseUrl) {
      setError("Base URL is required");
      return;
    }
    if (type === "immich" && !immichBaseUrl) {
      setError("Base URL is required");
      return;
    }
    if (type === "image" && !imageUrl) { setError("Image URL is required"); return; }

    const body: WidgetRequest = {
      type: type as WidgetRequest["type"],
      title: title.trim(),
      enabled,
      config: buildConfig() as WidgetRequest["config"],
    };

    try {
      if (isEdit && widget) {
        await updateMut.mutateAsync({ id: widget.id, ...body });
      } else {
        await createMut.mutateAsync(body);
      }
      onOpenChange(false);
      setError("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed");
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{isEdit ? "Edit Widget" : "Add Widget"}</DialogTitle>
          <DialogDescription>{isEdit ? "Update widget settings" : "Add a new widget to your dashboard"}</DialogDescription>
        </DialogHeader>
        <div className="flex flex-col gap-4 py-2">
          <div className="flex flex-col gap-1.5">
            <Label>Type</Label>
            <Select value={type} onValueChange={setType} disabled={isEdit}>
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>
                {WIDGET_TYPES.map((t) => (
                  <SelectItem key={t} value={t}>{t.charAt(0).toUpperCase() + t.slice(1)}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="widget-title">Title</Label>
            <Input id="widget-title" value={title} onChange={(e) => setTitle(e.target.value)} placeholder="My Widget" />
          </div>
          <div className="flex items-center gap-2">
            <Switch checked={enabled} onCheckedChange={setEnabled} />
            <Label>Enabled</Label>
          </div>

          {type === "clock" && (
            <div className="flex flex-col gap-1.5">
              <Label>Timezones</Label>
              <div className="flex flex-wrap gap-1 mb-1">
                {selectedTzs.map((tz) => (
                  <Badge key={tz} variant="secondary" className="gap-1 text-xs">
                    {tz.split("/").pop()?.replace("_", " ")}
                    <button
                      type="button"
                      className="ml-0.5 rounded-full hover:bg-accent"
                      onClick={() => setSelectedTzs((prev) => prev.filter((t) => t !== tz))}
                    >
                      <X className="h-2.5 w-2.5" />
                    </button>
                  </Badge>
                ))}
              </div>
              <Popover open={tzPopoverOpen} onOpenChange={setTzPopoverOpen}>
                <PopoverTrigger asChild>
                  <Button variant="outline" type="button" className="justify-between text-xs font-normal">
                    Add timezone…
                    <ChevronsUpDown className="ml-2 h-3 w-3 shrink-0 opacity-50" />
                  </Button>
                </PopoverTrigger>
                <PopoverContent className="w-64 p-0" align="start">
                  <Command>
                    <CommandInput placeholder="Search timezone…" />
                    <CommandList>
                      <CommandEmpty>No timezone found.</CommandEmpty>
                      <CommandGroup>
                        {POPULAR_TIMEZONES.filter((tz) => !selectedTzs.includes(tz)).map((tz) => (
                          <CommandItem
                            key={tz}
                            value={tz}
                            onSelect={() => {
                              setSelectedTzs((prev) => [...prev, tz]);
                              setTzPopoverOpen(false);
                            }}
                          >
                            <Check className={cn("mr-2 h-3 w-3", selectedTzs.includes(tz) ? "opacity-100" : "opacity-0")} />
                            {tz}
                          </CommandItem>
                        ))}
                      </CommandGroup>
                    </CommandList>
                  </Command>
                </PopoverContent>
              </Popover>
            </div>
          )}
          {type === "image" && (
            <>
              <div className="flex flex-col gap-1.5">
                <Label>Image URL</Label>
                <Input value={imageUrl} onChange={(e) => setImageUrl(e.target.value)} placeholder="https://example.com/image.jpg" />
              </div>
              <div className="flex flex-col gap-1.5">
                <Label>Link URL (optional)</Label>
                <Input value={linkUrl} onChange={(e) => setLinkUrl(e.target.value)} placeholder="https://example.com" />
              </div>
            </>
          )}
          {type === "pihole" && (
            <>
              <div className="flex flex-col gap-1.5">
                <Label>Pi-hole Base URL</Label>
                <Input value={piholeBaseUrl} onChange={(e) => setPiholeBaseUrl(e.target.value)} placeholder="http://pihole.local" />
              </div>
              <div className="flex flex-col gap-1.5">
                <Label>API Token</Label>
                <Input type="password" value={piholeApiToken} onChange={(e) => setPiholeApiToken(e.target.value)} />
              </div>
            </>
          )}
          {type === "memos" && (
            <>
              <div className="flex flex-col gap-1.5">
                <Label>Memos Base URL</Label>
                <Input value={memosBaseUrl} onChange={(e) => setMemosBaseUrl(e.target.value)} placeholder="http://memos.local:5230" />
              </div>
              <div className="flex flex-col gap-1.5">
                <Label>API Token</Label>
                <Input type="password" value={memosApiToken} onChange={(e) => setMemosApiToken(e.target.value)} />
              </div>
              <div className="flex flex-col gap-1.5">
                <Label>Page Size</Label>
                <Input type="number" value={memosPageSize} onChange={(e) => setMemosPageSize(e.target.value)} min={1} max={20} />
              </div>
            </>
          )}
          {type === "immich" && (
            <>
              <div className="flex flex-col gap-1.5">
                <Label>Immich Base URL</Label>
                <Input value={immichBaseUrl} onChange={(e) => setImmichBaseUrl(e.target.value)} placeholder="http://immich.local:2283" />
              </div>
              <div className="flex flex-col gap-1.5">
                <Label>API Key</Label>
                <Input type="password" value={immichApiKey} onChange={(e) => setImmichApiKey(e.target.value)} />
              </div>
            </>
          )}
          {error && <span className="text-xs text-destructive">{error}</span>}
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
          <Button onClick={handleSubmit} disabled={createMut.isPending || updateMut.isPending}>
            {isEdit ? "Save" : "Add Widget"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
