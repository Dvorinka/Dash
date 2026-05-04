"use client";

import { useState, useRef } from "react";
import type { Service, ServiceUrlInput, ServiceRequest } from "@/lib/api/schema";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Plus, Trash2, Upload, Star } from "lucide-react";
import { useCreateService, useUpdateService, useUploadIcon } from "@/lib/api/hooks";

interface ServiceFormProps {
  service?: Service | null;
  groups: { id: string; name: string }[];
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

const EMPTY_URL: ServiceUrlInput = { label: "", kind: "local", url: "", isPrimary: false };

export function ServiceForm({ service, groups, open, onOpenChange }: ServiceFormProps) {
  const isEdit = !!service;
  const createMut = useCreateService();
  const updateMut = useUpdateService();
  const uploadMut = useUploadIcon();

  const [name, setName] = useState(service?.name || "");
  const [groupId, setGroupId] = useState<string | null>(service?.groupId || null);
  const [iconUrl, setIconUrl] = useState(service?.iconUrl || "");
  const [iconAssetId, setIconAssetId] = useState<string | null>(service?.iconAssetId || null);
  const [iconMode, setIconMode] = useState<"url" | "upload">("url");
  const [urls, setUrls] = useState<ServiceUrlInput[]>(
    service?.urls?.map((u) => ({ id: u.id, label: u.label, kind: u.kind, url: u.url, isPrimary: u.isPrimary })) || [{ ...EMPTY_URL, isPrimary: true }],
  );
  const [errors, setErrors] = useState<Record<string, string>>({});
  const fileRef = useRef<HTMLInputElement>(null);

  const addUrl = () => setUrls((prev) => [...prev, { ...EMPTY_URL }]);
  const removeUrl = (idx: number) => setUrls((prev) => prev.filter((_, i) => i !== idx));
  const updateUrl = (idx: number, field: keyof ServiceUrlInput, value: string | boolean) => {
    setUrls((prev) => {
      const next = [...prev];
      next[idx] = { ...next[idx], [field]: value };
      if (field === "isPrimary" && value === true) {
        next.forEach((u, i) => {
          if (i !== idx) u.isPrimary = false;
        });
      }
      return next;
    });
  };

  const handleFileUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    try {
      const asset = await uploadMut.mutateAsync(file);
      setIconAssetId(asset.id);
      setIconUrl("");
    } catch {
      setErrors((prev) => ({ ...prev, icon: "Upload failed" }));
    }
  };

