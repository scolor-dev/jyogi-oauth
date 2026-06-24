# 環境変数一覧

## 概要

すべての設定は環境変数で管理する。
Docker Compose では `.env.dev` / `.env.prod` ファイルで切り替え。

---

## 認可サーバー（Go）

### サーバー設定

| 変数名 | 必須 | デフォルト | 説明 |
|---|---|---|---|
| `AUTH_SERVER_PORT` | No | `8080` | 認可サーバーのリッスンポート |
| `AUTH_SERVER_HOST` | No | `0.0.0.0` | バインドするホスト |
| `AUTH_LOG_LEVEL` | No | `info` | ログレベル（debug / info / warn / error） |

### データベース接続

| 変数名 | 必須 | デフォルト | 説明 |
|---|---|---|---|
| `AUTH_DATABASE_URL` | Yes | - | PostgreSQL 接続文字列 |
| `AUTH_DATABASE_MAX_CONNECTIONS` | No | `10` | コネクションプール最大数 |
| `AUTH_REDIS_URL` | Yes | - | Redis 接続文字列（例: `redis://redis:6379/0`） |

### JWT 署名

| 変数名 | 必須 | デフォルト | 説明 |
|---|---|---|---|
| `AUTH_JWT_ALGORITHM` | No | `ES256` | JWT 署名アルゴリズム |
| `AUTH_JWT_PRIVATE_KEY_PATH` | Yes | - | 秘密鍵ファイルのパス（PEM 形式） |
| `AUTH_JWT_PUBLIC_KEY_PATH` | Yes | - | 公開鍵ファイルのパス（PEM 形式） |
| `AUTH_JWT_KID` | No | `key-1` | JWT ヘッダーの kid（鍵ローテーション用） |
| `AUTH_JWT_ACCESS_TOKEN_TTL` | No | `900` | アクセストークンの有効期間（秒）。15分 |
| `AUTH_JWT_ISSUER` | No | `https://oauth.example.internal` | JWT の iss クレーム |

### リフレッシュトークン

| 変数名 | 必須 | デフォルト | 説明 |
|---|---|---|---|
| `AUTH_REFRESH_TOKEN_TTL` | No | `604800` | リフレッシュトークンの有効期間（秒）。7日 |
| `AUTH_REFRESH_TOKEN_LENGTH` | No | `64` | リフレッシュトークンのランダム文字列長（バイト） |

### セッション

| 変数名 | 必須 | デフォルト | 説明 |
|---|---|---|---|
| `AUTH_SESSION_TTL` | No | `86400` | セッションの有効期間（秒）。24時間 |
| `AUTH_SESSION_COOKIE_NAME` | No | `jyogi_sid` | セッション Cookie 名 |
| `AUTH_SESSION_COOKIE_SECURE` | No | `true` | Cookie の Secure フラグ（Cloudflare 経由なので true） |
| `AUTH_SESSION_COOKIE_DOMAIN` | No | - | Cookie のドメイン（例: `.example.internal`） |

### 認可コード

| 変数名 | 必須 | デフォルト | 説明 |
|---|---|---|---|
| `AUTH_CODE_TTL` | No | `600` | 認可コードの有効期間（秒）。10分 |
| `AUTH_CODE_LENGTH` | No | `32` | 認可コードのランダム文字列長（バイト） |

### パスワード

| 変数名 | 必須 | デフォルト | 説明 |
|---|---|---|---|
| `AUTH_PASSWORD_ALGORITHM` | No | `argon2id` | パスワードハッシュアルゴリズム |
| `AUTH_ARGON2_MEMORY` | No | `65536` | Argon2 メモリコスト（KB）。64MB |
| `AUTH_ARGON2_ITERATIONS` | No | `3` | Argon2 反復回数 |
| `AUTH_ARGON2_PARALLELISM` | No | `2` | Argon2 並列度 |

### Rate Limit

| 変数名 | 必須 | デフォルト | 説明 |
|---|---|---|---|
| `AUTH_RATE_LIMIT_LOGIN` | No | `5` | ログイン試行上限（5分間） |
| `AUTH_RATE_LIMIT_TOKEN` | No | `20` | /token エンドポイント上限（60秒間） |
| `AUTH_RATE_LIMIT_IP` | No | `30` | IP 単位の上限（60秒間） |

---

## リソースサーバー（Rust）

### サーバー設定

| 変数名 | 必須 | デフォルト | 説明 |
|---|---|---|---|
| `RESOURCE_SERVER_PORT` | No | `8081` | リソースサーバーのリッスンポート |
| `RESOURCE_SERVER_HOST` | No | `0.0.0.0` | バインドするホスト |
| `RESOURCE_LOG_LEVEL` | No | `info` | ログレベル（debug / info / warn / error） |

