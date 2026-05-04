"use client";

import type { Group, Service } from "@/lib/api/schema";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { Button } from "@/components/ui/button";
import { ServiceCard } from "@/components/services/service-card";
import { ChevronDown, MoreVertical, Pencil, Trash2, GripVertical, FolderOpen } from "lucide-react";
import { useUpdateGroup, useDeleteGroup } from "@/lib/api/hooks";
import { cn } from "@/lib/utils";
import { useState } from "react";
import { useTheme } from "@/components/providers";

interface GroupSectionProps {
  group: Group;
  onEditService: (s: Service) => void;
  onDeleteService: (id: string) => void;
  onEditGroup: (g: Group) => void;
  dragHandleProps?: React.HTMLAttributes<HTMLDivElement>;
}

export function GroupSection({ group, onEditService, onDeleteService, onEditGroup, dragHandleProps }: GroupSectionProps) {
  const updateGroup = useUpdateGroup();
  const deleteGroup = useDeleteGroup();
  const [open, setOpen] = useState(!group.collapsed);
  const { theme } = useTheme();
  const isCasaOS = theme === "casaos";

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
      <div className={cn("mb-5 rounded-2xl group/group", isCasaOS && "bg-card border border-border")}>
        {/* Group header */}
        <div className="flex items-center gap-2 px-3 py-2.5">
          {dragHandleProps && (
            <div
              {...dragHandleProps}
              className="cursor-grab rounded-md p-1 opacity-0 transition-opacity hover:bg-accent group-hover/group:opacity-60"
            >
              <GripVertical className="h-4 w-4 text-muted-foreground" />
            </div>
          )}

          <CollapsibleTrigger asChild>
            <button
              className="flex flex-1 items-center gap-2.5 group/title min-w-0"
              onClick={handleToggle}
            >
              <div className={cn(
                "flex h-7 w-7 items-center justify-center rounded-lg transition-colors",
                isCasaOS ? "bg-white/10" : "bg-accent"
              )}>
                <FolderOpen className={cn("h-3.5 w-3.5", isCasaOS ? "text-blue-300" : "text-accent-foreground")} />
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

        {/* Divider */}
        <div className={cn("mx-3 h-px", isCasaOS ? "bg-white/5" : "bg-border/40")} />

        {/* Services grid */}
        <CollapsibleContent>
          <div className="p-3 pt-2">
            <div className="grid grid-cols-3 gap-3 sm:grid-cols-4 md:grid-cols-5 lg:grid-cols-6 xl:grid-cols-8">
              {group.services.map((s) => (
                <ServiceCard key={s.id} service={s} onEdit={onEditService} onDelete={onDeleteService} />
              ))}
            </div>
          </div>
        </CollapsibleContent>
      </div>
    </Collapsible>
  );
}
