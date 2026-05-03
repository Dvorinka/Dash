package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"dash/backend/internal/store/dbgen"
)

type Store struct {
	pool    *pgxpool.Pool
	queries *dbgen.Queries
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool, queries: dbgen.New(pool)}
}

func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

func (s *Store) Dashboard(ctx context.Context) (Dashboard, error) {
	groups, err := s.Groups(ctx)
	if err != nil {
		return Dashboard{}, err
	}
	services, err := s.Services(ctx)
	if err != nil {
		return Dashboard{}, err
	}
	widgets, err := s.Widgets(ctx)
	if err != nil {
		return Dashboard{}, err
	}
	if groups == nil {
		groups = []Group{}
	}
	if services == nil {
		services = []Service{}
	}
	if widgets == nil {
		widgets = []WidgetInstance{}
	}

	groupByID := make(map[string]*Group, len(groups))
	for i := range groups {
		groups[i].Services = []Service{}
		groupByID[groups[i].ID] = &groups[i]
	}

	ungrouped := make([]Service, 0)
	for _, service := range services {
		if service.GroupID == nil {
			ungrouped = append(ungrouped, service)
			continue
		}
		group := groupByID[*service.GroupID]
		if group == nil {
			ungrouped = append(ungrouped, service)
			continue
		}
		group.Services = append(group.Services, service)
	}

	return Dashboard{
		Groups:            groups,
		UngroupedServices: ungrouped,
		Widgets:           widgets,
	}, nil
}

func (s *Store) Groups(ctx context.Context) ([]Group, error) {
	rows, err := s.queries.ListGroups(ctx)
	if err != nil {
		return nil, err
	}

	groups := make([]Group, 0, len(rows))
	for _, row := range rows {
		groups = append(groups, Group{
			ID:        row.ID,
			Name:      row.Name,
			SortOrder: int(row.SortOrder),
			Collapsed: row.Collapsed,
			Services:  []Service{},
			CreatedAt: timestamptz(row.CreatedAt),
			UpdatedAt: timestamptz(row.UpdatedAt),
		})
	}
	return groups, nil
}

func (s *Store) Services(ctx context.Context) ([]Service, error) {
	rows, err := s.queries.ListServices(ctx)
	if err != nil {
		return nil, err
	}

	services := make([]Service, 0, len(rows))
	for _, row := range rows {
		services = append(services, Service{
			ID:          row.ID,
			GroupID:     uuidPtr(row.GroupID),
			Name:        row.Name,
			IconURL:     textPtr(row.IconUrl),
			IconAssetID: uuidPtr(row.IconAssetID),
			SortOrder:   int(row.SortOrder),
			URLs:        []ServiceURL{},
			CreatedAt:   timestamptz(row.CreatedAt),
			UpdatedAt:   timestamptz(row.UpdatedAt),
		})
	}
	if len(services) == 0 {
		return services, nil
	}
	urls, err := s.serviceURLs(ctx)
	if err != nil {
		return nil, err
	}
	for i := range services {
		if serviceURLs, ok := urls[services[i].ID]; ok {
			services[i].URLs = serviceURLs
		}
	}
	return services, nil
}

