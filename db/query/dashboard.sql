-- name: ListGroups :many
SELECT id::text, name, sort_order, collapsed, created_at, updated_at
FROM groups
ORDER BY sort_order ASC, created_at ASC;

-- name: ListServices :many
SELECT id::text, group_id, name, icon_url, icon_asset_id, sort_order, created_at, updated_at
FROM services
ORDER BY group_id NULLS FIRST, sort_order ASC, created_at ASC;

-- name: ListServiceURLs :many
SELECT id::text, service_id::text, label, kind, url, sort_order, is_primary, created_at, updated_at
FROM service_urls
ORDER BY service_id, sort_order ASC, created_at ASC;

-- name: ListWidgets :many
SELECT id::text, type, title, enabled, sort_order, config, created_at, updated_at
FROM widget_instances
ORDER BY sort_order ASC, created_at ASC;
