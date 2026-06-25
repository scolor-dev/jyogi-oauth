package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type AuthCodeData struct {
	ClientID            string `json:"client_id"`
	MemberID            string `json:"member_id"`
	RedirectURI         string `json:"redirect_uri"`
	Scope               string `json:"scope"`
	CodeChallenge       string `json:"code_challenge"`
	CodeChallengeMethod string `json:"code_challenge_method"`
	IssuedAt            int64  `json:"issued_at"`
}

type AuthCodeStore struct {
	client *redis.Client
	ttl    time.Duration
}

func NewAuthCodeStore(client *redis.Client, ttl time.Duration) *AuthCodeStore {
	return &AuthCodeStore{client: client, ttl: ttl}
}

func (s *AuthCodeStore) Save(ctx context.Context, code string, data *AuthCodeData) error {
	data.IssuedAt = time.Now().Unix()
	b, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal auth code: %w", err)
	}
	if err := s.client.Set(ctx, "auth:code:"+code, b, s.ttl).Err(); err != nil {
		return fmt.Errorf("save auth code: %w", err)
	}
	return nil
}

var consumeScript = redis.NewScript(`
local val = redis.call('GET', KEYS[1])
if val then
    redis.call('DEL', KEYS[1])
end
return val
`)

func (s *AuthCodeStore) Consume(ctx context.Context, code string) (*AuthCodeData, error) {
	result, err := consumeScript.Run(ctx, s.client, []string{"auth:code:" + code}).Result()
	if err == redis.Nil || result == nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("consume auth code: %w", err)
	}

	var data AuthCodeData
	if err := json.Unmarshal([]byte(result.(string)), &data); err != nil {
		return nil, fmt.Errorf("unmarshal auth code: %w", err)
	}
	return &data, nil
}
