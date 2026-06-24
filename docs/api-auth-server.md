# 認可サーバー API 設計

## ベース URL

```
http://127.0.0.1:8080
Nginx 経由: https://oauth.example.internal/oauth/*
```

---

## 認可エンドポイント

### GET /oauth/authorize

認可コードの発行を開始する。メンバーをログイン→同意画面へ誘導。

**リクエスト（クエリパラメータ）:**

| パラメータ | 必須 | 説明 |
|---|---|---|
| response_type | Yes | `code` 固定 |
| client_id | Yes | OAuth クライアント ID |
| redirect_uri | Yes | コールバック URI（クライアント登録済みのもの） |
| scope | Yes | 要求するスコープ（スペース区切り） |
| state | Yes | CSRF 対策用ランダム文字列 |
| code_challenge | Yes | PKCE チャレンジ（S256） |
| code_challenge_method | Yes | `S256` 固定 |

**処理フロー:**

```
1. client_id, redirect_uri の検証
2. 未ログイン → /login へリダイレクト（パラメータを session に保存）
3. ログイン済み → 同意済みか確認
4. 未同意 → /consent へリダイレクト
5. 同意済み → 認可コード発行 → redirect_uri へリダイレクト
```

**成功レスポンス（リダイレクト）:**

```
HTTP/1.1 302 Found
Location: {redirect_uri}?code={auth_code}&state={state}
```

**エラーレスポンス（リダイレクト）:**

```
HTTP/1.1 302 Found
Location: {redirect_uri}?error={error_code}&error_description={description}&state={state}
```

| error_code | 説明 |
|---|---|
| invalid_request | パラメータ不足・不正 |
| unauthorized_client | クライアントに権限がない |
| access_denied | メンバーが同意を拒否 |
| unsupported_response_type | response_type が未対応 |
| invalid_scope | 無効なスコープ |
| server_error | サーバーエラー |

---

## トークンエンドポイント

### POST /oauth/token

認可コードをアクセストークンに交換する。リフレッシュトークンによる再発行にも使用。

**Content-Type:** `application/x-www-form-urlencoded`

#### Grant Type: authorization_code

**リクエストボディ:**

| パラメータ | 必須 | 説明 |
|---|---|---|
| grant_type | Yes | `authorization_code` |
| code | Yes | 認可コード |
| redirect_uri | Yes | /authorize 時と同一の URI |
| client_id | Yes | クライアント ID |
| client_secret | Confidential のみ | クライアントシークレット |
| code_verifier | Yes | PKCE 検証用の元文字列 |

**成功レスポンス:**

```json
HTTP/1.1 200 OK
Content-Type: application/json
Cache-Control: no-store

{
  "access_token": "eyJhbGciOiJSUzI1NiIs...",
  "token_type": "Bearer",
  "expires_in": 900,
  "refresh_token": "dGhpcyBpcyBhIHJlZnJlc2g...",
  "scope": "read write"
}
```

#### Grant Type: refresh_token

**リクエストボディ:**

| パラメータ | 必須 | 説明 |
|---|---|---|
| grant_type | Yes | `refresh_token` |
| refresh_token | Yes | リフレッシュトークン |
| client_id | Yes | クライアント ID |
| client_secret | Confidential のみ | クライアントシークレット |
| scope | No | スコープの縮小（元のスコープ以下） |

**成功レスポンス:** authorization_code と同一形式

#### Grant Type: client_credentials

**リクエストボディ:**

| パラメータ | 必須 | 説明 |
|---|---|---|
| grant_type | Yes | `client_credentials` |
| client_id | Yes | クライアント ID |
| client_secret | Yes | クライアントシークレット |
| scope | No | 要求するスコープ |

**成功レスポンス:**

```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIs...",
  "token_type": "Bearer",
  "expires_in": 900,
  "scope": "read"
}
```

※ refresh_token は発行しない

**エラーレスポンス（共通）:**

```json
HTTP/1.1 400 Bad Request
Content-Type: application/json

{
  "error": "invalid_grant",
  "error_description": "Authorization code has expired"
}
```

| error | 説明 |
|---|---|
| invalid_request | パラメータ不足・不正 |
| invalid_client | クライアント認証失敗 |
| invalid_grant | 認可コード / リフレッシュトークンが無効 |
| unauthorized_client | このクライアントには許可されていない grant_type |
| unsupported_grant_type | 未対応の grant_type |
| invalid_scope | 無効なスコープ |

