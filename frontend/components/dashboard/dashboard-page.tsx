"use client";

import { useState } from "react";
import type { Service, Group, WidgetInstance, Dashboard } from "@/lib/api/schema";
import { useDashboard, useDeleteService, useDeleteWidget, useUpdateLayout } from "@/lib/api/hooks";
import { Header } from "@/components/shell/header";
import { ServiceCard } from "@/components/services/service-card";
import { ServiceForm } from "@/components/services/service-form";
import { GroupSection } from "@/components/groups/group-section";
import { GroupForm } from "@/components/groups/group-form";
import { WidgetCard } from "@/components/widgets/widget-card";
import { WidgetForm } from "@/components/widgets/widget-form";
import { Button } from "@/components/ui/button";
import { Plus, Loader2, AlertCircle, LayoutGrid, List, Pencil, Trash2, GripVertical } from "lucide-react";
import {
  DndContext,
  closestCenter,
  DragOverlay,
  DragStartEvent,
  DragEndEvent,
  DragOverEvent,
  PointerSensor,
  KeyboardSensor,
  useSensor,
  useSensors,
  MeasuringStrategy,
} from "@dnd-kit/core";
import {
  SortableContext,
  verticalListSortingStrategy,
  rectSortingStrategy,
  useSortable,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { cn } from "@/lib/utils";

/* ---------- Sortable wrappers ---------- */

function SortableGroup({
  group,
  onEditService,
  onDeleteService,
  onEditGroup,
}: {
  group: Group;
  onEditService: (s: Service) => void;
  onDeleteService: (id: string) => void;
  onEditGroup: (g: Group) => void;
}) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: group.id,
    data: { type: "group" },
  });
  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.4 : 1,
  };

  return (
    <div ref={setNodeRef} style={style} {...attributes}>
      <GroupSection
        group={group}
        onEditService={onEditService}
        onDeleteService={onDeleteService}
        onEditGroup={onEditGroup}
        dragHandleProps={listeners}
      />
    </div>
  );
}

function SortableService({
  service,
  onEdit,
  onDelete,
}: {
  service: Service;
  onEdit: (s: Service) => void;
  onDelete: (id: string) => void;
}) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: service.id,
    data: { type: "service", groupId: service.groupId },
  });
  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.4 : 1,
  };

  return (
    <div ref={setNodeRef} style={style} {...attributes}>
      <ServiceCard service={service} onEdit={onEdit} onDelete={onDelete} dragHandleProps={listeners} isDragging={isDragging} />
    </div>
  );
}

function SortableWidget({
  widget,
  onEdit,
  onDelete,
}: {
  widget: WidgetInstance;
  onEdit: (w: WidgetInstance) => void;
  onDelete: (id: string) => void;
}) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: widget.id,
    data: { type: "widget" },
  });
  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.4 : 1,
  };

  return (
    <div ref={setNodeRef} style={style} {...attributes}>
      <WidgetCard widget={widget} onEdit={onEdit} onDelete={onDelete} dragHandleProps={listeners} />
    </div>
  );
}

/* ---------- Add-app tile ---------- */

function AddAppTile({ onClick }: { onClick: () => void }) {
  return (
    <button
      onClick={onClick}
      className="service-card group flex aspect-square flex-col items-center justify-center gap-2.5 rounded-[24px] border border-dashed border-border bg-card p-4 transition-all duration-300 hover:-translate-y-1 hover:bg-accent hover:border-ring/40 hover:shadow-border-hover"
    >
      <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-secondary transition-colors group-hover:bg-accent">
        <Plus className="h-5 w-5 text-muted-foreground transition-colors group-hover:text-foreground" />
      </div>
      <span className="text-xs font-medium text-muted-foreground transition-colors group-hover:text-foreground">Add App</span>
    </button>
  );
}


/* ---------- Service List Item ---------- */

