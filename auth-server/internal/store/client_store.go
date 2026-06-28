package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jyogi-oauth/auth-server/internal/model"
)

type ClientStore struct {
	pool *pgxpool.Pool
}

func NewClientStore(pool *pgxpool.Pool) *ClientStore {
	return &ClientStore{pool: pool}
}

func (s *ClientStore) Create(ctx context.Context, clientID, name, clientType string, clientSecretHash *string, description *string, iconURL *string, redirectURIs, allowedGrantTypes []string, createdBy *uuid.UUID) (*model.Client, error) {
	c := &model.Client{}
	err := s.pool.QueryRow(ctx,
		`INSERT INTO auth.clients (client_id, client_secret_hash, name, description, icon_url, client_type, redirect_uris, allowed_grant_types, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING id, client_id, client_secret_hash, name, description, icon_url, client_type, redirect_uris, allowed_grant_types, is_active, created_by, created_at, updated_at`,
		clientID, clientSecretHash, name, description, iconURL, clientType, redirectURIs, allowedGrantTypes, createdBy,
	).Scan(&c.ID, &c.ClientID, &c.ClientSecretHash, &c.Name, &c.Description, &c.IconURL, &c.ClientType, &c.RedirectURIs, &c.AllowedGrantTypes, &c.IsActive, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert client: %w", err)
	}
	return c, nil
}

func (s *ClientStore) GetByClientID(ctx context.Context, clientID string) (*model.Client, error) {
	c := &model.Client{}
	err := s.pool.QueryRow(ctx,
		`SELECT id, client_id, client_secret_hash, name, description, icon_url, client_type, redirect_uris, allowed_grant_types, is_active, created_by, created_at, updated_at
		 FROM auth.clients WHERE client_id = $1`, clientID,
	).Scan(&c.ID, &c.ClientID, &c.ClientSecretHash, &c.Name, &c.Description, &c.IconURL, &c.ClientType, &c.RedirectURIs, &c.AllowedGrantTypes, &c.IsActive, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get client by client_id: %w", err)
	}
	return c, nil
}

func (s *ClientStore) GetByID(ctx context.Context, id uuid.UUID) (*model.Client, error) {
	c := &model.Client{}
	err := s.pool.QueryRow(ctx,
		`SELECT id, client_id, client_secret_hash, name, description, icon_url, client_type, redirect_uris, allowed_grant_types, is_active, created_by, created_at, updated_at
		 FROM auth.clients WHERE id = $1`, id,
	).Scan(&c.ID, &c.ClientID, &c.ClientSecretHash, &c.Name, &c.Description, &c.IconURL, &c.ClientType, &c.RedirectURIs, &c.AllowedGrantTypes, &c.IsActive, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get client: %w", err)
	}
	return c, nil
}

func (s *ClientStore) List(ctx context.Context, page, perPage int) ([]model.Client, int, error) {
	var total int
	err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM auth.clients`).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count clients: %w", err)
	}

	offset := (page - 1) * perPage
	rows, err := s.pool.Query(ctx,
		`SELECT id, client_id, client_secret_hash, name, description, icon_url, client_type, redirect_uris, allowed_grant_types, is_active, created_by, created_at, updated_at
			 FROM auth.clients ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		perPage, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list clients: %w", err)
	}
	defer rows.Close()

	var clients []model.Client
	for rows.Next() {
		var c model.Client
		if err := rows.Scan(&c.ID, &c.ClientID, &c.ClientSecretHash, &c.Name, &c.Description, &c.IconURL, &c.ClientType, &c.RedirectURIs, &c.AllowedGrantTypes, &c.IsActive, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan client: %w", err)
		}
		clients = append(clients, c)
	}
	return clients, total, nil
}

func (s *ClientStore) Update(ctx context.Context, id uuid.UUID, name *string, description *string, iconURL *string, redirectURIs []string, allowedGrantTypes []string, isActive *bool) error {
	nameVal := nullableString(name)
	descriptionVal := nullableString(description)
	iconURLVal := nullableString(iconURL)
	isActiveVal := nullableBool(isActive)
	_, err := s.pool.Exec(ctx,
		`UPDATE auth.clients SET
			name = COALESCE($2, name),
			description = COALESCE($3, description),
			icon_url = COALESCE($4, icon_url),
			redirect_uris = CASE WHEN $5::text[] IS NULL THEN redirect_uris ELSE $5 END,
			allowed_grant_types = CASE WHEN $6::text[] IS NULL THEN allowed_grant_types ELSE $6 END,
			is_active = COALESCE($7, is_active)
		 WHERE id = $1`,
		id, nameVal, descriptionVal, iconURLVal, redirectURIs, allowedGrantTypes, isActiveVal,
	)
	if err != nil {
		return fmt.Errorf("update client: %w", err)
	}
	return nil
}

func nullableString(v *string) any {
	if v == nil {
		return nil
	}
	return *v
}

func nullableBool(v *bool) any {
	if v == nil {
		return nil
	}
	return *v
}

func (s *ClientStore) ListByCreator(ctx context.Context, creatorID uuid.UUID) ([]model.Client, int, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, client_id, client_secret_hash, name, description, icon_url, client_type, redirect_uris, allowed_grant_types, is_active, created_by, created_at, updated_at
		 FROM auth.clients WHERE created_by = $1 AND is_active = true ORDER BY created_at DESC`,
		creatorID,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list clients by creator: %w", err)
	}
	defer rows.Close()

	var clients []model.Client
	for rows.Next() {
		var c model.Client
		if err := rows.Scan(&c.ID, &c.ClientID, &c.ClientSecretHash, &c.Name, &c.Description, &c.IconURL, &c.ClientType, &c.RedirectURIs, &c.AllowedGrantTypes, &c.IsActive, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan client: %w", err)
		}
		clients = append(clients, c)
	}
	return clients, len(clients), nil
}

func (s *ClientStore) UpdateSecret(ctx context.Context, id uuid.UUID, secretHash string) error {
	_, err := s.pool.Exec(ctx, `UPDATE auth.clients SET client_secret_hash = $1 WHERE id = $2`, secretHash, id)
	if err != nil {
		return fmt.Errorf("update client secret: %w", err)
	}
	return nil
}

func (s *ClientStore) Deactivate(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `UPDATE auth.clients SET is_active = false WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deactivate client: %w", err)
	}
	return nil
}