---

## トークン失効

### POST /oauth/revoke

発行済みトークンを無効化する。

**リクエストボディ:**

| パラメータ | 必須 | 説明 |
|---|---|---|
| token | Yes | 失効させるトークン |
| token_type_hint | No | `access_token` or `refresh_token` |
| client_id | Yes | クライアント ID |
| client_secret | Confidential のみ | クライアントシークレット |

**成功レスポンス:**

```
HTTP/1.1 200 OK
```

※ トークンが存在しない場合も 200 を返す（RFC 7009）

---

## トークンイントロスペクション

### POST /oauth/introspect

トークンの有効性と付随情報を返す。リソースサーバーが利用。

**リクエストボディ:**

| パラメータ | 必須 | 説明 |
|---|---|---|
| token | Yes | 検証対象のトークン |
| token_type_hint | No | `access_token` or `refresh_token` |

**認証:** リソースサーバーの認証情報（Basic 認証 or Bearer トークン）

**成功レスポンス（有効）:**

```json
{
  "active": true,
  "scope": "read write",
  "client_id": "my-app",
  "username": "tanaka",
  "token_type": "Bearer",
  "exp": 1719200000,
  "iat": 1719196400,
  "sub": "550e8400-e29b-41d4-a716-446655440000",
  "iss": "https://oauth.example.internal"
}
```

**成功レスポンス（無効）:**

```json
{
  "active": false
}
```

---

## JWKS エンドポイント

### GET /oauth/jwks

JWT 検証用の公開鍵を JSON Web Key Set 形式で返す。

**成功レスポンス:**

```json
{
  "keys": [
    {
      "kty": "RSA",
      "use": "sig",
      "kid": "key-2024-01",
      "alg": "RS256",
      "n": "0vx7agoebGcQ...",
      "e": "AQAB"
    }
  ]
}
```

**キャッシュ:** `Cache-Control: public, max-age=3600`

---

## メンバー情報

### GET /oauth/userinfo

アクセストークンに紐づくメンバー情報を返す。

**認証:** `Authorization: Bearer {access_token}`

**必要スコープ:** `openid`（または `profile`）

**成功レスポンス:**

```json
{
  "sub": "550e8400-e29b-41d4-a716-446655440000",
  "username": "tanaka",
  "email": "tanaka@example.internal",
  "name": "田中 太郎"
}
```

**エラーレスポンス:**

```
HTTP/1.1 401 Unauthorized
WWW-Authenticate: Bearer error="invalid_token"
```

---

## 認証 API（フロントエンド向け）

### POST /oauth/login

ログインフォームからの認証リクエストを処理する。

**リクエストボディ（JSON）:**

```json
{
  "username": "tanaka",
  "password": "secure_password"
}
```

**成功レスポンス:**

```json
HTTP/1.1 200 OK

{
  "redirect_to": "/oauth/authorize?..."
}
```

セッション Cookie を発行し、元の認可フローに戻す。

**エラーレスポンス:**

```json
HTTP/1.1 401 Unauthorized

{
  "error": "invalid_credentials",
  "error_description": "Username or password is incorrect"
}
```

### POST /oauth/consent

同意フォームからのリクエストを処理する。

**リクエストボディ（JSON）:**

```json
{
  "consent_id": "session-stored-id",
  "approved": true,
  "scopes": ["read", "write"]
}
```

**成功レスポンス:**

```json
{
  "redirect_to": "https://client-app.example/callback?code=xxx&state=yyy"
}
```

---

## 管理 API（管理画面向け）

認証: 管理者権限のセッション Cookie または Bearer トークン（scope: `admin`）

### クライアント管理

#### GET /oauth/admin/clients

クライアント一覧を取得する。

**クエリパラメータ:**

| パラメータ | 必須 | 説明 |
|---|---|---|
| page | No | ページ番号（デフォルト: 1） |
| per_page | No | 件数（デフォルト: 20, 最大: 100） |

**成功レスポンス:**