  const validate = (): boolean => {
    const e: Record<string, string> = {};
    if (!name.trim()) e.name = "Name is required";
    if (urls.length === 0) e.urls = "At least one URL is required";
    urls.forEach((u, i) => {
      if (!u.label.trim()) e[`url-label-${i}`] = "Label required";
      if (!u.url.trim()) e[`url-${i}`] = "URL required";
      else if (!/^https?:\/\//.test(u.url)) e[`url-${i}`] = "Must be http(s)";
    });
    setErrors(e);
    return Object.keys(e).length === 0;
  };

  const handleSubmit = async () => {
    if (!validate()) return;
    const body: ServiceRequest = {
      name: name.trim(),
      groupId,
      iconUrl: iconMode === "url" && iconUrl ? iconUrl : null,
      iconAssetId: iconMode === "upload" && iconAssetId ? iconAssetId : null,
      urls: urls.map((u) => ({ label: u.label.trim(), kind: u.kind, url: u.url.trim(), isPrimary: u.isPrimary })),
    };
    try {
      if (isEdit && service) {
        await updateMut.mutateAsync({ id: service.id, ...body });
      } else {
        await createMut.mutateAsync(body);
      }
      onOpenChange(false);
    } catch (err) {
      setErrors({ submit: err instanceof Error ? err.message : "Failed" });
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{isEdit ? "Edit App" : "Add App"}</DialogTitle>
          <DialogDescription>{isEdit ? "Update app details" : "Add a new app to your dashboard"}</DialogDescription>
        </DialogHeader>
        <div className="flex flex-col gap-4 py-2">
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="name">Name</Label>
            <Input id="name" value={name} onChange={(e) => setName(e.target.value)} placeholder="Jellyfin" />
            {errors.name && <span className="text-xs text-destructive">{errors.name}</span>}
          </div>

          <div className="flex flex-col gap-1.5">
            <Label>Icon</Label>
            <div className="flex gap-2">
              <Button type="button" variant={iconMode === "url" ? "secondary" : "ghost"} size="sm" onClick={() => setIconMode("url")}>
                URL
              </Button>
              <Button type="button" variant={iconMode === "upload" ? "secondary" : "ghost"} size="sm" onClick={() => setIconMode("upload")}>
                Upload
              </Button>
            </div>
            {iconMode === "url" ? (
              <Input value={iconUrl} onChange={(e) => setIconUrl(e.target.value)} placeholder="https://example.com/icon.png" />
            ) : (
              <div className="flex items-center gap-2">
                <input ref={fileRef} type="file" accept="image/*" className="hidden" onChange={handleFileUpload} />
                <Button type="button" variant="outline" size="sm" onClick={() => fileRef.current?.click()}>
                  <Upload className="h-3 w-3" /> Choose file
                </Button>
                {iconAssetId && <span className="text-xs text-muted-foreground">Uploaded</span>}
              </div>
            )}
            {errors.icon && <span className="text-xs text-destructive">{errors.icon}</span>}
          </div>

          <div className="flex flex-col gap-1.5">
            <Label>Group</Label>
            <Select value={groupId || "__none__"} onValueChange={(v: string) => setGroupId(v === "__none__" ? null : v)}>
              <SelectTrigger><SelectValue placeholder="No group" /></SelectTrigger>
              <SelectContent>
                <SelectItem value="__none__">No group</SelectItem>
                {groups.map((g) => (
                  <SelectItem key={g.id} value={g.id}>{g.name}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="flex flex-col gap-2">
            <div className="flex items-center justify-between">
              <Label>URLs</Label>
              <Button type="button" variant="ghost" size="sm" onClick={addUrl}>
                <Plus className="h-3 w-3" /> Add URL
              </Button>
            </div>
            {urls.map((u, i) => (
              <div key={i} className="flex flex-col gap-1.5 rounded-md border border-border p-2">
                <div className="flex items-center gap-2">
                  <Input className="flex-1" value={u.label} onChange={(e) => updateUrl(i, "label", e.target.value)} placeholder="Label" />
                  <Select value={u.kind} onValueChange={(v: string) => updateUrl(i, "kind", v)}>
                    <SelectTrigger className="w-28"><SelectValue /></SelectTrigger>
                    <SelectContent>
                      <SelectItem value="local">Local</SelectItem>
                      <SelectItem value="external">External</SelectItem>
                      <SelectItem value="custom">Custom</SelectItem>
                    </SelectContent>
                  </Select>
                  <Button type="button" variant="ghost" size="icon" className="h-8 w-8 shrink-0" onClick={() => removeUrl(i)}>
                    <Trash2 className="h-3 w-3" />
                  </Button>
                </div>
                <div className="flex items-center gap-2">
                  <Input className="flex-1" value={u.url} onChange={(e) => updateUrl(i, "url", e.target.value)} placeholder="https://" />
                  <Button
                    type="button"
                    variant={u.isPrimary ? "secondary" : "ghost"}
                    size="icon"
                    className="h-8 w-8 shrink-0"
                    onClick={() => updateUrl(i, "isPrimary", !u.isPrimary)}
                    title="Primary URL"
                  >
                    <Star className={u.isPrimary ? "h-3 w-3 fill-current" : "h-3 w-3"} />
                  </Button>
                </div>
                {errors[`url-label-${i}`] && <span className="text-xs text-destructive">{errors[`url-label-${i}`]}</span>}
                {errors[`url-${i}`] && <span className="text-xs text-destructive">{errors[`url-${i}`]}</span>}
              </div>
            ))}
          </div>

          {errors.submit && <span className="text-xs text-destructive">{errors.submit}</span>}
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
          <Button onClick={handleSubmit} disabled={createMut.isPending || updateMut.isPending}>
            {isEdit ? "Save" : "Add App"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
