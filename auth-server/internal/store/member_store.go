package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jyogi-oauth/auth-server/internal/model"
)

type MemberStore struct {
	pool *pgxpool.Pool
}

func NewMemberStore(pool *pgxpool.Pool) *MemberStore {
	return &MemberStore{pool: pool}
}

func (s *MemberStore) Create(ctx context.Context, username, passwordHash, email string, mustChangePassword bool) (*model.Member, error) {
	m := &model.Member{}
	err := s.pool.QueryRow(ctx,
		`INSERT INTO auth.members (username, password_hash, email, must_change_password)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, username, password_hash, email, role, must_change_password, is_active, created_at, updated_at`,
		username, passwordHash, email, mustChangePassword,
	).Scan(&m.ID, &m.Username, &m.PasswordHash, &m.Email, &m.Role, &m.MustChangePassword, &m.IsActive, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert member: %w", err)
	}
	return m, nil
}

func (s *MemberStore) GetByID(ctx context.Context, id uuid.UUID) (*model.Member, error) {
	m := &model.Member{}
	err := s.pool.QueryRow(ctx,
		`SELECT id, username, password_hash, email, role, must_change_password, is_active, created_at, updated_at
		 FROM auth.members WHERE id = $1`, id,
	).Scan(&m.ID, &m.Username, &m.PasswordHash, &m.Email, &m.Role, &m.MustChangePassword, &m.IsActive, &m.CreatedAt, &m.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get member: %w", err)
	}
	return m, nil
}

func (s *MemberStore) GetByUsername(ctx context.Context, username string) (*model.Member, error) {
	m := &model.Member{}
	err := s.pool.QueryRow(ctx,
		`SELECT id, username, password_hash, email, role, must_change_password, is_active, created_at, updated_at
		 FROM auth.members WHERE username = $1`, username,
	).Scan(&m.ID, &m.Username, &m.PasswordHash, &m.Email, &m.Role, &m.MustChangePassword, &m.IsActive, &m.CreatedAt, &m.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get member by username: %w", err)
	}
	return m, nil
}

func (s *MemberStore) List(ctx context.Context, page, perPage int) ([]model.Member, int, error) {
	var total int
	err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM auth.members`).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count members: %w", err)
	}

	offset := (page - 1) * perPage
	rows, err := s.pool.Query(ctx,
		`SELECT id, username, password_hash, email, role, must_change_password, is_active, created_at, updated_at
		 FROM auth.members ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		perPage, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list members: %w", err)
	}
	defer rows.Close()

	var members []model.Member
	for rows.Next() {
		var m model.Member
		if err := rows.Scan(&m.ID, &m.Username, &m.PasswordHash, &m.Email, &m.Role, &m.MustChangePassword, &m.IsActive, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan member: %w", err)
		}
		members = append(members, m)
	}
	return members, total, nil
}

func (s *MemberStore) Update(ctx context.Context, id uuid.UUID, username, email *string, isActive *bool) error {
	usernameVal := nullableString(username)
	emailVal := nullableString(email)
	isActiveVal := nullableBool(isActive)
	_, err := s.pool.Exec(ctx,
		`UPDATE auth.members SET
			username = COALESCE($2, username),
			email = COALESCE($3, email),
			is_active = COALESCE($4, is_active)
		 WHERE id = $1`,
		id, usernameVal, emailVal, isActiveVal,
	)
	if err != nil {
		return fmt.Errorf("update member: %w", err)
	}
	return nil
}

func (s *MemberStore) UpdateRole(ctx context.Context, id uuid.UUID, role string) error {
	_, err := s.pool.Exec(ctx, `UPDATE auth.members SET role = $1 WHERE id = $2`, role, id)
	if err != nil {
		return fmt.Errorf("update role: %w", err)
	}
	return nil
}

func (s *MemberStore) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	_, err := s.pool.Exec(ctx, `UPDATE auth.members SET password_hash = $1 WHERE id = $2`, passwordHash, id)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	return nil
}

func (s *MemberStore) SetMustChangePassword(ctx context.Context, id uuid.UUID, val bool) error {
	_, err := s.pool.Exec(ctx, `UPDATE auth.members SET must_change_password = $1 WHERE id = $2`, val, id)
	if err != nil {
		return fmt.Errorf("set must_change_password: %w", err)
	}
	return nil
}

func (s *MemberStore) ResetPassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE auth.members SET password_hash = $1, must_change_password = true WHERE id = $2`,
		passwordHash, id,
	)
	if err != nil {
		return fmt.Errorf("reset password: %w", err)
	}
	return nil
}

func (s *MemberStore) Deactivate(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `UPDATE auth.members SET is_active = false WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deactivate member: %w", err)
	}
	return nil
}