```json
{
  "clients": [
    {
      "id": "550e8400-...",
      "client_id": "my-app",
      "name": "My Application",
      "client_type": "confidential",
      "redirect_uris": ["https://my-app.example/callback"],
      "allowed_scopes": ["read", "write"],
      "is_active": true,
      "created_at": "2026-01-15T09:00:00Z"
    }
  ],
  "total": 5,
  "page": 1,
  "per_page": 20
}
```

#### POST /oauth/admin/clients

クライアントを新規登録する。

**リクエストボディ:**

```json
{
  "name": "My Application",
  "client_type": "confidential",
  "redirect_uris": ["https://my-app.example/callback"],
  "allowed_scopes": ["read", "write"]
}
```

**成功レスポンス:**

```json
HTTP/1.1 201 Created

{
  "id": "550e8400-...",
  "client_id": "generated-client-id",
  "client_secret": "generated-secret-shown-only-once",
  "name": "My Application",
  "client_type": "confidential",
  "redirect_uris": ["https://my-app.example/callback"],
  "allowed_scopes": ["read", "write"],
  "is_active": true,
  "created_at": "2026-06-24T09:00:00Z"
}
```

※ `client_secret` はこのレスポンスでのみ平文表示。以降はハッシュ保存。

#### GET /oauth/admin/clients/{id}

クライアント詳細を取得する。

#### PUT /oauth/admin/clients/{id}

クライアント情報を更新する。

**リクエストボディ:**

```json
{
  "name": "Updated App Name",
  "redirect_uris": ["https://my-app.example/callback", "https://my-app.example/callback2"],
  "allowed_scopes": ["read", "write", "admin"],
  "is_active": true
}
```

#### DELETE /oauth/admin/clients/{id}

クライアントを削除する（論理削除: is_active = false）。

### メンバー管理

#### GET /oauth/admin/members

メンバー一覧を取得する。

**クエリパラメータ:** `page`, `per_page`（クライアント管理と同様）

#### POST /oauth/admin/members

メンバーを新規登録する。

**リクエストボディ:**

```json
{
  "username": "tanaka",
  "password": "initial_password",
  "email": "tanaka@example.internal",
  "name": "田中 太郎"
}
```

#### GET /oauth/admin/members/{id}

#### PUT /oauth/admin/members/{id}

#### DELETE /oauth/admin/members/{id}

### スコープ管理

#### GET /oauth/admin/scopes

#### POST /oauth/admin/scopes

**リクエストボディ:**

```json
{
  "name": "read",
  "description": "リソースの読み取り権限"
}
```

#### PUT /oauth/admin/scopes/{id}

#### DELETE /oauth/admin/scopes/{id}

### トークン管理

#### GET /oauth/admin/tokens

発行済みトークン一覧を取得する。

**クエリパラメータ:**

| パラメータ | 必須 | 説明 |
|---|---|---|
| member_id | No | メンバーで絞り込み |
| client_id | No | クライアントで絞り込み |
| page | No | ページ番号 |

**成功レスポンス:**

```json
{
  "tokens": [
    {
      "token_id": "hash-of-token",
      "member_id": "550e8400-...",
      "username": "tanaka",
      "client_id": "my-app",
      "scope": "read write",
      "issued_at": "2026-06-24T09:00:00Z",
      "expires_at": "2026-06-24T10:00:00Z"
    }
  ],
  "total": 12,
  "page": 1,
  "per_page": 20
}
```

#### DELETE /oauth/admin/tokens/{token_id}

指定トークンを失効させる。

---

## 共通エラーフォーマット

OAuth エンドポイント以外の API エラー:

```json
{
  "error": {
    "code": "not_found",
    "message": "Resource not found"
  }
}
```

## 共通ヘッダー

**リクエスト:**

| ヘッダー | 説明 |
|---|---|
| Content-Type | `application/json` (管理API) / `application/x-www-form-urlencoded` (OAuth) |
| Authorization | `Bearer {token}` (保護エンドポイント) |
| X-Request-ID | リクエスト追跡用（任意） |

**レスポンス:**

| ヘッダー | 説明 |
|---|---|
| X-Request-ID | リクエスト追跡用（エコーバック） |
| X-RateLimit-Limit | Rate Limit 上限 |
| X-RateLimit-Remaining | 残りリクエスト数 |
| X-RateLimit-Reset | リセット時刻（Unix timestamp） |