func (s *Store) Group(ctx context.Context, id string) (Group, error) {
	var group Group
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, name, sort_order, collapsed, created_at, updated_at
		FROM groups WHERE id = $1`, id).
		Scan(&group.ID, &group.Name, &group.SortOrder, &group.Collapsed, &group.CreatedAt, &group.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Group{}, ErrNotFound
	}
	if err != nil {
		return Group{}, err
	}
	group.Services = []Service{}
	return group, nil
}

func (s *Store) CreateGroup(ctx context.Context, name string) (Group, error) {
	var group Group
	err := s.pool.QueryRow(ctx, `
		INSERT INTO groups (name, sort_order)
		VALUES ($1, COALESCE((SELECT max(sort_order) + 1 FROM groups), 0))
		RETURNING id::text, name, sort_order, collapsed, created_at, updated_at`, name).
		Scan(&group.ID, &group.Name, &group.SortOrder, &group.Collapsed, &group.CreatedAt, &group.UpdatedAt)
	return group, err
}

func (s *Store) UpdateGroup(ctx context.Context, id string, input GroupInput) (Group, error) {
	group, err := s.Group(ctx, id)
	if err != nil {
		return Group{}, err
	}
	name := group.Name
	collapsed := group.Collapsed
	if input.Name != nil {
		name = *input.Name
	}
	if input.Collapsed != nil {
		collapsed = *input.Collapsed
	}
	err = s.pool.QueryRow(ctx, `
		UPDATE groups SET name = $2, collapsed = $3
		WHERE id = $1
		RETURNING id::text, name, sort_order, collapsed, created_at, updated_at`, id, name, collapsed).
		Scan(&group.ID, &group.Name, &group.SortOrder, &group.Collapsed, &group.CreatedAt, &group.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Group{}, ErrNotFound
	}
	return group, err
}

func (s *Store) DeleteGroup(ctx context.Context, id string, moveServices bool) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer rollback(ctx, tx)

	var count int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM services WHERE group_id = $1`, id).Scan(&count); err != nil {
		return err
	}
	if count > 0 && !moveServices {
		return ErrConflict
	}
	if count > 0 {
		_, err = tx.Exec(ctx, `
			WITH moved AS (
				SELECT id, row_number() OVER (ORDER BY sort_order, created_at) - 1 rn
				FROM services WHERE group_id = $1
			), base AS (
				SELECT COALESCE(max(sort_order) + 1, 0) next_order
				FROM services WHERE group_id IS NULL
			)
			UPDATE services s
			SET group_id = NULL, sort_order = base.next_order + moved.rn
			FROM moved, base
			WHERE s.id = moved.id`, id)
		if err != nil {
			return err
		}
	}
	tag, err := tx.Exec(ctx, `DELETE FROM groups WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return tx.Commit(ctx)
}

func (s *Store) Service(ctx context.Context, id string) (Service, error) {
	var service Service
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, group_id::text, name, icon_url, icon_asset_id::text, sort_order, created_at, updated_at
		FROM services WHERE id = $1`, id).
		Scan(&service.ID, &service.GroupID, &service.Name, &service.IconURL, &service.IconAssetID, &service.SortOrder, &service.CreatedAt, &service.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Service{}, ErrNotFound
	}
	if err != nil {
		return Service{}, err
	}
	urls, err := s.serviceURLs(ctx)
	if err != nil {
		return Service{}, err
	}
	service.URLs = urls[service.ID]
	return service, nil
}

func (s *Store) CreateService(ctx context.Context, input ServiceInput) (Service, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Service{}, err
	}
	defer rollback(ctx, tx)

	var service Service
	err = tx.QueryRow(ctx, `
		INSERT INTO services (group_id, name, icon_url, icon_asset_id, sort_order)
		VALUES ($1, $2, $3, $4, (
			SELECT COALESCE(max(sort_order) + 1, 0) FROM services
			WHERE group_id IS NOT DISTINCT FROM $1::uuid
		))
		RETURNING id::text, group_id::text, name, icon_url, icon_asset_id::text, sort_order, created_at, updated_at`,
		input.GroupID, input.Name, input.IconURL, input.IconAssetID).
		Scan(&service.ID, &service.GroupID, &service.Name, &service.IconURL, &service.IconAssetID, &service.SortOrder, &service.CreatedAt, &service.UpdatedAt)
	if err != nil {
		return Service{}, err
	}
	if err := replaceServiceURLs(ctx, tx, service.ID, input.URLs); err != nil {
		return Service{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Service{}, err
	}
	return s.Service(ctx, service.ID)
}

func (s *Store) UpdateService(ctx context.Context, id string, input ServiceInput) (Service, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Service{}, err
	}
	defer rollback(ctx, tx)

	tag, err := tx.Exec(ctx, `
		UPDATE services
		SET group_id = $2, name = $3, icon_url = $4, icon_asset_id = $5
		WHERE id = $1`, id, input.GroupID, input.Name, input.IconURL, input.IconAssetID)
	if err != nil {
		return Service{}, err
	}
	if tag.RowsAffected() == 0 {
		return Service{}, ErrNotFound
	}
	if err := replaceServiceURLs(ctx, tx, id, input.URLs); err != nil {
		return Service{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Service{}, err
	}
	return s.Service(ctx, id)
}

func (s *Store) DeleteService(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM services WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) Widgets(ctx context.Context) ([]WidgetInstance, error) {
	rows, err := s.queries.ListWidgets(ctx)
	if err != nil {
		return nil, err
	}

	widgets := make([]WidgetInstance, 0, len(rows))
	for _, row := range rows {
		widgets = append(widgets, WidgetInstance{
			ID:        row.ID,
			Type:      row.Type,
			Title:     row.Title,
			Enabled:   row.Enabled,
			SortOrder: int(row.SortOrder),
			Config:    json.RawMessage(row.Config),
			CreatedAt: timestamptz(row.CreatedAt),
			UpdatedAt: timestamptz(row.UpdatedAt),
		})
	}
	return widgets, nil
}

