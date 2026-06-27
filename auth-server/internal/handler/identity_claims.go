package handler

import (
	"context"
	"errors"
	"log"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type identityClaims struct {
	DisplayName *string
	AvatarURL   *string
	ThemeColor  *string
	Tagline     *string
}

func getIdentityClaims(ctx context.Context, pool *pgxpool.Pool, memberID uuid.UUID) *identityClaims {
	var ic identityClaims
	err := pool.QueryRow(ctx,
		`SELECT display_name, avatar_url, theme_color, tagline FROM resource.member_identities WHERE member_id = $1`,
		memberID,
	).Scan(&ic.DisplayName, &ic.AvatarURL, &ic.ThemeColor, &ic.Tagline)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			log.Printf("Failed to fetch identity claims for member %s: %v", memberID, err)
		}
		return nil
	}
	return &ic
}
