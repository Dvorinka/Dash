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
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
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
  rectSortingStrategy,
  useSortable,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { cn } from "@/lib/utils";

/* ---------- Sortable wrapper for widgets only ---------- */

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
        <Button variant="ghost" size="icon" className="h-7 w-7 rounded-lg text-destructive hover:bg-accent" onClick={() => onDelete(service.id)}>
          <Trash2 className="h-3.5 w-3.5" />
        </Button>
      </div>
    </div>
  );
}

/* ---------- Drag Overlay (widgets only) ---------- */

function DashboardDragOverlay({ activeId, dashboard }: { activeId: string; dashboard: Dashboard }) {
  const widget = dashboard.widgets.find((w) => w.id === activeId);

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

  return null;
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

    const groupIds = dashboard.groups.map((g) => g.id);
    const widgetIds = dashboard.widgets.map((w) => w.id);

    const isActiveWidget = widgetIds.includes(activeIdStr);
    const isOverWidget = widgetIds.includes(overIdStr);

    // Widget reorder only
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
        <div className="h-14 border-b border-border" />
        <div className="flex flex-1 flex-col items-center justify-center gap-4">
          <div className="flex h-14 w-14 items-center justify-center rounded-2xl bg-muted">
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
            <div className="flex h-20 w-20 items-center justify-center rounded-[24px] bg-gradient-to-br from-secondary to-accent border border-border shadow-border-card">
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
            {/* Widgets section */}
            <Card className="mb-6">
              <CardHeader className="flex flex-row items-center justify-between pb-3">
                <div className="flex items-center gap-2">
                  <div className="h-4 w-0.5 rounded-full bg-ring" />
                  <CardTitle className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Widgets</CardTitle>
                </div>
                <Button variant="ghost" size="sm" onClick={openAddWidget} className="gap-1.5 text-xs rounded-lg hover:bg-accent">
                  <Plus className="h-3.5 w-3.5" />
                  <span className="hidden sm:inline">Add Widget</span>
                </Button>
              </CardHeader>
              <CardContent>
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
                    className="flex w-full items-center justify-center gap-2 rounded-xl border border-dashed border-border bg-secondary/50 p-6 text-sm text-muted-foreground transition-all hover:border-ring/40 hover:bg-accent hover:text-foreground"
                  >
                    <Plus className="h-4 w-4" /> Add your first widget
                  </button>
                )}
              </CardContent>
            </Card>

            {/* Apps section */}
            <Card>
              <CardHeader className="flex flex-row items-center justify-between pb-3">
                <div className="flex items-center gap-2">
                  <div className="h-4 w-0.5 rounded-full bg-ring" />
                  <CardTitle className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Apps</CardTitle>
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
                    <div className="w-px h-3.5 bg-border" />
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
              </CardHeader>
              <CardContent className="space-y-4">
                {/* Groups */}
                {groups.map((g) => (
                  <GroupSection
                    key={g.id}
                    group={g}
                    onEditService={handleEditService}
                    onDeleteService={handleDeleteService}
                    onEditGroup={handleEditGroup}
                  />
                ))}

                {/* Ungrouped services */}
                {ungrouped.length > 0 && (
                  <div>
                    {groups.length > 0 && (
                      <div className="mb-3 flex items-center gap-2">
                        <div className="h-4 w-0.5 rounded-full bg-ring" />
                        <span className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Ungrouped</span>
                        <span className="text-xs text-muted-foreground font-mono">{ungrouped.length}</span>
                      </div>
                    )}
                    {viewMode === "grid" ? (
                      <div className="grid grid-cols-3 gap-3 sm:grid-cols-4 md:grid-cols-5 lg:grid-cols-6 xl:grid-cols-8">
                        {ungrouped.map((s) => (
                          <ServiceCard key={s.id} service={s} onEdit={handleEditService} onDelete={handleDeleteService} />
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
                  </div>
                )}

                {/* Add tile when no ungrouped but groups exist */}
                {ungrouped.length === 0 && groups.length > 0 && (
                  <AddAppTile onClick={openAddService} />
                )}

                {/* No apps at all - show empty state within apps section */}
                {groups.length === 0 && ungrouped.length === 0 && (
                  <button
                    onClick={openAddService}
                    className="flex w-full items-center justify-center gap-2 rounded-xl border border-dashed border-border bg-secondary/50 p-8 text-sm text-muted-foreground transition-all hover:border-ring/40 hover:bg-accent hover:text-foreground"
                  >
                    <Plus className="h-4 w-4" /> Add your first app
                  </button>
                )}
              </CardContent>
            </Card>

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