function ServiceListItem({
  service,
  onEdit,
  onDelete,
}: {
  service: Service;
  onEdit: (s: Service) => void;
  onDelete: (id: string) => void;
}) {
  const primaryUrl = service.urls.find((u) => u.isPrimary) || service.urls[0];
  return (
    <div className="group flex items-center gap-3 rounded-xl border border-border bg-card px-4 py-3 transition-all hover:bg-accent hover:border-border hover:shadow-border">
      <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-secondary font-mono text-sm font-semibold text-secondary-foreground">
        {service.name.slice(0, 2).toUpperCase()}
      </div>
      <div className="min-w-0 flex-1">
        <div className="text-sm font-semibold truncate">{service.name}</div>
        {primaryUrl && (
          <a
            href={primaryUrl.url}
            target="_blank"
            rel="noopener noreferrer"
            className="text-xs text-muted-foreground hover:text-foreground truncate block transition-colors"
          >
            {primaryUrl.url}
          </a>
        )}
      </div>
      <div className="flex items-center gap-1 opacity-0 transition-opacity group-hover:opacity-100">
        <Button variant="ghost" size="icon" className="h-7 w-7 rounded-lg hover:bg-accent" onClick={() => onEdit(service)}>
          <Pencil className="h-3.5 w-3.5" />
        </Button>
        <Button variant="ghost" size="icon" className="h-7 w-7 rounded-lg text-destructive hover:bg-destructive/10" onClick={() => onDelete(service.id)}>
          <Trash2 className="h-3.5 w-3.5" />
        </Button>
      </div>
    </div>
  );
}

/* ---------- Drag Overlay ---------- */

function DashboardDragOverlay({ activeId, dashboard }: { activeId: string; dashboard: Dashboard }) {
  const allServices = [
    ...dashboard.ungroupedServices,
    ...dashboard.groups.flatMap((g) => g.services),
  ];
  const service = allServices.find((s) => s.id === activeId);
  const group = dashboard.groups.find((g) => g.id === activeId);
  const widget = dashboard.widgets.find((w) => w.id === activeId);

  if (service) {
    return (
      <div className="drag-overlay flex aspect-square w-28 flex-col items-center justify-center gap-2 rounded-2xl bg-card border border-ring/50 p-3 shadow-2xl">
        <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-gradient-to-br from-secondary to-accent font-mono text-sm font-bold text-secondary-foreground">
          {service.name.slice(0, 2).toUpperCase()}
        </div>
        <span className="text-xs font-semibold text-center truncate w-full">{service.name}</span>
      </div>
    );
  }

  if (group) {
    return (
      <div className="drag-overlay flex w-64 items-center gap-3 rounded-xl bg-card border border-ring/50 px-4 py-3 shadow-2xl">
        <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-accent">
          <GripVertical className="h-4 w-4 text-accent-foreground" />
        </div>
        <div>
          <span className="text-sm font-semibold">{group.name}</span>
          <span className="text-xs text-muted-foreground ml-2">{group.services.length} apps</span>
        </div>
      </div>
    );
  }

  if (widget) {
    return (
      <div className="drag-overlay flex w-56 items-center gap-3 rounded-xl bg-card border border-ring/50 px-4 py-3 shadow-2xl">
        <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-accent">
          <GripVertical className="h-4 w-4 text-accent-foreground" />
        </div>
        <div>
          <span className="text-sm font-semibold">{widget.title}</span>
          <span className="text-xs text-muted-foreground ml-2 uppercase">{widget.type}</span>
        </div>
      </div>
    );
  }

  return <div className="drag-overlay rounded-xl bg-card p-4 shadow-2xl border border-ring/50">Moving…</div>;
}

/* ---------- Main Dashboard ---------- */

