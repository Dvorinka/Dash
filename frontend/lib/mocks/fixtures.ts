import type { Dashboard, Group, Service, WidgetInstance, WidgetData, AssetFile } from "@/lib/api/schema";

export const FIXTURE_ASSET: AssetFile = {
  id: "a1b2c3d4-0000-0000-0000-000000000001",
  originalName: "jellyfin.png",
  storedName: "jellyfin-abc123.png",
  mimeType: "image/png",
  sizeBytes: 12345,
  publicPath: "/uploads/icons/jellyfin-abc123.png",
  createdAt: "2025-01-01T00:00:00Z",
};

export const FIXTURE_SERVICES: Service[] = [
  {
    id: "s1-0000-0000-0000-000000000001",
    groupId: "g1-0000-0000-0000-000000000001",
    name: "Jellyfin",
    iconUrl: null,
    iconAssetId: FIXTURE_ASSET.id,
    sortOrder: 0,
    urls: [
      { id: "u1", label: "Local", kind: "local", url: "http://jellyfin.local:8096", sortOrder: 0, isPrimary: true },
      { id: "u2", label: "External", kind: "external", url: "https://jellyfin.example.com", sortOrder: 1, isPrimary: false },
    ],
    createdAt: "2025-01-01T00:00:00Z",
    updatedAt: "2025-01-01T00:00:00Z",
  },
  {
    id: "s2-0000-0000-0000-000000000002",
    groupId: "g1-0000-0000-0000-000000000001",
    name: "Pi-hole",
    iconUrl: null,
    iconAssetId: null,
    sortOrder: 1,
    urls: [
      { id: "u3", label: "Dashboard", kind: "local", url: "http://pihole.local/admin", sortOrder: 0, isPrimary: true },
    ],
    createdAt: "2025-01-01T00:00:00Z",
    updatedAt: "2025-01-01T00:00:00Z",
  },
  {
    id: "s3-0000-0000-0000-000000000003",
    groupId: null,
    name: "Proxmox",
    iconUrl: "https://proxmox.com/favicon.ico",
    iconAssetId: null,
    sortOrder: 0,
    urls: [
      { id: "u4", label: "Web UI", kind: "local", url: "https://proxmox.local:8006", sortOrder: 0, isPrimary: true },
    ],
    createdAt: "2025-01-01T00:00:00Z",
    updatedAt: "2025-01-01T00:00:00Z",
  },
];

export const FIXTURE_GROUPS: Group[] = [
  {
    id: "g1-0000-0000-0000-000000000001",
    name: "Media",
    sortOrder: 0,
    collapsed: false,
    services: FIXTURE_SERVICES.filter((s) => s.groupId === "g1-0000-0000-0000-000000000001"),
    createdAt: "2025-01-01T00:00:00Z",
    updatedAt: "2025-01-01T00:00:00Z",
  },
];

export const FIXTURE_WIDGETS: WidgetInstance[] = [
  {
    id: "w1-0000-0000-0000-000000000001",
    type: "clock",
    title: "Clock",
    enabled: true,
    sortOrder: 0,
    config: { timezones: ["Europe/Prague", "America/New_York"] },
    createdAt: "2025-01-01T00:00:00Z",
    updatedAt: "2025-01-01T00:00:00Z",
  },
  {
    id: "w2-0000-0000-0000-000000000002",
    type: "pihole",
    title: "Pi-hole Stats",
    enabled: true,
    sortOrder: 1,
    config: { baseUrl: "http://pihole.local", apiToken: "••••••••" },
    createdAt: "2025-01-01T00:00:00Z",
    updatedAt: "2025-01-01T00:00:00Z",
  },
];

export const FIXTURE_WIDGET_DATA: Record<string, WidgetData> = {
  "w1-0000-0000-0000-000000000001": {
    widgetId: "w1-0000-0000-0000-000000000001",
    status: "fresh",
    data: {},
    fetchedAt: "2025-01-01T12:00:00Z",
    expiresAt: "2025-01-01T12:01:00Z",
  },
  "w2-0000-0000-0000-000000000002": {
    widgetId: "w2-0000-0000-0000-000000000002",
    status: "fresh",
    data: {
      status: "enabled",
      ads_blocked_today: 45231,
      dns_queries_today: 120000,
      ads_percentage_today: 37.69,
    },
    fetchedAt: "2025-01-01T12:00:00Z",
    expiresAt: "2025-01-01T12:01:00Z",
  },
};

export const FIXTURE_DASHBOARD: Dashboard = {
  groups: FIXTURE_GROUPS,
  ungroupedServices: FIXTURE_SERVICES.filter((s) => s.groupId === null),
  widgets: FIXTURE_WIDGETS,
};
