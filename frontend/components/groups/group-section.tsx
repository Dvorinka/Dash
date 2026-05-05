"use client";

import type { Group, Service } from "@/lib/api/schema";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { ServiceCard } from "@/components/services/service-card";
import { ChevronDown, MoreVertical, Pencil, Trash2, FolderOpen } from "lucide-react";
import { useUpdateGroup, useDeleteGroup } from "@/lib/api/hooks";
import { cn } from "@/lib/utils";
import { useState } from "react";

interface GroupSectionProps {
  group: Group;
  onEditService: (s: Service) => void;
  onDeleteService: (id: string) => void;
  onEditGroup: (g: Group) => void;
}

export function GroupSection({ group, onEditService, onDeleteService, onEditGroup }: GroupSectionProps) {
  const updateGroup = useUpdateGroup();
  const deleteGroup = useDeleteGroup();
  const [open, setOpen] = useState(!group.collapsed);

  const handleToggle = () => {
    const next = !open;
    setOpen(next);
    updateGroup.mutate({ id: group.id, collapsed: !next });
  };

  const handleDelete = () => {
    if (group.services.length > 0) {
      deleteGroup.mutate({ id: group.id, moveServices: true });
    } else {
      deleteGroup.mutate({ id: group.id });
    }
  };

  return (
    <Collapsible open={open} onOpenChange={setOpen}>
      <Card className="mb-4 overflow-hidden">
        {/* Group header */}
        <div className="flex items-center gap-2 px-4 py-3">
          <CollapsibleTrigger asChild>
            <button
              className="flex flex-1 items-center gap-2.5 group/title min-w-0"
              onClick={handleToggle}
            >
              <div className="flex h-7 w-7 items-center justify-center rounded-lg transition-colors bg-secondary">
                <FolderOpen className="h-3.5 w-3.5 text-secondary-foreground" />
              </div>
              <div className="flex items-center gap-2 min-w-0">
                <span className="text-sm font-semibold truncate">{group.name}</span>
                <span className="text-xs text-muted-foreground font-mono">{group.services.length}</span>
              </div>
              <ChevronDown className={cn(
                "ml-auto h-4 w-4 text-muted-foreground transition-transform duration-200 shrink-0",
                !open && "-rotate-90"
              )} />
            </button>
          </CollapsibleTrigger>

          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="icon" className="h-7 w-7 rounded-lg shrink-0 hover:bg-accent">
                <MoreVertical className="h-3.5 w-3.5 text-muted-foreground" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-40 rounded-xl">
              <DropdownMenuItem onClick={() => onEditGroup(group)} className="gap-2 text-xs">
                <Pencil className="h-3.5 w-3.5" /> Rename
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem className="gap-2 text-xs text-destructive" onClick={handleDelete}>
                <Trash2 className="h-3.5 w-3.5" /> Delete
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>

        <Separator />

        {/* Services grid */}
        <CollapsibleContent>
          <div className="p-4">
            <div className="grid grid-cols-3 gap-3 sm:grid-cols-4 md:grid-cols-5 lg:grid-cols-6 xl:grid-cols-8">
              {group.services.map((s) => (
                <ServiceCard key={s.id} service={s} onEdit={onEditService} onDelete={onDeleteService} />
              ))}
            </div>
          </div>
        </CollapsibleContent>
      </Card>
    </Collapsible>
  );
}
