# jyogi-oauth 設計書

## 概要

部内向け OAuth 2.0 認可基盤。
認可サーバー・リソースサーバー・フロントエンドを1台の Linux サーバーに Docker Compose で構築し、将来的にサーバー分離が可能な設計とする。

---

## システム構成

### コンテナ一覧（Docker Compose）

| コンテナ | ポート | 言語/技術 | 役割 |
|---|---|---|---|
| nginx | :80 / :443 | Nginx | 静的ファイル配信、リバースプロキシ |
| auth-server | :8080 | Go | OAuth 2.0 認可エンドポイント |
| resource-server | :8081 | Rust | 保護された API の提供 |
| redis | :6379 | Redis | 一時データ（リフレッシュトークン、認可コード、セッション） |
| postgres | :5432 | PostgreSQL | 永続データ（メンバー、クライアント、権限） |

---

## アーキテクチャ図

```
                    クライアントアプリ
                         │
                         ▼
              ┌─────────────────────┐
              │  Nginx (:80/:443)   │
              │  静的配信 + Proxy    │
              └───┬───┬───┬────────┘
                  │   │   │
      ┌───────────┘   │   └───────────┐
      ▼               ▼               ▼
Vue SPA (静的)   Go 認可 (:8080)  Rust リソース (:8081)
/login           /oauth/*         /api/*
/consent
/admin
                 │    │           │
                 │    ▼           │
                 │  Redis         │
                 │  (:6379)       │
                 ▼                ▼
               PostgreSQL (:5432)
```

すべて Docker コンテナとして動作。コンテナ間は Docker ネットワークで通信。

---

## OAuth 2.0 フロー（Authorization Code + PKCE）

```
1.  クライアント → Nginx → Go:  GET /oauth/authorize
        ?response_type=code
        &client_id=xxx
        &redirect_uri=https://...
        &scope=read write
        &state=random
        &code_challenge=xxx
        &code_challenge_method=S256

2.  Go → ブラウザ: ログイン画面 (/login) へリダイレクト

3.  メンバー → Go: 認証情報を送信（ID/パスワード）

4.  Go → ブラウザ: 同意画面 (/consent) を表示

5.  メンバー → Go: 同意を送信

6.  Go → Redis: 認可コードを保存（TTL: 600秒）

7.  Go → クライアント: redirect_uri に認可コード付きでリダイレクト
        ?code=auth_code&state=random

8.  クライアント → Nginx → Go:  POST /oauth/token
        grant_type=authorization_code
        &code=auth_code
        &redirect_uri=https://...
        &client_id=xxx
        &code_verifier=xxx

9.  Go → Redis: 認可コードを検証・削除
    Go: PKCE 検証、JWT 署名（アクセストークン、TTL: 15分）
    Go → Redis: リフレッシュトークンを保存（TTL: 7日）

10. Go → クライアント: トークンレスポンス
        { access_token(JWT), refresh_token, token_type, expires_in: 900, scope }

11. クライアント → Nginx → Rust:  GET /api/resource
        Authorization: Bearer <access_token>

12. Rust: JWT 署名検証 + exp チェック（Redis参照なし）
    Rust → PostgreSQL: リソース取得

13. Rust → クライアント: リソースレスポンス
```

---

## データ設計

### Redis（一時データ）

| キー | 値 | TTL | 用途 |
|---|---|---|---|
| `auth:code:{認可コード}` | `{client_id, member_id, scope, code_challenge, redirect_uri}` | 600秒 | 認可コード |
| `auth:refresh:{hash}` | `{member_id, client_id, scope}` | 604800秒 | リフレッシュトークン |
| `auth:session:{id}` | `{member_id, oauth_params}` | 86400秒 | メンバーセッション |
| `auth:member_refreshes:{member_id}` | SET of refresh hashes | なし | メンバー別失効管理 |
| `ratelimit:*` | カウント | 60〜300秒 | Rate Limit |

※ アクセストークンは JWT（TTL: 15分）のため Redis に保存しない。
失効制御はリフレッシュトークンの削除で対応。

### PostgreSQL（永続データ）

#### members テーブル

