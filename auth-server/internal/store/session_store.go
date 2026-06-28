package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"crypto/rand"
	"encoding/base64"

	"github.com/redis/go-redis/v9"
)

type OAuthFlowParams struct {
	ClientID            string `json:"client_id"`
	RedirectURI         string `json:"redirect_uri"`
	Scope               string `json:"scope"`
	State               string `json:"state"`
	CodeChallenge       string `json:"code_challenge"`
	CodeChallengeMethod string `json:"code_challenge_method"`
	Nonce               string `json:"nonce,omitempty"`
}

type SessionData struct {
	MemberID           string           `json:"member_id"`
	Username           string           `json:"username"`
	MustChangePassword bool             `json:"must_change_password,omitempty"`
	IPAddress      string           `json:"ip_address"`
	UserAgent      string           `json:"user_agent"`
	CreatedAt      int64            `json:"created_at"`
	LastAccessedAt int64            `json:"last_accessed_at"`
	OAuthParams    *OAuthFlowParams `json:"oauth_params,omitempty"`
}

type SessionStore struct {
	client *redis.Client
	ttl    time.Duration
}

func NewSessionStore(client *redis.Client, ttl time.Duration) *SessionStore {
	return &SessionStore{client: client, ttl: ttl}
}

func (s *SessionStore) Create(ctx context.Context, data *SessionData) (string, error) {
	rb := make([]byte, 32)
	if _, err := rand.Read(rb); err != nil {
		return "", fmt.Errorf("generate session id: %w", err)
	}
	sessionID := base64.RawURLEncoding.EncodeToString(rb)

	now := time.Now().Unix()
	data.CreatedAt = now
	data.LastAccessedAt = now

	b, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("marshal session: %w", err)
	}

	if err := s.client.Set(ctx, "auth:session:"+sessionID, b, s.ttl).Err(); err != nil {
		return "", fmt.Errorf("save session: %w", err)
	}

	if data.MemberID != "" {
		memberSetKey := "auth:member_sessions:" + data.MemberID
		if err := s.client.SAdd(ctx, memberSetKey, sessionID).Err(); err != nil {
			return "", fmt.Errorf("track session: %w", err)
		}
		s.client.Expire(ctx, memberSetKey, s.ttl)
	}

	return sessionID, nil
}

func (s *SessionStore) Get(ctx context.Context, sessionID string) (*SessionData, error) {
	b, err := s.client.Get(ctx, "auth:session:"+sessionID).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	var data SessionData
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, fmt.Errorf("unmarshal session: %w", err)
	}
	return &data, nil
}

func (s *SessionStore) Update(ctx context.Context, sessionID string, data *SessionData) error {
	data.LastAccessedAt = time.Now().Unix()
	b, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}
	if err := s.client.Set(ctx, "auth:session:"+sessionID, b, s.ttl).Err(); err != nil {
		return fmt.Errorf("update session: %w", err)
	}
	if data.MemberID != "" {
		s.client.Expire(ctx, "auth:member_sessions:"+data.MemberID, s.ttl)
	}
	return nil
}

func (s *SessionStore) Delete(ctx context.Context, sessionID string) error {
	data, _ := s.Get(ctx, sessionID)
	if err := s.client.Del(ctx, "auth:session:"+sessionID).Err(); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	if data != nil && data.MemberID != "" {
		s.client.SRem(ctx, "auth:member_sessions:"+data.MemberID, sessionID)
	}
	return nil
}

func (s *SessionStore) ListByMember(ctx context.Context, memberID string) ([]map[string]any, error) {
	memberSetKey := "auth:member_sessions:" + memberID
	sessionIDs, err := s.client.SMembers(ctx, memberSetKey).Result()
	if err != nil {
		return nil, fmt.Errorf("list member sessions: %w", err)
	}

	var sessions []map[string]any
	for _, sid := range sessionIDs {
		data, err := s.Get(ctx, sid)
		if err != nil || data == nil {
			s.client.SRem(ctx, memberSetKey, sid)
			continue
		}
		sessions = append(sessions, map[string]any{
			"session_id":       sid,
			"ip_address":       data.IPAddress,
			"user_agent":       data.UserAgent,
			"created_at":       data.CreatedAt,
			"last_accessed_at": data.LastAccessedAt,
		})
	}
	return sessions, nil
}

func (s *SessionStore) DeleteByID(ctx context.Context, sessionID, memberID string) error {
	data, err := s.Get(ctx, sessionID)
	if err != nil || data == nil {
		return fmt.Errorf("session not found")
	}
	if data.MemberID != memberID {
		return fmt.Errorf("session does not belong to this member")
	}
	if err := s.client.Del(ctx, "auth:session:"+sessionID).Err(); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	s.client.SRem(ctx, "auth:member_sessions:"+memberID, sessionID)
	return nil
}
