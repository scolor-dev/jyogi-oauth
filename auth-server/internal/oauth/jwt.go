package oauth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func HasScope(scopeStr, target string) bool {
	return slices.Contains(strings.Fields(scopeStr), target)
}

type JWTService struct {
	activeKey      *SigningKey
	keys           []SigningKey
	issuer         string
	accessTokenTTL time.Duration
}

type SigningKey struct {
	KID        string
	PrivateKey *ecdsa.PrivateKey
	PublicKey  *ecdsa.PublicKey
}

type AccessTokenClaims struct {
	MemberID string
	ClientID string
	Scope    string
	Username string
}

type JWKSResponse struct {
	Keys []JWK `json:"keys"`
}

type JWK struct {
	Kty string `json:"kty"`
	Use string `json:"use"`
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	Crv string `json:"crv"`
	X   string `json:"x"`
	Y   string `json:"y"`
}

func NewJWTService(privateKeyPath, publicKeyPath, kid, issuer string, accessTokenTTL time.Duration) (*JWTService, error) {
	privPEM, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read private key: %w", err)
	}

	block, _ := pem.Decode(privPEM)
	if block == nil {
		return nil, fmt.Errorf("decode private key PEM")
	}

	privKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	if privKey.Curve != elliptic.P256() {
		return nil, fmt.Errorf("key must use P-256 curve for ES256")
	}

	pubPEM, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read public key: %w", err)
	}

	pubBlock, _ := pem.Decode(pubPEM)
	if pubBlock == nil {
		return nil, fmt.Errorf("decode public key PEM")
	}

	pubIface, err := x509.ParsePKIXPublicKey(pubBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}

	pubKey, ok := pubIface.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("public key is not ECDSA")
	}

	key := SigningKey{KID: kid, PrivateKey: privKey, PublicKey: pubKey}
	return &JWTService{
		activeKey:      &key,
		keys:           []SigningKey{key},
		issuer:         issuer,
		accessTokenTTL: accessTokenTTL,
	}, nil
}

func NewJWTServiceFromDir(keysDir, activeKID, issuer string, accessTokenTTL time.Duration) (*JWTService, error) {
	matches, err := filepath.Glob(filepath.Join(keysDir, "*.public.pem"))
	if err != nil {
		return nil, fmt.Errorf("scan public keys: %w", err)
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("no public keys found in %s", keysDir)
	}

	var keys []SigningKey
	activeIndex := -1
	for _, publicPath := range matches {
		kid := strings.TrimSuffix(filepath.Base(publicPath), ".public.pem")
		privatePath := filepath.Join(keysDir, kid+".private.pem")

		pubKey, err := loadPublicKey(publicPath)
		if err != nil {
			return nil, err
		}

		var privateKey *ecdsa.PrivateKey
		if _, err := os.Stat(privatePath); err == nil {
			privateKey, err = loadPrivateKey(privatePath)
			if err != nil {
				return nil, err
			}
		}

		key := SigningKey{KID: kid, PrivateKey: privateKey, PublicKey: pubKey}
		keys = append(keys, key)
		if kid == activeKID {
			activeIndex = len(keys) - 1
		}
	}

	if activeIndex == -1 {
		return nil, fmt.Errorf("active JWT key %q not found", activeKID)
	}
	active := &keys[activeIndex]
	if active.PrivateKey == nil {
		return nil, fmt.Errorf("active JWT key %q is missing private key", activeKID)
	}

	return &JWTService{
		activeKey:      active,
		keys:           keys,
		issuer:         issuer,
		accessTokenTTL: accessTokenTTL,
	}, nil
}

func loadPrivateKey(path string) (*ecdsa.PrivateKey, error) {
	privPEM, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read private key %s: %w", path, err)
	}
	block, _ := pem.Decode(privPEM)
	if block == nil {
		return nil, fmt.Errorf("decode private key PEM %s", path)
	}
	privKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key %s: %w", path, err)
	}
	if privKey.Curve != elliptic.P256() {
		return nil, fmt.Errorf("key %s must use P-256 curve for ES256", path)
	}
	return privKey, nil
}

func loadPublicKey(path string) (*ecdsa.PublicKey, error) {
	pubPEM, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read public key %s: %w", path, err)
	}
	pubBlock, _ := pem.Decode(pubPEM)
	if pubBlock == nil {
		return nil, fmt.Errorf("decode public key PEM %s", path)
	}
	pubIface, err := x509.ParsePKIXPublicKey(pubBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse public key %s: %w", path, err)
	}
	pubKey, ok := pubIface.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("public key %s is not ECDSA", path)
	}
	if pubKey.Curve != elliptic.P256() {
		return nil, fmt.Errorf("public key %s must use P-256 curve for ES256", path)
	}
	return pubKey, nil
}