| カラム | 型 | 説明 |
|---|---|---|
| id | UUID | 主キー |
| username | VARCHAR(255) | ログイン用ユーザー名 |
| password_hash | VARCHAR(255) | argon2id ハッシュ |
| email | VARCHAR(255) | メールアドレス |
| is_active | BOOLEAN | 有効/無効 |
| created_at | TIMESTAMP | 作成日時 |
| updated_at | TIMESTAMP | 更新日時 |

#### clients テーブル

| カラム | 型 | 説明 |
|---|---|---|
| id | UUID | 主キー |
| client_id | VARCHAR(255) | OAuth クライアント ID |
| client_secret_hash | VARCHAR(255) | シークレットのハッシュ |
| name | VARCHAR(255) | アプリ名 |
| redirect_uris | TEXT[] | 許可されたリダイレクト URI |
| allowed_scopes | TEXT[] | 許可されたスコープ |
| client_type | VARCHAR(50) | confidential / public |
| is_active | BOOLEAN | 有効/無効 |
| created_at | TIMESTAMP | 作成日時 |

#### scopes テーブル

| カラム | 型 | 説明 |
|---|---|---|
| id | UUID | 主キー |
| name | VARCHAR(100) | スコープ名（例: read, write, admin） |
| description | TEXT | スコープの説明 |

#### consent_records テーブル

| カラム | 型 | 説明 |
|---|---|---|
| id | UUID | 主キー |
| member_id | UUID | FK → members |
| client_id | UUID | FK → clients |
| scopes | TEXT[] | 同意済みスコープ |
| granted_at | TIMESTAMP | 同意日時 |

---

## 各サーバーの詳細

### Go 認可サーバー (:8080)

**エンドポイント:**

| メソッド | パス | 説明 |
|---|---|---|
| GET | /oauth/authorize | 認可リクエスト |
| POST | /oauth/token | トークン発行 |
| POST | /oauth/revoke | トークン失効 |
| GET | /oauth/jwks | 公開鍵 (JWKS) |
| POST | /oauth/introspect | トークンイントロスペクション |
| GET | /oauth/userinfo | メンバー情報（OAuth標準エンドポイント名） |

**依存:**
- Redis: リフレッシュトークン・セッション・認可コード管理
- PostgreSQL: メンバー認証、クライアント検証

### Rust リソースサーバー (:8081)

**エンドポイント:**

