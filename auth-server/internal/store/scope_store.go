package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jyogi-oauth/auth-server/internal/model"
)

type ScopeStore struct {
	pool *pgxpool.Pool
}

func NewScopeStore(pool *pgxpool.Pool) *ScopeStore {
	return &ScopeStore{pool: pool}
}

func (s *ScopeStore) Create(ctx context.Context, name string, description *string, isDefault bool) (*model.Scope, error) {
	sc := &model.Scope{}
	err := s.pool.QueryRow(ctx,
		`INSERT INTO auth.scopes (name, description, is_default) VALUES ($1, $2, $3)
		 RETURNING id, name, description, is_default, created_at`,
		name, description, isDefault,
	).Scan(&sc.ID, &sc.Name, &sc.Description, &sc.IsDefault, &sc.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert scope: %w", err)
	}
	return sc, nil
}

func (s *ScopeStore) GetByID(ctx context.Context, id uuid.UUID) (*model.Scope, error) {
	sc := &model.Scope{}
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, description, is_default, created_at FROM auth.scopes WHERE id = $1`, id,
	).Scan(&sc.ID, &sc.Name, &sc.Description, &sc.IsDefault, &sc.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get scope: %w", err)
	}
	return sc, nil
}

func (s *ScopeStore) GetByName(ctx context.Context, name string) (*model.Scope, error) {
	sc := &model.Scope{}
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, description, is_default, created_at FROM auth.scopes WHERE name = $1`, name,
	).Scan(&sc.ID, &sc.Name, &sc.Description, &sc.IsDefault, &sc.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get scope by name: %w", err)
	}
	return sc, nil
}

func (s *ScopeStore) GetByNames(ctx context.Context, names []string) ([]model.Scope, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, name, description, is_default, created_at FROM auth.scopes WHERE name = ANY($1)`, names,
	)
	if err != nil {
		return nil, fmt.Errorf("get scopes by names: %w", err)
	}
	defer rows.Close()

	var scopes []model.Scope
	for rows.Next() {
		var sc model.Scope
		if err := rows.Scan(&sc.ID, &sc.Name, &sc.Description, &sc.IsDefault, &sc.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan scope: %w", err)
		}
		scopes = append(scopes, sc)
	}
	return scopes, nil
}

func (s *ScopeStore) List(ctx context.Context) ([]model.Scope, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, name, description, is_default, created_at FROM auth.scopes ORDER BY name`,
	)
	if err != nil {
		return nil, fmt.Errorf("list scopes: %w", err)
	}
	defer rows.Close()

	var scopes []model.Scope
	for rows.Next() {
		var sc model.Scope
		if err := rows.Scan(&sc.ID, &sc.Name, &sc.Description, &sc.IsDefault, &sc.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan scope: %w", err)
		}
		scopes = append(scopes, sc)
	}
	return scopes, nil
}

func (s *ScopeStore) Update(ctx context.Context, id uuid.UUID, name *string, description *string, isDefault *bool) error {
	if name != nil {
		if _, err := s.pool.Exec(ctx, `UPDATE auth.scopes SET name = $1 WHERE id = $2`, *name, id); err != nil {
			return fmt.Errorf("update name: %w", err)
		}
	}
	if description != nil {
		if _, err := s.pool.Exec(ctx, `UPDATE auth.scopes SET description = $1 WHERE id = $2`, *description, id); err != nil {
			return fmt.Errorf("update description: %w", err)
		}
	}
	if isDefault != nil {
		if _, err := s.pool.Exec(ctx, `UPDATE auth.scopes SET is_default = $1 WHERE id = $2`, *isDefault, id); err != nil {
			return fmt.Errorf("update is_default: %w", err)
		}
	}
	return nil
}

func (s *ScopeStore) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM auth.scopes WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete scope: %w", err)
	}
	return nil
}
