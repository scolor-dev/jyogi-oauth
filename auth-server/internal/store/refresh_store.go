package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"crypto/sha256"
	"encoding/hex"

	"github.com/redis/go-redis/v9"
)

type RefreshTokenData struct {
	MemberID  string `json:"member_id"`
	ClientID  string `json:"client_id"`
	Scope     string `json:"scope"`
	IssuedAt  int64  `json:"issued_at"`
	ExpiresAt int64  `json:"expires_at"`
}

type RefreshStore struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRefreshStore(client *redis.Client, ttl time.Duration) *RefreshStore {
	return &RefreshStore{client: client, ttl: ttl}
}

func (s *RefreshStore) Save(ctx context.Context, token string, data *RefreshTokenData) error {
	tokenHash := hashToken(token)
	now := time.Now()
	data.IssuedAt = now.Unix()
	data.ExpiresAt = now.Add(s.ttl).Unix()

	b, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal refresh token: %w", err)
	}

	pipe := s.client.Pipeline()
	pipe.Set(ctx, "auth:refresh:"+tokenHash, b, s.ttl)
	pipe.SAdd(ctx, "auth:member_refreshes:"+data.MemberID, tokenHash)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("save refresh token: %w", err)
	}
	return nil
}

func (s *RefreshStore) Get(ctx context.Context, token string) (*RefreshTokenData, error) {
	tokenHash := hashToken(token)
	b, err := s.client.Get(ctx, "auth:refresh:"+tokenHash).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get refresh token: %w", err)
	}

	var data RefreshTokenData
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, fmt.Errorf("unmarshal refresh token: %w", err)
	}
	return &data, nil
}

func (s *RefreshStore) Delete(ctx context.Context, token, memberID string) error {
	tokenHash := hashToken(token)
	pipe := s.client.Pipeline()
	pipe.Del(ctx, "auth:refresh:"+tokenHash)
	pipe.SRem(ctx, "auth:member_refreshes:"+memberID, tokenHash)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("delete refresh token: %w", err)
	}
	return nil
}

func (s *RefreshStore) DeleteAllForMember(ctx context.Context, memberID string) error {
	key := "auth:member_refreshes:" + memberID
	hashes, err := s.client.SMembers(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("get member refreshes: %w", err)
	}

	if len(hashes) == 0 {
		return nil
	}

	pipe := s.client.Pipeline()
	for _, h := range hashes {
		pipe.Del(ctx, "auth:refresh:"+h)
	}
	pipe.Del(ctx, key)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("delete all refresh tokens: %w", err)
	}
	return nil
}

func (s *RefreshStore) ListForMember(ctx context.Context, memberID string) ([]string, error) {
	return s.client.SMembers(ctx, "auth:member_refreshes:"+memberID).Result()
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
