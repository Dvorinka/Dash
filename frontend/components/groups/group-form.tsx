"use client";

import { useState } from "react";
import type { Group } from "@/lib/api/schema";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useCreateGroup, useUpdateGroup } from "@/lib/api/hooks";

interface GroupFormProps {
  group?: Group | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function GroupForm({ group, open, onOpenChange }: GroupFormProps) {
  const isEdit = !!group;
  const createMut = useCreateGroup();
  const updateMut = useUpdateGroup();
  const [name, setName] = useState(group?.name || "");
  const [error, setError] = useState("");

  const handleSubmit = async () => {
    if (!name.trim()) {
      setError("Name is required");
      return;
    }
    try {
      if (isEdit && group) {
        await updateMut.mutateAsync({ id: group.id, name: name.trim() });
      } else {
        await createMut.mutateAsync({ name: name.trim() });
      }
      onOpenChange(false);
      setName("");
      setError("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed");
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{isEdit ? "Rename Group" : "Create Group"}</DialogTitle>
          <DialogDescription>{isEdit ? "Update group name" : "Add a new group for organizing apps"}</DialogDescription>
        </DialogHeader>
        <div className="flex flex-col gap-2 py-2">
          <Label htmlFor="group-name">Name</Label>
          <Input id="group-name" value={name} onChange={(e) => setName(e.target.value)} placeholder="Infrastructure" />
          {error && <span className="text-xs text-destructive">{error}</span>}
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
          <Button onClick={handleSubmit} disabled={createMut.isPending || updateMut.isPending}>
            {isEdit ? "Save" : "Create"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
