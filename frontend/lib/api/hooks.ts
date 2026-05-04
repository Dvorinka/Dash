"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import * as api from "./client";
import type {
  Dashboard,
  CreateGroupRequest,
  PatchGroupRequest,
  ServiceRequest,
  LayoutRequest,
  WidgetRequest,
} from "./schema";

const DASHBOARD_KEY = ["dashboard"];
const WIDGETS_KEY = ["widgets"];

export function useDashboard() {
  return useQuery({ queryKey: DASHBOARD_KEY, queryFn: api.getDashboard });
}

export function useCreateGroup() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: CreateGroupRequest) => api.createGroup(body),
    onSuccess: () => qc.invalidateQueries({ queryKey: DASHBOARD_KEY }),
  });
}

export function useUpdateGroup() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, ...body }: PatchGroupRequest & { id: string }) => api.patchGroup(id, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: DASHBOARD_KEY }),
  });
}

export function useDeleteGroup() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, moveServices }: { id: string; moveServices?: boolean }) =>
      api.deleteGroup(id, moveServices),
    onSuccess: () => qc.invalidateQueries({ queryKey: DASHBOARD_KEY }),
  });
}

export function useCreateService() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: ServiceRequest) => api.createService(body),
    onSuccess: () => qc.invalidateQueries({ queryKey: DASHBOARD_KEY }),
  });
}

export function useUpdateService() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, ...body }: ServiceRequest & { id: string }) => api.patchService(id, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: DASHBOARD_KEY }),
  });
}

export function useDeleteService() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.deleteService(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: DASHBOARD_KEY }),
  });
}

export function useUpdateLayout() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: LayoutRequest) => api.putLayout(body),
    onMutate: async () => {
      await qc.cancelQueries({ queryKey: DASHBOARD_KEY });
      const prev = qc.getQueryData<Dashboard>(DASHBOARD_KEY);
      return { prev };
    },
    onError: (_err, _vars, ctx) => {
      if (ctx?.prev) qc.setQueryData(DASHBOARD_KEY, ctx.prev);
    },
    onSettled: () => qc.invalidateQueries({ queryKey: DASHBOARD_KEY }),
  });
}

export function useUploadIcon() {
  return useMutation({ mutationFn: (file: File) => api.uploadIcon(file) });
}

export function useCreateWidget() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: WidgetRequest) => api.createWidget(body),
    onSuccess: () => qc.invalidateQueries({ queryKey: DASHBOARD_KEY }),
  });
}

export function useUpdateWidget() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, ...body }: WidgetRequest & { id: string }) => api.patchWidget(id, body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: DASHBOARD_KEY });
      qc.invalidateQueries({ queryKey: WIDGETS_KEY });
    },
  });
}

export function useDeleteWidget() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.deleteWidget(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: DASHBOARD_KEY }),
  });
}

export function useWidgetData(widgetId: string | null) {
  return useQuery({
    queryKey: ["widget-data", widgetId],
    queryFn: () => api.getWidgetData(widgetId!),
    enabled: !!widgetId,
    refetchInterval: 60_000,
  });
}

export function useRefreshWidget() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.refreshWidget(id),
    onSuccess: (_data, id) => {
      qc.invalidateQueries({ queryKey: ["widget-data", id] });
    },
  });
}