func (s *Store) Widget(ctx context.Context, id string) (WidgetInstance, error) {
	var widget WidgetInstance
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, type, title, enabled, sort_order, config, created_at, updated_at
		FROM widget_instances WHERE id = $1`, id).
		Scan(&widget.ID, &widget.Type, &widget.Title, &widget.Enabled, &widget.SortOrder, &widget.Config, &widget.CreatedAt, &widget.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return WidgetInstance{}, ErrNotFound
	}
	return widget, err
}

func (s *Store) CreateWidget(ctx context.Context, input WidgetInput) (WidgetInstance, error) {
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	var widget WidgetInstance
	err := s.pool.QueryRow(ctx, `
		INSERT INTO widget_instances (type, title, enabled, config, sort_order)
		VALUES ($1, $2, $3, $4, COALESCE((SELECT max(sort_order) + 1 FROM widget_instances), 0))
		RETURNING id::text, type, title, enabled, sort_order, config, created_at, updated_at`,
		input.Type, input.Title, enabled, jsonOrEmpty(input.Config)).
		Scan(&widget.ID, &widget.Type, &widget.Title, &widget.Enabled, &widget.SortOrder, &widget.Config, &widget.CreatedAt, &widget.UpdatedAt)
	return widget, err
}

func (s *Store) UpdateWidget(ctx context.Context, id string, input WidgetInput) (WidgetInstance, error) {
	current, err := s.Widget(ctx, id)
	if err != nil {
		return WidgetInstance{}, err
	}
	enabled := current.Enabled
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	widgetType := input.Type
	if widgetType == "" {
		widgetType = current.Type
	}
	title := input.Title
	if title == "" {
		title = current.Title
	}
	config := input.Config
	if len(config) == 0 {
		config = current.Config
	}
	var widget WidgetInstance
	err = s.pool.QueryRow(ctx, `
		UPDATE widget_instances
		SET type = $2, title = $3, enabled = $4, config = $5
		WHERE id = $1
		RETURNING id::text, type, title, enabled, sort_order, config, created_at, updated_at`,
		id, widgetType, title, enabled, jsonOrEmpty(config)).
		Scan(&widget.ID, &widget.Type, &widget.Title, &widget.Enabled, &widget.SortOrder, &widget.Config, &widget.CreatedAt, &widget.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return WidgetInstance{}, ErrNotFound
	}
	return widget, err
}

func (s *Store) DeleteWidget(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM widget_instances WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) WidgetData(ctx context.Context, widgetID string) (WidgetData, error) {
	var data WidgetData
	err := s.pool.QueryRow(ctx, `
		SELECT widget_id::text, status, data, error, fetched_at, expires_at
		FROM widget_cache WHERE widget_id = $1`, widgetID).
		Scan(&data.WidgetID, &data.Status, &data.Data, &data.Error, &data.FetchedAt, &data.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return WidgetData{}, ErrNotFound
	}
	return data, err
}

func (s *Store) SaveWidgetData(ctx context.Context, data WidgetData) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO widget_cache (widget_id, status, data, error, fetched_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (widget_id) DO UPDATE
		SET status = EXCLUDED.status,
		    data = EXCLUDED.data,
		    error = EXCLUDED.error,
		    fetched_at = EXCLUDED.fetched_at,
		    expires_at = EXCLUDED.expires_at`,
		data.WidgetID, data.Status, nilIfEmptyJSON(data.Data), data.Error, data.FetchedAt, data.ExpiresAt)
	return err
}

func (s *Store) CreateAsset(ctx context.Context, file AssetFile) (AssetFile, error) {
	err := s.pool.QueryRow(ctx, `
		INSERT INTO asset_files (original_name, stored_name, mime_type, size_bytes, public_path)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id::text, original_name, stored_name, mime_type, size_bytes, public_path, created_at`,
		file.OriginalName, file.StoredName, file.MimeType, file.SizeBytes, file.PublicPath).
		Scan(&file.ID, &file.OriginalName, &file.StoredName, &file.MimeType, &file.SizeBytes, &file.PublicPath, &file.CreatedAt)
	return file, err
}