### データベース接続

| 変数名 | 必須 | デフォルト | 説明 |
|---|---|---|---|
| `RESOURCE_DATABASE_URL` | Yes | - | PostgreSQL 接続文字列 |
| `RESOURCE_DATABASE_MAX_CONNECTIONS` | No | `10` | コネクションプール最大数 |

### JWT 検証

| 変数名 | 必須 | デフォルト | 説明 |
|---|---|---|---|
| `RESOURCE_JWKS_URL` | Yes | - | 認可サーバーの JWKS エンドポイント URL |
| `RESOURCE_JWKS_CACHE_TTL` | No | `3600` | JWKS キャッシュ有効期間（秒） |
| `RESOURCE_JWT_ISSUER` | No | `https://oauth.example.internal` | 期待する JWT の iss クレーム |

### Rate Limit

| 変数名 | 必須 | デフォルト | 説明 |
|---|---|---|---|
| `RESOURCE_RATE_LIMIT_MEMBER` | No | `100` | メンバー単位の上限（60秒間） |
| `RESOURCE_RATE_LIMIT_IP` | No | `30` | IP 単位の上限（60秒間） |
| `RESOURCE_REDIS_URL` | No | - | Rate Limit 用 Redis（設定時のみ Redis 使用） |

---

## PostgreSQL

| 変数名 | 必須 | デフォルト | 説明 |
|---|---|---|---|
| `POSTGRES_DB` | Yes | - | データベース名 |
| `POSTGRES_USER` | Yes | - | 管理者ユーザー名 |
| `POSTGRES_PASSWORD` | Yes | - | 管理者パスワード |

---

## Redis

| 変数名 | 必須 | デフォルト | 説明 |
|---|---|---|---|
| `REDIS_MAXMEMORY` | No | `64mb` | メモリ上限 |
| `REDIS_MAXMEMORY_POLICY` | No | `volatile-ttl` | メモリ上限到達時のポリシー |

---

## 環境別設定ファイル例

### .env.dev

```env
# PostgreSQL
POSTGRES_DB=jyogi_oauth
POSTGRES_USER=postgres
POSTGRES_PASSWORD=dev_password

# 認可サーバー
AUTH_SERVER_PORT=8080
AUTH_DATABASE_URL=postgres://postgres:dev_password@postgres:5432/jyogi_oauth?search_path=auth
AUTH_REDIS_URL=redis://redis:6379/0
AUTH_JWT_PRIVATE_KEY_PATH=/keys/private.pem
AUTH_JWT_PUBLIC_KEY_PATH=/keys/public.pem
AUTH_JWT_ISSUER=http://localhost
AUTH_SESSION_COOKIE_SECURE=false
AUTH_LOG_LEVEL=debug

# リソースサーバー
RESOURCE_SERVER_PORT=8081
RESOURCE_DATABASE_URL=postgres://postgres:dev_password@postgres:5432/jyogi_oauth?search_path=resource
RESOURCE_JWKS_URL=http://auth-server:8080/oauth/jwks
RESOURCE_JWT_ISSUER=http://localhost
RESOURCE_LOG_LEVEL=debug
```

### .env.prod

```env
# PostgreSQL
POSTGRES_DB=jyogi_oauth
POSTGRES_USER=postgres
POSTGRES_PASSWORD=${SECURE_GENERATED_PASSWORD}

# 認可サーバー
AUTH_SERVER_PORT=8080
AUTH_DATABASE_URL=postgres://jyogi_auth:${AUTH_DB_PASSWORD}@postgres:5432/jyogi_oauth?search_path=auth
AUTH_REDIS_URL=redis://redis:6379/0
AUTH_JWT_PRIVATE_KEY_PATH=/keys/private.pem
AUTH_JWT_PUBLIC_KEY_PATH=/keys/public.pem
AUTH_JWT_ISSUER=https://oauth.example.internal
AUTH_SESSION_COOKIE_SECURE=true
AUTH_SESSION_COOKIE_DOMAIN=.example.internal
AUTH_LOG_LEVEL=info

# リソースサーバー
RESOURCE_SERVER_PORT=8081
RESOURCE_DATABASE_URL=postgres://jyogi_resource:${RESOURCE_DB_PASSWORD}@postgres:5432/jyogi_oauth?search_path=resource
RESOURCE_JWKS_URL=http://auth-server:8080/oauth/jwks
RESOURCE_JWT_ISSUER=https://oauth.example.internal
RESOURCE_LOG_LEVEL=info
```

※ `.env.prod` は `.gitignore` に追加し、Git にコミットしないこと。