export default function DashboardPage() {
  const { data: dashboard, isLoading, error } = useDashboard();
  const deleteService = useDeleteService();
  const deleteWidget = useDeleteWidget();
  const updateLayout = useUpdateLayout();

  const [serviceFormOpen, setServiceFormOpen] = useState(false);
  const [editingService, setEditingService] = useState<Service | null>(null);
  const [groupFormOpen, setGroupFormOpen] = useState(false);
  const [editingGroup, setEditingGroup] = useState<Group | null>(null);
  const [widgetFormOpen, setWidgetFormOpen] = useState(false);
  const [editingWidget, setEditingWidget] = useState<WidgetInstance | null>(null);
  const [activeId, setActiveId] = useState<string | null>(null);
  const [viewMode, setViewMode] = useState<"grid" | "list">("grid");

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 8 } }),
    useSensor(KeyboardSensor),
  );

  const handleDragStart = (event: DragStartEvent) => {
    setActiveId(String(event.active.id));
  };

  const handleDragOver = (_event: DragOverEvent) => {
    void _event;
    // Visual feedback placeholder
  };

  const handleDragEnd = (event: DragEndEvent) => {
    setActiveId(null);
    const { active, over } = event;
    if (!over || active.id === over.id || !dashboard) return;

    const activeIdStr = String(active.id);
    const overIdStr = String(over.id);

    const allServiceIds = [
      ...dashboard.ungroupedServices.map((s) => s.id),
      ...dashboard.groups.flatMap((g) => g.services.map((s) => s.id)),
    ];
    const groupIds = dashboard.groups.map((g) => g.id);
    const widgetIds = dashboard.widgets.map((w) => w.id);

    const isActiveService = allServiceIds.includes(activeIdStr);
    const isOverService = allServiceIds.includes(overIdStr);
    const isActiveGroup = groupIds.includes(activeIdStr);
    const isOverGroup = groupIds.includes(overIdStr);
    const isActiveWidget = widgetIds.includes(activeIdStr);
    const isOverWidget = widgetIds.includes(overIdStr);

    // Service → Service (reorder / cross-group)
    if (isActiveService && isOverService) {
      const findServiceLocation = (sid: string): { groupId: string | null; index: number } => {
        const ungroupedIdx = dashboard.ungroupedServices.findIndex((s) => s.id === sid);
        if (ungroupedIdx !== -1) return { groupId: null, index: ungroupedIdx };
        for (const g of dashboard.groups) {
          const idx = g.services.findIndex((s) => s.id === sid);
          if (idx !== -1) return { groupId: g.id, index: idx };
        }
        return { groupId: null, index: -1 };
      };

      const activeLoc = findServiceLocation(activeIdStr);
      const overLoc = findServiceLocation(overIdStr);

      const groupServices: Record<string, string[]> = {};
      for (const g of dashboard.groups) {
        const ids = [...g.services.map((s) => s.id)];
        if (activeLoc.groupId === g.id) ids.splice(activeLoc.index, 1);
        if (overLoc.groupId === g.id) {
          const insertIdx = activeLoc.groupId === g.id && activeLoc.index < overLoc.index ? overLoc.index : overLoc.index;
          ids.splice(insertIdx, 0, activeIdStr);
        }
        groupServices[g.id] = ids;
      }

      const ungroupedIds = [...dashboard.ungroupedServices.map((s) => s.id)];
      if (activeLoc.groupId === null) ungroupedIds.splice(activeLoc.index, 1);
      if (overLoc.groupId === null) {
        const insertIdx = activeLoc.groupId === null && activeLoc.index < overLoc.index ? overLoc.index : overLoc.index;
        ungroupedIds.splice(insertIdx, 0, activeIdStr);
      }
      if (activeLoc.groupId !== null && overLoc.groupId === null) {
        ungroupedIds.splice(overLoc.index, 0, activeIdStr);
      }

      updateLayout.mutate({ groupIds, widgetIds, ungroupedServiceIds: ungroupedIds, groupServices });
      return;
    }

    // Service → Group header (move into group)
    if (isActiveService && isOverGroup) {
      const groupServices: Record<string, string[]> = {};
      const ungroupedIds = [...dashboard.ungroupedServices.map((s) => s.id)];

      for (const g of dashboard.groups) {
        const ids = g.services.map((s) => s.id);
        const idx = ids.indexOf(activeIdStr);
        if (idx !== -1) ids.splice(idx, 1);
        if (g.id === overIdStr) ids.push(activeIdStr);
        groupServices[g.id] = ids;
      }
      const uIdx = ungroupedIds.indexOf(activeIdStr);
      if (uIdx !== -1) ungroupedIds.splice(uIdx, 1);

      updateLayout.mutate({ groupIds, widgetIds, ungroupedServiceIds: ungroupedIds, groupServices });
      return;
    }

    // Group reorder
    if (isActiveGroup && isOverGroup) {
      const newGroupIds = [...groupIds];
      const fromIdx = newGroupIds.indexOf(activeIdStr);
      const toIdx = newGroupIds.indexOf(overIdStr);
      if (fromIdx !== -1 && toIdx !== -1) {
        const [moved] = newGroupIds.splice(fromIdx, 1);
        newGroupIds.splice(toIdx, 0, moved);
        const groupServices: Record<string, string[]> = {};
        for (const g of dashboard.groups) groupServices[g.id] = g.services.map((s) => s.id);
        updateLayout.mutate({ groupIds: newGroupIds, widgetIds, ungroupedServiceIds: dashboard.ungroupedServices.map((s) => s.id), groupServices });
      }
      return;
    }

    // Widget reorder
    if (isActiveWidget && isOverWidget) {
      const newWidgetIds = [...widgetIds];
      const fromIdx = newWidgetIds.indexOf(activeIdStr);
      const toIdx = newWidgetIds.indexOf(overIdStr);
      if (fromIdx !== -1 && toIdx !== -1) {
        const [moved] = newWidgetIds.splice(fromIdx, 1);
        newWidgetIds.splice(toIdx, 0, moved);
        const groupServices: Record<string, string[]> = {};
        for (const g of dashboard.groups) groupServices[g.id] = g.services.map((s) => s.id);
        updateLayout.mutate({ groupIds, widgetIds: newWidgetIds, ungroupedServiceIds: dashboard.ungroupedServices.map((s) => s.id), groupServices });
      }
    }
  };

  const handleEditService = (s: Service) => { setEditingService(s); setServiceFormOpen(true); };
  const handleDeleteService = (id: string) => { if (confirm("Delete this app?")) deleteService.mutate(id); };
  const handleEditGroup = (g: Group) => { setEditingGroup(g); setGroupFormOpen(true); };
  const handleEditWidget = (w: WidgetInstance) => { setEditingWidget(w); setWidgetFormOpen(true); };
  const handleDeleteWidget = (id: string) => { if (confirm("Delete this widget?")) deleteWidget.mutate(id); };

  const openAddService = () => { setEditingService(null); setServiceFormOpen(true); };
  const openAddGroup = () => { setEditingGroup(null); setGroupFormOpen(true); };
  const openAddWidget = () => { setEditingWidget(null); setWidgetFormOpen(true); };

  if (isLoading) {
    return (
      <div className="flex h-screen flex-col bg-background">
        <div className="h-14 border-b border-border/50" />
        <div className="flex flex-1 items-center justify-center">
          <div className="flex flex-col items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-accent">
              <Loader2 className="h-5 w-5 animate-spin text-accent-foreground" />
            </div>
            <span className="text-xs text-muted-foreground font-medium">Loading dashboard...</span>
          </div>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex h-screen flex-col bg-background">
        <div className="h-14 border-b border-border/50" />
        <div className="flex flex-1 flex-col items-center justify-center gap-4">
          <div className="flex h-14 w-14 items-center justify-center rounded-2xl bg-destructive/10">
            <AlertCircle className="h-6 w-6 text-destructive" />
          </div>
          <div className="text-center">
            <p className="text-sm font-semibold text-foreground">Failed to load dashboard</p>
            <p className="text-xs text-muted-foreground mt-1">{error.message}</p>
          </div>
          <Button variant="outline" size="sm" onClick={() => window.location.reload()}>
            Retry
          </Button>
        </div>
      </div>
    );
  }

  const groups = dashboard?.groups || [];
  const ungrouped = dashboard?.ungroupedServices || [];
  const widgets = dashboard?.widgets || [];
  const isEmpty = groups.length === 0 && ungrouped.length === 0 && widgets.length === 0;

  return (
    <div className="flex min-h-screen flex-col">
      <Header onAddService={openAddService} onAddWidget={openAddWidget} onAddGroup={openAddGroup} />

      <main className="mx-auto w-full max-w-6xl flex-1 px-4 py-6">
        {isEmpty ? (
          <div className="flex flex-col items-center justify-center gap-6 py-32">
            <div className="flex h-20 w-20 items-center justify-center rounded-[24px] bg-gradient-to-br from-secondary to-accent border border-border/50 shadow-border-card">
              <LayoutGrid className="h-8 w-8 text-muted-foreground" />
            </div>
            <div className="text-center">
              <h2 className="text-xl font-semibold text-foreground tracking-tight mb-2">Welcome to Dash</h2>
              <p className="text-sm text-muted-foreground max-w-xs">Your homelab dashboard is empty. Add apps and widgets to get started.</p>
            </div>
            <div className="flex gap-3">
              <Button onClick={openAddService} className="gap-2 rounded-xl">
                <Plus className="h-4 w-4" /> Add App
              </Button>
              <Button onClick={openAddWidget} variant="outline" className="gap-2 rounded-xl">
                <Plus className="h-4 w-4" /> Add Widget
              </Button>
            </div>
          </div>
        ) : (
          <DndContext
            sensors={sensors}
            collisionDetection={closestCenter}
            measuring={{ droppable: { strategy: MeasuringStrategy.Always } }}
            onDragStart={handleDragStart}
            onDragOver={handleDragOver}
            onDragEnd={handleDragEnd}
          >
            {/* Widgets strip */}
            <section className="mb-8">
              <div className="mb-4 flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <div className="h-4 w-0.5 rounded-full bg-ring" />
                  <span className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Widgets</span>
                </div>
                <Button variant="ghost" size="sm" onClick={openAddWidget} className="gap-1.5 text-xs rounded-lg hover:bg-accent">
                  <Plus className="h-3.5 w-3.5" />
                  <span className="hidden sm:inline">Add Widget</span>
                </Button>
              </div>
              {widgets.length > 0 ? (
                <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
                  <SortableContext items={widgets.map((w) => w.id)} strategy={rectSortingStrategy}>
                    {widgets.map((w) => (
                      <SortableWidget key={w.id} widget={w} onEdit={handleEditWidget} onDelete={handleDeleteWidget} />
                    ))}
                  </SortableContext>
                </div>
              ) : (
                <button
                  onClick={openAddWidget}
                  className="flex w-full items-center justify-center gap-2 rounded-2xl border border-dashed border-border bg-card p-6 text-sm text-muted-foreground transition-all hover:border-ring/40 hover:bg-accent hover:text-foreground"
                >
                  <Plus className="h-4 w-4" /> Add your first widget
                </button>
              )}
            </section>

            {/* Apps section */}
            <section className="mb-4">
              <div className="mb-4 flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <div className="h-4 w-0.5 rounded-full bg-ring" />
                  <span className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Apps</span>
                </div>
                <div className="flex items-center gap-1">
                  <div className="flex items-center rounded-lg border border-border overflow-hidden mr-1 bg-card">
                    <button
                      onClick={() => setViewMode("grid")}
                      className={cn("px-2.5 py-1.5 text-xs transition-colors rounded-l-lg", viewMode === "grid" ? "bg-accent text-accent-foreground" : "text-muted-foreground hover:text-foreground")}
                      title="Grid view"
                    >
                      <LayoutGrid className="h-3.5 w-3.5" />
                    </button>
                    <div className="w-px h-3.5 bg-border/50" />
                    <button
                      onClick={() => setViewMode("list")}
                      className={cn("px-2.5 py-1.5 text-xs transition-colors rounded-r-lg", viewMode === "list" ? "bg-accent text-accent-foreground" : "text-muted-foreground hover:text-foreground")}
                      title="List view"
                    >
                      <List className="h-3.5 w-3.5" />
                    </button>
                  </div>
                  <Button variant="ghost" size="sm" onClick={openAddGroup} className="gap-1.5 text-xs rounded-lg hover:bg-accent">
                    <Plus className="h-3.5 w-3.5" />
                    <span className="hidden sm:inline">Group</span>
                  </Button>
                  <Button variant="ghost" size="sm" onClick={openAddService} className="gap-1.5 text-xs rounded-lg hover:bg-accent">
                    <Plus className="h-3.5 w-3.5" />
                    <span className="hidden sm:inline">App</span>
                  </Button>
                </div>
              </div>

              {/* Groups */}
              <SortableContext items={groups.map((g) => g.id)} strategy={verticalListSortingStrategy}>
                {groups.map((g) => (
                  <SortableGroup
                    key={g.id}
                    group={g}
                    onEditService={handleEditService}
                    onDeleteService={handleDeleteService}
                    onEditGroup={handleEditGroup}
                  />
                ))}
              </SortableContext>

              {/* Ungrouped services */}
              {ungrouped.length > 0 && (
                <div className="mb-2">
                  {groups.length > 0 && (
                    <div className="mb-4 flex items-center gap-2">
                      <div className="h-4 w-0.5 rounded-full bg-ring" />
                      <span className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Ungrouped</span>
                      <span className="text-xs text-muted-foreground font-mono">{ungrouped.length}</span>
                    </div>
                  )}
                  <SortableContext items={ungrouped.map((s) => s.id)} strategy={rectSortingStrategy}>
                    {viewMode === "grid" ? (
                      <div className="grid grid-cols-3 gap-3 sm:grid-cols-4 md:grid-cols-5 lg:grid-cols-6 xl:grid-cols-8">
                        {ungrouped.map((s) => (
                          <SortableService key={s.id} service={s} onEdit={handleEditService} onDelete={handleDeleteService} />
                        ))}
                        <AddAppTile onClick={openAddService} />
                      </div>
                    ) : (
                      <div className="flex flex-col gap-2">
                        {ungrouped.map((s) => (
                          <ServiceListItem key={s.id} service={s} onEdit={handleEditService} onDelete={handleDeleteService} />
                        ))}
                      </div>
                    )}
                  </SortableContext>
                </div>
              )}

              {/* In-grid add tile when no ungrouped but groups exist */}
              {ungrouped.length === 0 && groups.length > 0 && (
                <div className="mt-2">
                  <AddAppTile onClick={openAddService} />
                </div>
              )}

              {/* No apps at all - show empty state within apps section */}
              {groups.length === 0 && ungrouped.length === 0 && (
                <button
                  onClick={openAddService}
                  className="flex w-full items-center justify-center gap-2 rounded-2xl border border-dashed border-border bg-card p-8 text-sm text-muted-foreground transition-all hover:border-ring/40 hover:bg-accent hover:text-foreground"
                >
                  <Plus className="h-4 w-4" /> Add your first app
                </button>
              )}
            </section>

            <DragOverlay>
              {activeId && dashboard ? (
                <DashboardDragOverlay activeId={activeId} dashboard={dashboard} />
              ) : null}
            </DragOverlay>
          </DndContext>
        )}
      </main>

      {/* Modals */}
      <ServiceForm
        service={editingService}
        groups={groups.map((g) => ({ id: g.id, name: g.name }))}
        open={serviceFormOpen}
        onOpenChange={setServiceFormOpen}
      />
      <GroupForm group={editingGroup} open={groupFormOpen} onOpenChange={setGroupFormOpen} />
      <WidgetForm widget={editingWidget} open={widgetFormOpen} onOpenChange={setWidgetFormOpen} />
    </div>
  );
}
