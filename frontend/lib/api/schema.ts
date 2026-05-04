// Auto-generated from ../openapi/openapi.yaml
// Run: npm run api:generate

export interface Dashboard {
  groups: Group[];
  ungroupedServices: Service[];
  widgets: WidgetInstance[];
}

export interface Group {
  id: string;
  name: string;
  sortOrder: number;
  collapsed: boolean;
  services: Service[];
  createdAt: string;
  updatedAt: string;
}

export interface Service {
  id: string;
  groupId: string | null;
  name: string;
  iconUrl: string | null;
  iconAssetId: string | null;
  sortOrder: number;
  urls: ServiceUrl[];
  createdAt: string;
  updatedAt: string;
}

export interface ServiceUrl {
  id: string;
  label: string;
  kind: "local" | "external" | "custom";
  url: string;
  sortOrder: number;
  isPrimary: boolean;
}

export interface WidgetInstance {
  id: string;
  type: "clock" | "image" | "pihole" | "memos" | "immich";
  title: string;
  enabled: boolean;
  sortOrder: number;
  config: Record<string, unknown>;
  createdAt: string;
  updatedAt: string;
}

export interface WidgetData {
  widgetId: string;
  status: "fresh" | "stale" | "error";
  data?: Record<string, unknown>;
  error?: string | null;
  fetchedAt?: string | null;
  expiresAt?: string | null;
}

export interface AssetFile {
  id: string;
  originalName: string;
  storedName: string;
  mimeType: string;
  sizeBytes: number;
  publicPath: string;
  createdAt: string;
}

export interface ErrorResponse {
  code:
    | "validation_error"
    | "not_found"
    | "conflict"
    | "upload_too_large"
    | "unsupported_media_type"
    | "widget_fetch_failed"
    | "internal_error";
  message: string;
  details: Record<string, unknown> | null;
}

export interface CreateGroupRequest {
  name: string;
}

export interface PatchGroupRequest {
  name?: string;
  collapsed?: boolean;
}

export interface ServiceRequest {
  groupId?: string | null;
  name: string;
  iconUrl?: string | null;
  iconAssetId?: string | null;
  urls: ServiceUrlInput[];
}

export interface ServiceUrlInput {
  id?: string;
  label: string;
  kind: "local" | "external" | "custom";
  url: string;
  isPrimary?: boolean;
}

export interface LayoutRequest {
  groupIds: string[];
  widgetIds: string[];
  ungroupedServiceIds: string[];
  groupServices: Record<string, string[]>;
}

export interface WidgetRequest {
  type: "clock" | "image" | "pihole" | "memos" | "immich";
  title: string;
  enabled?: boolean;
  config: ClockWidgetConfig | ImageWidgetConfig | PiHoleWidgetConfig | MemosWidgetConfig | ImmichWidgetConfig;
}

export interface ClockWidgetConfig {
  timezones?: string[];
}

export interface ImageWidgetConfig {
  imageUrl: string;
  linkUrl?: string | null;
}

export interface PiHoleWidgetConfig {
  baseUrl: string;
  apiToken: string;
}

export interface MemosWidgetConfig {
  baseUrl: string;
  apiToken: string;
  pageSize?: number;
}

export interface ImmichWidgetConfig {
  baseUrl: string;
  apiKey: string;
}
