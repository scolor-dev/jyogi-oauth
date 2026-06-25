package oauth

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/http"

	"github.com/jyogi-oauth/auth-server/internal/model"
	"github.com/jyogi-oauth/auth-server/internal/store"
)

func HashClientSecret(secret string) string {
	h := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(h[:])
}

func AuthenticateClient(ctx context.Context, r *http.Request, clientStore *store.ClientStore) (*model.Client, string, error) {
	clientID, clientSecret, hasBasic := r.BasicAuth()
	if !hasBasic {
		clientID = r.FormValue("client_id")
		clientSecret = r.FormValue("client_secret")
	}

	if clientID == "" {
		return nil, "", nil
	}

	client, err := clientStore.GetByClientID(ctx, clientID)
	if err != nil {
		return nil, "", err
	}
	if client == nil || !client.IsActive {
		return nil, "", nil
	}

	if client.ClientType == "confidential" {
		if clientSecret == "" || client.ClientSecretHash == nil {
			return nil, "", nil
		}
		hash := HashClientSecret(clientSecret)
		if subtle.ConstantTimeCompare([]byte(hash), []byte(*client.ClientSecretHash)) != 1 {
			return nil, "", nil
		}
	}

	return client, clientSecret, nil
}