func (j *JWTService) SignAccessToken(claims AccessTokenClaims) (string, error) {
	now := time.Now()
	jti := uuid.New().String()

	mapClaims := jwt.MapClaims{
		"iss":       j.issuer,
		"sub":       claims.MemberID,
		"aud":       j.issuer + "/api",
		"exp":       jwt.NewNumericDate(now.Add(j.accessTokenTTL)),
		"iat":       jwt.NewNumericDate(now),
		"jti":       jti,
		"client_id": claims.ClientID,
		"scope":     claims.Scope,
	}

	if claims.Username != "" {
		mapClaims["username"] = claims.Username
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, mapClaims)
	token.Header["kid"] = j.activeKey.KID

	return token.SignedString(j.activeKey.PrivateKey)
}

func (j *JWTService) SignClientCredentialsToken(clientID, scope string) (string, error) {
	now := time.Now()
	jti := uuid.New().String()

	mapClaims := jwt.MapClaims{
		"iss":        j.issuer,
		"sub":        clientID,
		"aud":        j.issuer + "/api",
		"exp":        jwt.NewNumericDate(now.Add(j.accessTokenTTL)),
		"iat":        jwt.NewNumericDate(now),
		"jti":        jti,
		"client_id":  clientID,
		"scope":      scope,
		"grant_type": "client_credentials",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, mapClaims)
	token.Header["kid"] = j.activeKey.KID

	return token.SignedString(j.activeKey.PrivateKey)
}

type IDTokenClaims struct {
	MemberID    string
	ClientID    string
	Nonce       string
	AuthTime    int64
	Scope       string
	AccessToken string
	// profile claims
	Name              string
	PreferredUsername string
	// identity claims
	Picture string
	Color   string
	Tagline string
}

func (j *JWTService) SignIDToken(claims IDTokenClaims) (string, error) {
	now := time.Now()

	mapClaims := jwt.MapClaims{
		"iss": j.issuer,
		"sub": claims.MemberID,
		"aud": claims.ClientID,
		"exp": jwt.NewNumericDate(now.Add(j.accessTokenTTL)),
		"iat": jwt.NewNumericDate(now),
	}

	if claims.AuthTime != 0 {
		mapClaims["auth_time"] = claims.AuthTime
	}

	if claims.AccessToken != "" {
		mapClaims["at_hash"] = computeAtHash(claims.AccessToken)
	}

	if claims.Nonce != "" {
		mapClaims["nonce"] = claims.Nonce
	}

	if HasScope(claims.Scope, "profile") {
		if claims.Name != "" {
			mapClaims["name"] = claims.Name
		}
		if claims.PreferredUsername != "" {
			mapClaims["preferred_username"] = claims.PreferredUsername
		}
	}
	if HasScope(claims.Scope, "identity") {
		if claims.Name != "" {
			mapClaims["name"] = claims.Name
		}
		if claims.Picture != "" {
			mapClaims["picture"] = claims.Picture
		}
		if claims.Color != "" {
			mapClaims["color"] = claims.Color
		}
		if claims.Tagline != "" {
			mapClaims["tagline"] = claims.Tagline
		}
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, mapClaims)
	token.Header["kid"] = j.activeKey.KID

	return token.SignedString(j.activeKey.PrivateKey)
}

func computeAtHash(accessToken string) string {
	h := sha256.Sum256([]byte(accessToken))
	return base64.RawURLEncoding.EncodeToString(h[:16])
}

func (j *JWTService) VerifyToken(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		kid, _ := t.Header["kid"].(string)
		for i := range j.keys {
			if j.keys[i].KID == kid {
				return j.keys[i].PublicKey, nil
			}
		}
		return nil, fmt.Errorf("unknown kid: %s", kid)
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

func (j *JWTService) GetJWKS() JWKSResponse {
	keys := make([]JWK, 0, len(j.keys))
	for _, key := range j.keys {
		keys = append(keys, JWK{
			Kty: "EC",
			Use: "sig",
			Kid: key.KID,
			Alg: "ES256",
			Crv: "P-256",
			X:   base64URLEncode(key.PublicKey.X),
			Y:   base64URLEncode(key.PublicKey.Y),
		})
	}
	return JWKSResponse{Keys: keys}
}

func base64URLEncode(n *big.Int) string {
	b := n.Bytes()
	// P-256 coordinates are 32 bytes
	padded := make([]byte, 32)
	copy(padded[32-len(b):], b)
	return base64.RawURLEncoding.EncodeToString(padded)
}