func (s *Store) ApplyLayout(ctx context.Context, input LayoutInput) (Dashboard, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Dashboard{}, err
	}
	defer rollback(ctx, tx)

	if err := validateLayoutRefs(ctx, tx, input); err != nil {
		return Dashboard{}, err
	}
	for order, id := range input.GroupIDs {
		if _, err := tx.Exec(ctx, `UPDATE groups SET sort_order = $2 WHERE id = $1`, id, order); err != nil {
			return Dashboard{}, err
		}
	}
	for order, id := range input.WidgetIDs {
		if _, err := tx.Exec(ctx, `UPDATE widget_instances SET sort_order = $2 WHERE id = $1`, id, order); err != nil {
			return Dashboard{}, err
		}
	}
	for order, id := range input.UngroupedServices {
		if _, err := tx.Exec(ctx, `UPDATE services SET group_id = NULL, sort_order = $2 WHERE id = $1`, id, order); err != nil {
			return Dashboard{}, err
		}
	}
	groupIDs := make([]string, 0, len(input.GroupServices))
	for groupID := range input.GroupServices {
		groupIDs = append(groupIDs, groupID)
	}
	sort.Strings(groupIDs)
	for _, groupID := range groupIDs {
		for order, serviceID := range input.GroupServices[groupID] {
			if _, err := tx.Exec(ctx, `UPDATE services SET group_id = $2, sort_order = $3 WHERE id = $1`, serviceID, groupID, order); err != nil {
				return Dashboard{}, err
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return Dashboard{}, err
	}
	return s.Dashboard(ctx)
}

func (s *Store) serviceURLs(ctx context.Context) (map[string][]ServiceURL, error) {
	rows, err := s.queries.ListServiceURLs(ctx)
	if err != nil {
		return nil, err
	}

	urls := make(map[string][]ServiceURL)
	for _, row := range rows {
		serviceURL := ServiceURL{
			ID:        row.ID,
			ServiceID: row.ServiceID,
			Label:     row.Label,
			Kind:      row.Kind,
			URL:       row.Url,
			SortOrder: int(row.SortOrder),
			IsPrimary: row.IsPrimary,
			CreatedAt: timestamptz(row.CreatedAt),
			UpdatedAt: timestamptz(row.UpdatedAt),
		}
		urls[serviceURL.ServiceID] = append(urls[serviceURL.ServiceID], serviceURL)
	}
	return urls, nil
}

func replaceServiceURLs(ctx context.Context, tx pgx.Tx, serviceID string, urls []ServiceURLInput) error {
	if _, err := tx.Exec(ctx, `DELETE FROM service_urls WHERE service_id = $1`, serviceID); err != nil {
		return err
	}
	for i, serviceURL := range urls {
		_, err := tx.Exec(ctx, `
			INSERT INTO service_urls (service_id, label, kind, url, sort_order, is_primary)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			serviceID, serviceURL.Label, serviceURL.Kind, serviceURL.URL, i, serviceURL.IsPrimary)
		if err != nil {
			return err
		}
	}
	return nil
}

func validateLayoutRefs(ctx context.Context, tx pgx.Tx, input LayoutInput) error {
	if input.GroupServices == nil {
		return fmt.Errorf("%w: groupServices is required", ErrValidation)
	}
	if err := ensureFullIDSet(ctx, tx, "groups", input.GroupIDs); err != nil {
		return err
	}
	if err := ensureFullIDSet(ctx, tx, "widget_instances", input.WidgetIDs); err != nil {
		return err
	}
	allServices := append([]string{}, input.UngroupedServices...)
	for groupID, serviceIDs := range input.GroupServices {
		if err := ensureIDs(ctx, tx, "groups", []string{groupID}); err != nil {
			return err
		}
		allServices = append(allServices, serviceIDs...)
	}
	if err := noDuplicates(allServices, "service"); err != nil {
		return err
	}
	return ensureFullIDSet(ctx, tx, "services", allServices)
}

func ensureIDs(ctx context.Context, tx pgx.Tx, table string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	if err := noDuplicates(ids, table); err != nil {
		return err
	}
	rows, err := tx.Query(ctx, fmt.Sprintf(`SELECT id::text FROM %s WHERE id = ANY($1::uuid[])`, table), ids)
	if err != nil {
		return fmt.Errorf("%w: invalid %s id", ErrValidation, table)
	}
	defer rows.Close()
	seen := map[string]struct{}{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return err
		}
		seen[id] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(seen) != len(ids) {
		return ErrNotFound
	}
	return nil
}

func ensureFullIDSet(ctx context.Context, tx pgx.Tx, table string, ids []string) error {
	if err := ensureIDs(ctx, tx, table, ids); err != nil {
		return err
	}
	var count int
	if err := tx.QueryRow(ctx, fmt.Sprintf(`SELECT count(*) FROM %s`, table)).Scan(&count); err != nil {
		return err
	}
	if count != len(ids) {
		return fmt.Errorf("%w: %s layout must include every existing row exactly once", ErrValidation, table)
	}
	return nil
}

func noDuplicates(ids []string, label string) error {
	seen := map[string]struct{}{}
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			return fmt.Errorf("%w: %s id appears more than once", ErrValidation, label)
		}
		seen[id] = struct{}{}
	}
	return nil
}

func rollback(ctx context.Context, tx pgx.Tx) {
	_ = tx.Rollback(ctx)
}

func jsonOrEmpty(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage(`{}`)
	}
	return raw
}

func nilIfEmptyJSON(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}
	return raw
}

func Fresh(data WidgetData, now time.Time) bool {
	return data.Status == "fresh" && data.ExpiresAt != nil && data.ExpiresAt.After(now)
}

func timestamptz(value pgtype.Timestamptz) time.Time {
	if !value.Valid {
		return time.Time{}
	}
	return value.Time
}

func textPtr(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

func uuidPtr(value pgtype.UUID) *string {
	if !value.Valid {
		return nil
	}
	out := uuid.UUID(value.Bytes).String()
	return &out
}
