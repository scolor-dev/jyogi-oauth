# JWT クレーム定義

## 概要

アクセストークンは JWT（JSON Web Token）として発行する。
署名アルゴリズムは **ES256**（ECDSA P-256）。

リソースサーバーは JWT の署名と exp のみで検証を完結する（Redis 参照なし）。

---

## JWT ヘッダー

```json
{
  "alg": "ES256",
  "typ": "JWT",
  "kid": "key-1"
}
```

| フィールド | 値 | 説明 |
|---|---|---|
| alg | `ES256` | 署名アルゴリズム（ECDSA P-256 + SHA-256） |
| typ | `JWT` | トークン種別 |
| kid | `key-1` | 鍵 ID。JWKS エンドポイントの鍵と対応。鍵ローテーション時に変更 |

---

## JWT ペイロード（クレーム）

### アクセストークン

```json
{
  "iss": "https://oauth.example.internal",
  "sub": "550e8400-e29b-41d4-a716-446655440000",
  "aud": "https://oauth.example.internal/api",
  "exp": 1719197300,
  "iat": 1719196400,
  "jti": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "client_id": "my-app",
  "scope": "read write",
  "username": "tanaka"
}
```

| クレーム | 型 | 必須 | 説明 |
|---|---|---|---|
| iss | string | Yes | 発行者（認可サーバーの URL） |
| sub | string (UUID) | Yes | メンバー ID（auth.members.id） |
| aud | string | Yes | 対象者（リソースサーバーの URL） |
| exp | number | Yes | 有効期限（Unix timestamp）。発行から15分後 |
| iat | number | Yes | 発行日時（Unix timestamp） |
| jti | string (UUID) | Yes | トークン固有 ID。重複防止用 |
| client_id | string | Yes | 認可したクライアントの client_id |
| scope | string | Yes | 許可されたスコープ（スペース区切り） |
| username | string | No | メンバー名（表示用。profile スコープ時のみ含む） |

### Client Credentials Grant のアクセストークン

メンバー不在のため `sub` にクライアント ID を使用し、`username` は含まない。

```json
{
  "iss": "https://oauth.example.internal",
  "sub": "my-service-app",
  "aud": "https://oauth.example.internal/api",
  "exp": 1719197300,
  "iat": 1719196400,
  "jti": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "client_id": "my-service-app",
  "scope": "read",
  "grant_type": "client_credentials"
}
```

| クレーム | 差分 |
|---|---|
| sub | メンバー UUID ではなく client_id |
| grant_type | `client_credentials` を明示 |
| username | 含まない |

---

## JWKS エンドポイントのレスポンス

`GET /oauth/jwks`

```json
{
  "keys": [
    {
      "kty": "EC",
      "use": "sig",
      "kid": "key-1",
      "alg": "ES256",
      "crv": "P-256",
      "x": "f83OJ3D2xF1Bg8vub9tLe1gHMzV76e8Tus9uPHvRVEU",
      "y": "x_FEzRu9m36HLN_tue659LNpXW6pCyStikYjKIWI5a0"
    }
  ]
}
```

| フィールド | 説明 |
|---|---|
| kty | 鍵の種類。EC（楕円曲線） |
| use | 用途。sig（署名） |
| kid | 鍵 ID。JWT ヘッダーの kid と一致 |
| alg | アルゴリズム。ES256 |
| crv | 曲線。P-256 |
| x, y | 公開鍵の座標（Base64url エンコード） |

---

## リソースサーバーでの検証手順

```
1. JWT ヘッダーの kid を取得
2. JWKS キャッシュから kid に一致する公開鍵を検索
   - 一致なし → JWKS を再取得してリトライ（鍵ローテーション対応）
3. ES256 で署名検証
4. クレームの検証:
   a. iss == AUTH_JWT_ISSUER（環境変数）
   b. aud == 期待する audience
   c. exp > 現在時刻（有効期限内）
   d. iat <= 現在時刻（未来のトークンを拒否）
5. scope からエンドポイントに必要なスコープがあるか確認
6. sub, client_id をリクエストコンテキストに格納
```

---

## 鍵ローテーション

### 手順

```
1. 新しい鍵ペアを生成（kid: key-2）
2. JWKS に新旧両方の鍵を公開
3. 認可サーバーの署名鍵を key-2 に切り替え
4. 旧トークンがすべて失効するまで待つ（最大15分）
5. JWKS から旧鍵（key-1）を削除
```

### 鍵生成コマンド（ES256）

```bash
# 秘密鍵の生成
openssl ecparam -genkey -name prime256v1 -noout -out private.pem

# 公開鍵の抽出
openssl ec -in private.pem -pubout -out public.pem
```

---

## トークンサイズの目安

ES256 のアクセストークン: **約 400〜500 bytes**（Base64url エンコード後）

```
eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6ImtleS0xIn0.
eyJpc3MiOiJodHRwczovL29hdXRoLmV4YW1wbGUuaW50ZXJuYWwiLCJz
dWIiOiI1NTBlODQwMC1lMjliLTQxZDQtYTcxNi00NDY2NTU0NDAwMDAi
LCJhdWQiOiJodHRwczovL29hdXRoLmV4YW1wbGUuaW50ZXJuYWwvYXBp
IiwiZXhwIjoxNzE5MTk3MzAwLCJpYXQiOjE3MTkxOTY0MDAsImp0aSI6
ImExYjJjM2Q0LWU1ZjYtNzg5MC1hYmNkLWVmMTIzNDU2Nzg5MCIsImNs
aWVudF9pZCI6Im15LWFwcCIsInNjb3BlIjoicmVhZCB3cml0ZSJ9.
<signature: 64 bytes>
```
