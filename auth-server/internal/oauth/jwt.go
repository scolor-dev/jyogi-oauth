package oauth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type JWTService struct {
	privateKey     *ecdsa.PrivateKey
	publicKey      *ecdsa.PublicKey
	kid            string
	issuer         string
	accessTokenTTL time.Duration
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

	return &JWTService{
		privateKey:     privKey,
		publicKey:      pubKey,
		kid:            kid,
		issuer:         issuer,
		accessTokenTTL: accessTokenTTL,
	}, nil
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
	token.Header["kid"] = j.kid

	return token.SignedString(j.privateKey)
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
	token.Header["kid"] = j.kid

	return token.SignedString(j.privateKey)
}

type IDTokenClaims struct {
	MemberID string
	ClientID string
	Nonce    string
	AuthTime int64
	Scope    string
	// profile claims
	Name              string
	PreferredUsername  string
	// identity claims
	Picture string
	Color   string
	Tagline string
}

func (j *JWTService) SignIDToken(claims IDTokenClaims) (string, error) {
	now := time.Now()

	mapClaims := jwt.MapClaims{
		"iss":       j.issuer,
		"sub":       claims.MemberID,
		"aud":       claims.ClientID,
		"exp":       jwt.NewNumericDate(now.Add(j.accessTokenTTL)),
		"iat":       jwt.NewNumericDate(now),
		"auth_time": claims.AuthTime,
	}

	if claims.Nonce != "" {
		mapClaims["nonce"] = claims.Nonce
	}

	scopes := splitScopes(claims.Scope)
	if contains(scopes, "profile") {
		if claims.Name != "" {
			mapClaims["name"] = claims.Name
		}
		if claims.PreferredUsername != "" {
			mapClaims["preferred_username"] = claims.PreferredUsername
		}
	}
	if contains(scopes, "identity") {
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
	token.Header["kid"] = j.kid

	return token.SignedString(j.privateKey)
}

func splitScopes(s string) []string {
	var out []string
	for _, p := range strings.Split(s, " ") {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func contains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

func (j *JWTService) VerifyToken(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return j.publicKey, nil
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
	return JWKSResponse{
		Keys: []JWK{
			{
				Kty: "EC",
				Use: "sig",
				Kid: j.kid,
				Alg: "ES256",
				Crv: "P-256",
				X:   base64URLEncode(j.publicKey.X),
				Y:   base64URLEncode(j.publicKey.Y),
			},
		},
	}
}

func base64URLEncode(n *big.Int) string {
	b := n.Bytes()
	// P-256 coordinates are 32 bytes
	padded := make([]byte, 32)
	copy(padded[32-len(b):], b)
	return base64.RawURLEncoding.EncodeToString(padded)
}
