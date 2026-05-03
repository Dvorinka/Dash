-- +goose Up
ALTER TABLE widget_instances DROP CONSTRAINT IF EXISTS widget_instances_type_check;
ALTER TABLE widget_instances ADD CONSTRAINT widget_instances_type_check CHECK (type IN ('clock', 'image', 'pihole', 'memos', 'immich'));

-- +goose Down
ALTER TABLE widget_instances DROP CONSTRAINT IF EXISTS widget_instances_type_check;
ALTER TABLE widget_instances ADD CONSTRAINT widget_instances_type_check CHECK (type IN ('clock', 'image', 'pihole', 'memos'));
