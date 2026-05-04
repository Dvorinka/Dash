import { http, HttpResponse } from "msw";
import { FIXTURE_DASHBOARD, FIXTURE_WIDGET_DATA } from "./fixtures";

const BASE = process.env.NEXT_PUBLIC_API_BASE_URL || "http://localhost:8080";

export const handlers = [
  http.get(`${BASE}/api/v1/dashboard`, () => {
    return HttpResponse.json(FIXTURE_DASHBOARD);
  }),

  http.get(`${BASE}/api/v1/groups`, () => {
    return HttpResponse.json(FIXTURE_DASHBOARD.groups);
  }),

  http.post(`${BASE}/api/v1/groups`, async ({ request }) => {
    const body = await request.json();
    const newGroup = {
      id: crypto.randomUUID(),
      name: (body as { name: string }).name,
      sortOrder: FIXTURE_DASHBOARD.groups.length,
      collapsed: false,
      services: [],
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    };
    return HttpResponse.json(newGroup, { status: 201 });
  }),

  http.get(`${BASE}/api/v1/services`, () => {
    const all = [...FIXTURE_DASHBOARD.ungroupedServices, ...FIXTURE_DASHBOARD.groups.flatMap((g) => g.services)];
    return HttpResponse.json(all);
  }),

  http.post(`${BASE}/api/v1/services`, async ({ request }) => {
    const body = await request.json();
    const newService = {
      id: crypto.randomUUID(),
      ...(body as Record<string, unknown>),
      sortOrder: 0,
      urls: (body as { urls: Record<string, unknown>[] }).urls.map((u, i) => ({ ...u, id: crypto.randomUUID(), sortOrder: i })),
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    };
    return HttpResponse.json(newService, { status: 201 });
  }),

  http.put(`${BASE}/api/v1/layout`, async ({ request }) => {
    const body = await request.json();
    return HttpResponse.json({ ...FIXTURE_DASHBOARD, ...(body as Record<string, unknown>) });
  }),

  http.post(`${BASE}/api/v1/assets/icons`, async () => {
    return HttpResponse.json(
      {
        id: crypto.randomUUID(),
        originalName: "icon.png",
        storedName: "icon-mock.png",
        mimeType: "image/png",
        sizeBytes: 1024,
        publicPath: "/uploads/icons/icon-mock.png",
        createdAt: new Date().toISOString(),
      },
      { status: 201 },
    );
  }),

  http.get(`${BASE}/api/v1/widgets`, () => {
    return HttpResponse.json(FIXTURE_DASHBOARD.widgets);
  }),

  http.post(`${BASE}/api/v1/widgets`, async ({ request }) => {
    const body = await request.json();
    const newWidget = {
      id: crypto.randomUUID(),
      ...(body as Record<string, unknown>),
      sortOrder: FIXTURE_DASHBOARD.widgets.length,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    };
    return HttpResponse.json(newWidget, { status: 201 });
  }),

  Object.entries(FIXTURE_WIDGET_DATA).map(([widgetId, data]) =>
    http.get(`${BASE}/api/v1/widgets/${widgetId}/data`, () => {
      return HttpResponse.json(data);
    })
  ),

  http.post(`${BASE}/api/v1/widgets/:widgetId/refresh`, ({ params }) => {
    const data = FIXTURE_WIDGET_DATA[params.widgetId as string];
    return HttpResponse.json(data || { widgetId: params.widgetId, status: "fresh", data: {}, fetchedAt: new Date().toISOString() });
  }),
].flat();
