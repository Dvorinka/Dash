-- +goose Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE groups (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name text NOT NULL,
  sort_order integer NOT NULL,
  collapsed boolean NOT NULL DEFAULT false,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE asset_files (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  original_name text NOT NULL,
  stored_name text NOT NULL,
  mime_type text NOT NULL,
  size_bytes integer NOT NULL,
  public_path text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE services (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  group_id uuid NULL REFERENCES groups(id) ON DELETE SET NULL,
  name text NOT NULL,
  icon_url text NULL,
  icon_asset_id uuid NULL REFERENCES asset_files(id) ON DELETE SET NULL,
  sort_order integer NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE service_urls (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  service_id uuid NOT NULL REFERENCES services(id) ON DELETE CASCADE,
  label text NOT NULL,
  kind text NOT NULL CHECK (kind IN ('local', 'external', 'custom')),
  url text NOT NULL,
  sort_order integer NOT NULL,
  is_primary boolean NOT NULL DEFAULT false,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE widget_instances (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  type text NOT NULL CHECK (type IN ('clock', 'image', 'pihole')),
  title text NOT NULL,
  enabled boolean NOT NULL DEFAULT true,
  sort_order integer NOT NULL,
  config jsonb NOT NULL DEFAULT '{}',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE widget_cache (
  widget_id uuid PRIMARY KEY REFERENCES widget_instances(id) ON DELETE CASCADE,
  status text NOT NULL CHECK (status IN ('fresh', 'stale', 'error')),
  data jsonb NULL,
  error text NULL,
  fetched_at timestamptz NULL,
  expires_at timestamptz NULL,
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX groups_sort_order_idx ON groups(sort_order);
CREATE INDEX services_group_sort_order_idx ON services(group_id, sort_order);
CREATE INDEX services_ungrouped_sort_order_idx ON services(sort_order) WHERE group_id IS NULL;
CREATE INDEX service_urls_service_sort_order_idx ON service_urls(service_id, sort_order);
CREATE INDEX widget_instances_enabled_sort_order_idx ON widget_instances(enabled, sort_order);
CREATE INDEX asset_files_created_at_idx ON asset_files(created_at);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS trigger AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER groups_set_updated_at BEFORE UPDATE ON groups
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER services_set_updated_at BEFORE UPDATE ON services
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER service_urls_set_updated_at BEFORE UPDATE ON service_urls
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER widget_instances_set_updated_at BEFORE UPDATE ON widget_instances
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER widget_cache_set_updated_at BEFORE UPDATE ON widget_cache
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TRIGGER IF EXISTS widget_cache_set_updated_at ON widget_cache;
DROP TRIGGER IF EXISTS widget_instances_set_updated_at ON widget_instances;
DROP TRIGGER IF EXISTS service_urls_set_updated_at ON service_urls;
DROP TRIGGER IF EXISTS services_set_updated_at ON services;
DROP TRIGGER IF EXISTS groups_set_updated_at ON groups;
DROP FUNCTION IF EXISTS set_updated_at();
DROP TABLE IF EXISTS widget_cache;
DROP TABLE IF EXISTS widget_instances;
DROP TABLE IF EXISTS service_urls;
DROP TABLE IF EXISTS services;
DROP TABLE IF EXISTS asset_files;
DROP TABLE IF EXISTS groups;
