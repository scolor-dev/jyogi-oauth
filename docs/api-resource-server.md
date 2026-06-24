# リソースサーバー API 設計

## ベース URL

```
http://127.0.0.1:8081
Nginx 経由: https://oauth.example.internal/api/*
```

---

## 認証・認可（共通）

すべての保護エンドポイントで以下を適用する。

### 認証方式

```
Authorization: Bearer {access_token}
```

### 認可フロー（ミドルウェア）

```
1. Authorization ヘッダーから Bearer トークン抽出
2. JWT 署名検証（JWKS 公開鍵）
3. トークン有効期限チェック (exp, TTL: 15分)
4. スコープチェック（エンドポイントごとの要求スコープと照合）
5. リクエストコンテキストに member_id, client_id, scopes を格納
```

※ Redis 参照なし。JWT の署名と exp のみで検証完結。
失効制御はリフレッシュトークン側で行い、JWT は最大15分で自然失効する。

### JWKS キャッシュ戦略

```
- 起動時に認可サーバーの /oauth/jwks から公開鍵を取得
- メモリにキャッシュ（TTL: 1時間）
- 検証失敗時にキャッシュを更新（鍵ローテーション対応）
- 認可サーバーに到達できない場合はキャッシュで継続
```

### 共通エラーレスポンス

**401 Unauthorized（トークンなし / 無効）:**

```
HTTP/1.1 401 Unauthorized
WWW-Authenticate: Bearer realm="jyogi-oauth"

{
  "error": {
    "code": "unauthorized",
    "message": "Invalid or expired access token"
  }
}
```

**403 Forbidden（スコープ不足）:**

```json
HTTP/1.1 403 Forbidden

{
  "error": {
    "code": "insufficient_scope",
    "message": "Required scope: admin"
  }
}
```

**429 Too Many Requests:**

```json
HTTP/1.1 429 Too Many Requests
Retry-After: 30

{
  "error": {
    "code": "rate_limit_exceeded",
    "message": "Too many requests. Retry after 30 seconds"
  }
}
```

---

## ヘルスチェック

### GET /api/health

認証不要。監視・ロードバランサー用。

**成功レスポンス:**

```json
HTTP/1.1 200 OK

{
  "status": "healthy",
  "version": "0.1.0",
  "uptime_seconds": 86400,
  "dependencies": {
    "postgresql": "connected",
    "auth_server": "reachable"
  }
}
```

**異常時:**

```json
HTTP/1.1 503 Service Unavailable

{
  "status": "unhealthy",
  "dependencies": {
    "postgresql": "disconnected",
    "auth_server": "unreachable"
  }
}
```

---

## メンバー Identity API

メンバーの基本識別情報（表示名、アイコン、テーマカラー）の取得・更新。
クライアントシステムから最も高頻度でアクセスされるエンドポイント。

### GET /api/v1/members/me/identity

自分自身の Identity を取得する。

**必要スコープ:** `identity`

**成功レスポンス:**

```json
{
  "member_id": "550e8400-e29b-41d4-a716-446655440000",
  "display_name": "田中 太郎",
  "avatar_url": "https://cdn.example.internal/avatars/550e8400.png",
  "theme_color": "#4A90D9",
  "updated_at": "2026-06-24T09:00:00Z"
}
```

**404 Not Found（Identity 未作成）:**

```json
{
  "error": {
    "code": "identity_not_found",
    "message": "Identity has not been created yet"
  }
}
```

### PUT /api/v1/members/me/identity

自分自身の Identity を作成・更新する（Upsert）。

**必要スコープ:** `identity:write` or `write`

**リクエストボディ:**

```json
{
  "display_name": "田中 太郎",
  "avatar_url": "https://cdn.example.internal/avatars/550e8400.png",
  "theme_color": "#4A90D9"
}
```

**バリデーション:**

| フィールド | ルール |
|---|---|
| display_name | 必須、1〜100文字 |
| avatar_url | 任意、URL 形式、最大2048文字 |
| theme_color | 必須、HEX カラーコード（`#` + 6文字、例: `#FF5733`） |

**成功レスポンス（作成）:**

```json
HTTP/1.1 201 Created

{
  "member_id": "550e8400-e29b-41d4-a716-446655440000",
  "display_name": "田中 太郎",
  "avatar_url": "https://cdn.example.internal/avatars/550e8400.png",
  "theme_color": "#4A90D9",
  "updated_at": "2026-06-24T09:00:00Z"
}
```

**成功レスポンス（更新）:**

```json
HTTP/1.1 200 OK

{
  "member_id": "550e8400-e29b-41d4-a716-446655440000",
  "display_name": "田中 太郎（更新）",
  "avatar_url": "https://cdn.example.internal/avatars/550e8400.png",
  "theme_color": "#FF5733",
  "updated_at": "2026-06-24T10:00:00Z"
}
```

### GET /api/v1/members/{member_id}/identity

指定メンバーの Identity を取得する。他メンバーの表示名やアイコンの取得に使用。

**必要スコープ:** `identity`

**パスパラメータ:**

| パラメータ | 必須 | 説明 |
|---|---|---|
| member_id | Yes | 対象メンバーの UUID |

**成功レスポンス:** `GET /me/identity` と同一形式

### GET /api/v1/members/identities

複数メンバーの Identity を一括取得する。チャットのメンバーリスト等で使用。

**必要スコープ:** `identity`

**クエリパラメータ:**

| パラメータ | 必須 | 説明 |
|---|---|---|
| ids | Yes | カンマ区切りの member_id（最大50件） |

**リクエスト例:**

```
GET /api/v1/members/identities?ids=uuid-1,uuid-2,uuid-3
```

**成功レスポンス:**

```json
{
  "data": [
    {
      "member_id": "uuid-1",
      "display_name": "田中 太郎",
      "avatar_url": "https://cdn.example.internal/avatars/uuid-1.png",
      "theme_color": "#4A90D9"
    },
    {
      "member_id": "uuid-2",
      "display_name": "鈴木 花子",
      "avatar_url": null,
      "theme_color": "#E74C3C"
    }
  ]
}
```

※ 存在しない member_id は結果に含まない（エラーにしない）

---

## 共通レスポンスフォーマット

### 成功（単一リソース）

```json
{
  "id": "...",
  "field": "value"
}
```

### 成功（一覧）

```json
{
  "data": [...],
  "pagination": {
    "total": 100,
    "page": 1,
    "per_page": 20,
    "total_pages": 5
  }
}
```

### エラー

```json
{
  "error": {
    "code": "error_code",
    "message": "Human-readable message",
    "details": {}
  }
}
```

### HTTPステータスコード

| コード | 用途 |
|---|---|
| 200 | 取得・更新成功 |
| 201 | 作成成功 |
| 204 | 削除成功 |
| 400 | バリデーションエラー |
| 401 | 認証エラー |
| 403 | 認可エラー（スコープ不足） |
| 404 | リソースが見つからない |
| 409 | 競合（重複など） |
| 429 | Rate Limit 超過 |
| 500 | サーバーエラー |

---

## 共通ヘッダー

**レスポンス:**

| ヘッダー | 説明 |
|---|---|
| X-Request-ID | リクエスト追跡用 |
| X-RateLimit-Limit | Rate Limit 上限 |
| X-RateLimit-Remaining | 残りリクエスト数 |
| X-RateLimit-Reset | リセット時刻（Unix timestamp） |

---

## Rate Limit

| 対象 | 上限 | ウィンドウ |
|---|---|---|
| 認証済みメンバー | 100 req | 60秒 |
| クライアント単位 | 1000 req | 60秒 |

カウンタは Redis に保持。
