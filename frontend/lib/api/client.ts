import type {
  Dashboard,
  Group,
  Service,
  WidgetInstance,
  WidgetData,
  AssetFile,
  CreateGroupRequest,
  PatchGroupRequest,
  ServiceRequest,
  LayoutRequest,
  WidgetRequest,
} from "./schema";

const API_BASE = process.env.NEXT_PUBLIC_API_BASE_URL || "http://localhost:8080";

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    headers: { "Content-Type": "application/json", ...init?.headers },
    ...init,
  });
  if (res.status === 204) return undefined as T;
  if (!res.ok) {
    const err = await res.json().catch(() => ({ message: res.statusText }));
    throw new ApiError(res.status, err.code || "unknown", err.message || "Request failed", err.details);
  }
  return res.json();
}

export class ApiError extends Error {
  constructor(
    public status: number,
    public code: string,
    message: string,
    public details?: unknown,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

// ── Dashboard ──
export function getDashboard(): Promise<Dashboard> {
  return request("/api/v1/dashboard");
}

// ── Groups ──
export function listGroups(): Promise<Group[]> {
  return request("/api/v1/groups");
}
export function createGroup(body: CreateGroupRequest): Promise<Group> {
  return request("/api/v1/groups", { method: "POST", body: JSON.stringify(body) });
}
export function patchGroup(id: string, body: PatchGroupRequest): Promise<Group> {
  return request(`/api/v1/groups/${id}`, { method: "PATCH", body: JSON.stringify(body) });
}
export function deleteGroup(id: string, moveServices = false): Promise<void> {
  return request(`/api/v1/groups/${id}?moveServicesToUngrouped=${moveServices}`, { method: "DELETE" });
}

// ── Services ──
export function listServices(): Promise<Service[]> {
  return request("/api/v1/services");
}
export function createService(body: ServiceRequest): Promise<Service> {
  return request("/api/v1/services", { method: "POST", body: JSON.stringify(body) });
}
export function patchService(id: string, body: ServiceRequest): Promise<Service> {
  return request(`/api/v1/services/${id}`, { method: "PATCH", body: JSON.stringify(body) });
}
export function deleteService(id: string): Promise<void> {
  return request(`/api/v1/services/${id}`, { method: "DELETE" });
}

// ── Layout ──
export function putLayout(body: LayoutRequest): Promise<Dashboard> {
  return request("/api/v1/layout", { method: "PUT", body: JSON.stringify(body) });
}

// ── Assets ──
export async function uploadIcon(file: File): Promise<AssetFile> {
  const form = new FormData();
  form.append("file", file);
  const res = await fetch(`${API_BASE}/api/v1/assets/icons`, { method: "POST", body: form });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ message: res.statusText }));
    throw new ApiError(res.status, err.code || "unknown", err.message || "Upload failed");
  }
  return res.json();
}

// ── Widgets ──
export function listWidgets(): Promise<WidgetInstance[]> {
  return request("/api/v1/widgets");
}
export function createWidget(body: WidgetRequest): Promise<WidgetInstance> {
  return request("/api/v1/widgets", { method: "POST", body: JSON.stringify(body) });
}
export function patchWidget(id: string, body: WidgetRequest): Promise<WidgetInstance> {
  return request(`/api/v1/widgets/${id}`, { method: "PATCH", body: JSON.stringify(body) });
}
export function deleteWidget(id: string): Promise<void> {
  return request(`/api/v1/widgets/${id}`, { method: "DELETE" });
}
export function getWidgetData(id: string): Promise<WidgetData> {
  return request(`/api/v1/widgets/${id}/data`);
}
export function refreshWidget(id: string): Promise<WidgetData> {
  return request(`/api/v1/widgets/${id}/refresh`, { method: "POST" });
}