| メソッド | パス | 説明 |
|---|---|---|
| GET | /api/v1/* | 保護されたリソース API |
| GET | /api/health | ヘルスチェック |

**ミドルウェア:**
- JWT 検証（Go の /oauth/jwks から公開鍵取得、キャッシュ）
- スコープベースアクセス制御
- Rate Limiting

**依存:**
- PostgreSQL: 業務データ取得
- Go 認可サーバー: JWKS エンドポイント（公開鍵取得）

---

## フロントエンド（Vue SPA）

Nginx が静的ファイルとして配信。CSR（クライアントサイドレンダリング）。

### 画面一覧

| パス | 画面 | 説明 |
|---|---|---|
| /login | ログイン | メンバー認証 |
| /consent | 認可同意 | スコープ許可確認 |
| /admin | ダッシュボード | 管理トップ |
| /admin/clients | クライアント管理 | OAuth クライアントの CRUD |
| /admin/members | メンバー管理 | メンバーの CRUD |
| /admin/scopes | スコープ管理 | スコープの CRUD |
| /admin/tokens | トークン管理 | 発行済みトークンの確認・失効 |

---

## Nginx 設定

```nginx
server {
    listen 80;
    server_name oauth.example.internal;

    # Vue SPA 静的ファイル
    location / {
        root /var/www/jyogi-oauth/dist;
        try_files $uri $uri/ /index.html;
    }

    # 認可サーバー
    location /oauth/ {
        proxy_pass http://auth-server:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # リソースサーバー
    location /api/ {
        proxy_pass http://resource-server:8081;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

※ Docker Compose のサービス名で名前解決（`auth-server`, `resource-server`）

---

## Docker 構成

### docker-compose.yml

```yaml
services:
  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx/default.conf:/etc/nginx/conf.d/default.conf
      - ./web/dist:/var/www/jyogi-oauth/dist
    depends_on:
      - auth-server
      - resource-server

  auth-server:
    build: ./auth-server
    expose:
      - "8080"
    environment:
      - REDIS_URL=redis://redis:6379/0
      - DATABASE_URL=postgres://jyogi_auth:xxx@postgres:5432/jyogi_oauth?search_path=auth
    depends_on:
      - redis
      - postgres

  resource-server:
    build: ./resource-server
    expose:
      - "8081"
    environment:
      - DATABASE_URL=postgres://jyogi_resource:xxx@postgres:5432/jyogi_oauth?search_path=resource
      - JWKS_URL=http://auth-server:8080/oauth/jwks
    depends_on:
      - postgres
      - auth-server

  redis:
    image: redis:7-alpine
    volumes:
      - redis-data:/data
    command: redis-server --appendonly yes

  postgres:
    image: postgres:16-alpine
    environment:
      - POSTGRES_DB=jyogi_oauth
      - POSTGRES_USER=postgres
      - POSTGRES_PASSWORD=xxx
    volumes:
      - postgres-data:/var/lib/postgresql/data
      - ./migrations:/docker-entrypoint-initdb.d

volumes:
  redis-data:
  postgres-data:
```

### Dockerfile（認可サーバー）

```dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o auth-server ./cmd/server

FROM alpine:3.20
COPY --from=builder /app/auth-server /usr/local/bin/
EXPOSE 8080
CMD ["auth-server"]
```

### Dockerfile（リソースサーバー）

```dockerfile
FROM rust:1.80-alpine AS builder
WORKDIR /app
COPY Cargo.toml Cargo.lock ./
COPY src ./src
RUN cargo build --release

FROM alpine:3.20
COPY --from=builder /app/target/release/resource-server /usr/local/bin/
EXPOSE 8081
CMD ["resource-server"]
```

---

## 開発環境

- **開発マシン:** macOS
- **本番:** Linux 1台（Docker Compose）
- **ローカル開発:** 同じ docker-compose.yml を使用

### 開発コマンド

```bash
# 全サービス起動
docker compose up -d

# 認可サーバーのみ再ビルド
docker compose up -d --build auth-server

# ログ確認
docker compose logs -f auth-server

# 全サービス停止
docker compose down
```

### フロントエンド開発

```bash
cd web
npm install
npm run dev    # Vite dev server (:5173)
npm run build  # dist/ にビルド → Nginx が配信
```

---

## 将来の分離計画

### フェーズ 1（現在）

```
Linux 1台: docker-compose で全コンテナ稼働
  nginx / auth-server / resource-server / redis / postgres
```

### フェーズ 2（負荷増大時）

```
Linux A: nginx + web (Vue SPA)
Linux B: auth-server + redis
Linux C: resource-server + postgres
```

**分離時の変更点:**
- Nginx の proxy_pass をサービス名 → 各サーバーの IP に変更
- 環境変数で Redis / PostgreSQL の接続先を切り替え
- 各サーバーの docker-compose.yml を分割

コード変更は不要。設定変更のみで分離可能。

---

## プロジェクト構造

```
jyogi-oauth/
├── docs/                  # 設計ドキュメント
├── auth-server/           # Go - 認可サーバー
│   ├── Dockerfile
│   ├── cmd/server/        # エントリポイント
│   ├── internal/
│   │   ├── handler/       # HTTP ハンドラ
│   │   ├── oauth/         # OAuth ロジック
│   │   ├── middleware/     # 認証ミドルウェア
│   │   └── store/         # Redis / PostgreSQL アクセス
│   ├── go.mod
│   └── go.sum
├── resource-server/       # Rust - リソースサーバー
│   ├── Dockerfile
│   ├── src/
│   │   ├── main.rs        # エントリポイント
│   │   ├── api/           # API ハンドラ
│   │   ├── middleware/    # JWT 検証、スコープチェック
│   │   └── db/            # PostgreSQL アクセス
│   └── Cargo.toml
├── web/                   # Vue SPA - フロントエンド
│   ├── src/
│   │   ├── views/
│   │   │   ├── Login.vue
│   │   │   ├── Consent.vue
│   │   │   └── admin/
│   │   ├── router/
│   │   ├── stores/
│   │   └── App.vue
│   ├── package.json
│   └── vite.config.ts
├── migrations/            # PostgreSQL マイグレーション
├── nginx/                 # Nginx 設定
│   └── default.conf
├── config/                # 環境設定
│   ├── .env.dev
│   └── .env.prod
└── docker-compose.yml     # 統一構成
```
