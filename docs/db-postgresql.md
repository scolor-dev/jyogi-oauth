# PostgreSQL データ設計

## 接続情報

```
Host: 127.0.0.1
Port: 5432
Database: jyogi_oauth
```

## スキーマ分離

| スキーマ | 用途 | アクセス元 |
|---|---|---|
| `auth` | 認可サーバー用（メンバー認証、クライアント、スコープ、同意） | Go 認可サーバー |
| `resource` | リソースサーバー用（プロフィール、業務データ） | Rust リソースサーバー |

将来の DB 分離時にスキーマ単位で移行可能。

---

## auth スキーマ（認可サーバー用）

### auth.members

メンバー情報。認証・認可の主体。

```sql
CREATE TABLE auth.members (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username      VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    email         VARCHAR(255) NOT NULL UNIQUE,
    is_active     BOOLEAN NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_members_username ON auth.members (username);
CREATE INDEX idx_members_email ON auth.members (email);
CREATE INDEX idx_members_is_active ON auth.members (is_active);
```

| カラム | 型 | 制約 | 説明 |
|---|---|---|---|
| id | UUID | PK | 主キー |
| username | VARCHAR(255) | UNIQUE, NOT NULL | ログイン用ユーザー名 |
| password_hash | VARCHAR(255) | NOT NULL | argon2id ハッシュ |
| email | VARCHAR(255) | UNIQUE, NOT NULL | メールアドレス |
| is_active | BOOLEAN | NOT NULL, DEFAULT true | 有効/無効 |
| created_at | TIMESTAMPTZ | NOT NULL | 作成日時 |
| updated_at | TIMESTAMPTZ | NOT NULL | 更新日時 |

※ 表示名（display_name）は `resource.member_identities` に持たせる。
auth スキーマは認証に必要な最小限の情報のみ保持する。

---

### auth.clients

OAuth クライアント。認可を要求するアプリケーション。

```sql
CREATE TABLE auth.clients (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id          VARCHAR(255) NOT NULL UNIQUE,
    client_secret_hash VARCHAR(255),
    name               VARCHAR(255) NOT NULL,
    description        TEXT,
    client_type        VARCHAR(50) NOT NULL DEFAULT 'confidential',
    redirect_uris      TEXT[] NOT NULL,
    allowed_grant_types TEXT[] NOT NULL DEFAULT '{authorization_code}',
    is_active          BOOLEAN NOT NULL DEFAULT true,
    created_by         UUID REFERENCES auth.members(id),
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT chk_client_type CHECK (client_type IN ('confidential', 'public'))
);

CREATE INDEX idx_clients_client_id ON auth.clients (client_id);
CREATE INDEX idx_clients_is_active ON auth.clients (is_active);
```

| カラム | 型 | 制約 | 説明 |
|---|---|---|---|
| id | UUID | PK | 内部 ID |
| client_id | VARCHAR(255) | UNIQUE, NOT NULL | OAuth クライアント ID（外部公開） |
| client_secret_hash | VARCHAR(255) | | シークレットのハッシュ（public の場合 NULL） |
| name | VARCHAR(255) | NOT NULL | アプリケーション名 |
| description | TEXT | | アプリケーション説明 |
| client_type | VARCHAR(50) | NOT NULL | `confidential` or `public` |
| redirect_uris | TEXT[] | NOT NULL | 許可されたリダイレクト URI 一覧 |
| allowed_grant_types | TEXT[] | NOT NULL | 許可された grant type 一覧 |
| is_active | BOOLEAN | NOT NULL | 有効/無効 |
| created_by | UUID | FK → members | 登録者 |
| created_at | TIMESTAMPTZ | NOT NULL | 作成日時 |
| updated_at | TIMESTAMPTZ | NOT NULL | 更新日時 |

---

### auth.scopes

スコープ定義。リソースへのアクセス権限の単位。

```sql
CREATE TABLE auth.scopes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    is_default  BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_scopes_name ON auth.scopes (name);
```

| カラム | 型 | 制約 | 説明 |
|---|---|---|---|
| id | UUID | PK | 主キー |
| name | VARCHAR(100) | UNIQUE, NOT NULL | スコープ名 |
| description | TEXT | | スコープの説明 |
| is_default | BOOLEAN | NOT NULL | デフォルトで付与するか |
| created_at | TIMESTAMPTZ | NOT NULL | 作成日時 |

**初期データ:**

```sql
INSERT INTO auth.scopes (name, description, is_default) VALUES
    ('read',             'リソースの読み取り', true),
    ('write',            'リソースの書き込み', false),
    ('admin',            '管理者権限', false),
    ('identity',         'メンバーの基本識別情報の読み取り（表示名、アイコン、カラー）', true),
    ('identity:write',   'メンバーの基本識別情報の更新', false),
    ('profile',          'メンバーの詳細プロフィールの読み取り（予定）', false);
```

---

### auth.client_scopes

クライアントに許可されたスコープ（多対多）。

```sql
CREATE TABLE auth.client_scopes (
    client_id UUID NOT NULL REFERENCES auth.clients(id) ON DELETE CASCADE,
    scope_id  UUID NOT NULL REFERENCES auth.scopes(id) ON DELETE CASCADE,
    PRIMARY KEY (client_id, scope_id)
);
```

---

### auth.consent_records

メンバーが同意したクライアント×スコープの記録。
再同意を省略するために使用。

```sql
CREATE TABLE auth.consent_records (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    member_id  UUID NOT NULL REFERENCES auth.members(id) ON DELETE CASCADE,
    client_id  UUID NOT NULL REFERENCES auth.clients(id) ON DELETE CASCADE,
    scopes     TEXT[] NOT NULL,
    granted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at TIMESTAMPTZ,

    CONSTRAINT uq_consent_member_client UNIQUE (member_id, client_id)
);

CREATE INDEX idx_consent_member_id ON auth.consent_records (member_id);
CREATE INDEX idx_consent_client_id ON auth.consent_records (client_id);
```

| カラム | 型 | 制約 | 説明 |
|---|---|---|---|
| id | UUID | PK | 主キー |
| member_id | UUID | FK → members, NOT NULL | 同意したメンバー |
| client_id | UUID | FK → clients, NOT NULL | 同意先クライアント |
| scopes | TEXT[] | NOT NULL | 同意したスコープ一覧 |
| granted_at | TIMESTAMPTZ | NOT NULL | 同意日時 |
| revoked_at | TIMESTAMPTZ | | 同意撤回日時（NULL = 有効） |

---

### auth.audit_logs

認可サーバーの操作ログ。セキュリティ監査用。

```sql
CREATE TABLE auth.audit_logs (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    member_id  UUID REFERENCES auth.members(id),
    client_id  VARCHAR(255),
    action     VARCHAR(100) NOT NULL,
    ip_address INET,
    user_agent TEXT,
    details    JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_member_id ON auth.audit_logs (member_id);
CREATE INDEX idx_audit_action ON auth.audit_logs (action);
CREATE INDEX idx_audit_created_at ON auth.audit_logs (created_at);
```

| カラム | 型 | 説明 |
|---|---|---|
| id | UUID | 主キー |
| member_id | UUID | 操作メンバー（未認証の場合 NULL） |
| client_id | VARCHAR(255) | 関連クライアント ID |
| action | VARCHAR(100) | 操作種別 |
| ip_address | INET | 接続元 IP |
| user_agent | TEXT | User-Agent |
| details | JSONB | 追加情報（自由形式） |
| created_at | TIMESTAMPTZ | 発生日時 |

**action の例:**

| action | 説明 |
|---|---|
| `login_success` | ログイン成功 |
| `login_failure` | ログイン失敗 |
| `consent_granted` | 同意付与 |
| `consent_revoked` | 同意撤回 |
| `token_issued` | トークン発行 |
| `token_revoked` | トークン失効 |
| `client_created` | クライアント作成 |
| `client_updated` | クライアント更新 |
| `client_deleted` | クライアント削除 |
| `member_created` | メンバー作成 |
| `member_updated` | メンバー更新 |
| `member_deactivated` | メンバー無効化 |

---

### auth.signing_keys

JWT 署名鍵の管理。鍵ローテーションに対応。

```sql
CREATE TABLE auth.signing_keys (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kid         VARCHAR(100) NOT NULL UNIQUE,
    algorithm   VARCHAR(10) NOT NULL DEFAULT 'ES256',
    public_key  TEXT NOT NULL,
    private_key TEXT NOT NULL,
    is_active   BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at  TIMESTAMPTZ,

    CONSTRAINT chk_algorithm CHECK (algorithm IN ('ES256', 'RS256'))
);
```

| カラム | 型 | 説明 |
|---|---|---|
| id | UUID | 主キー |
| kid | VARCHAR(100) | Key ID（JWT ヘッダーの kid に対応） |
| algorithm | VARCHAR(10) | 署名アルゴリズム（デフォルト: ES256） |
| public_key | TEXT | PEM 形式の公開鍵 |
| private_key | TEXT | PEM 形式の秘密鍵（暗号化して保存） |
| is_active | BOOLEAN | 現在使用中か |
| created_at | TIMESTAMPTZ | 作成日時 |
| expires_at | TIMESTAMPTZ | 有効期限 |

---

## resource スキーマ（リソースサーバー用）

### resource.member_identities

メンバーの基本識別情報。クライアントシステムから高頻度でアクセスされる。
表示名・アイコン・テーマカラーの最小構成。

```sql
CREATE TABLE resource.member_identities (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    member_id    UUID NOT NULL UNIQUE,
    display_name VARCHAR(100) NOT NULL,
    avatar_url   TEXT,
    theme_color  VARCHAR(7) NOT NULL DEFAULT '#000000',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_member_identities_member_id ON resource.member_identities (member_id);
```

| カラム | 型 | 制約 | 説明 |
|---|---|---|---|
| id | UUID | PK | 主キー |
| member_id | UUID | UNIQUE, NOT NULL | auth.members.id を参照（FK 制約なし） |
| display_name | VARCHAR(100) | NOT NULL | 表示名 |
| avatar_url | TEXT | | アイコン画像の URL |
| theme_color | VARCHAR(7) | NOT NULL, DEFAULT '#000000' | テーマカラー（HEXコード） |
| created_at | TIMESTAMPTZ | NOT NULL | 作成日時 |
| updated_at | TIMESTAMPTZ | NOT NULL | 更新日時 |

※ `member_id` は FK 制約を張らない（スキーマ分離・将来の DB 分離を考慮）。
アプリケーション層で整合性を担保する。

### resource.member_profiles（予定）

メンバーの詳細プロフィール。MVP 後に実装予定。

```
想定カラム: 誕生日、学年、学歴、学部/学科、入部年度、SNSリンク、スキル等
```

---

## ER 図

```
auth スキーマ:

  ┌───────────┐       ┌───────────────┐       ┌──────────┐
  │  members  │       │ consent_records│       │  scopes  │
  │───────────│       │───────────────│       │──────────│
  │ id (PK)   │◄──┐   │ id (PK)       │   ┌──►│ id (PK)  │
  │ username  │   │   │ member_id(FK) │───┘   │ name     │
  │ password  │   │   │ client_id(FK) │───┐   │ is_default│
  │ email     │   │   │ scopes        │   │   └──────────┘
  │ is_active │   │   │ granted_at    │   │        ▲
  └───────────┘   │   └───────────────┘   │        │
       ▲          │                       │   ┌────┴───────┐
       │          │   ┌──────────┐        │   │client_scopes│
       │          │   │ clients  │        │   │────────────│
       │          │   │──────────│        │   │client_id(FK)│
       │          └───│created_by│◄───────┘   │scope_id(FK)│
       │              │ id (PK)  │◄───────────┤            │
       │              │client_id │            └────────────┘
       │              │ name     │
       │              │ type     │
       │              │redirects │
       │              └──────────┘
       │                   ▲
       │                   │
       │           ┌───────┴──────┐
       │           │  audit_logs  │
       │           │──────────────│
       │           │ id (PK)      │
       │           │ member_id(FK)│
       │           │ client_id    │
       │           │ action       │
       │           └──────────────┘
       │
       │          ┌──────────────┐
       │          │ signing_keys │
       │          │──────────────│
       │          │ id (PK)      │
       │          │ kid          │
       │          │ public_key   │
       │          │ private_key  │
       │          └──────────────┘


resource スキーマ:

  ┌──────────────────────┐
  │  member_identities   │
  │──────────────────────│
  │ id (PK)              │
  │ member_id (UNIQUE)   │ ← auth.members.id を参照（FK なし）
  │ display_name         │
  │ avatar_url           │
  │ theme_color          │
  └──────────────────────┘

  ┌──────────────────────┐
  │  member_profiles     │ ← 予定
  │──────────────────────│
  │ member_id            │
  │ birthday, grade, ... │
  └──────────────────────┘
```

---

## マイグレーション

```
migrations/
├── 001_create_schemas.sql
├── 002_create_members.sql
├── 003_create_clients.sql
├── 004_create_scopes.sql
├── 005_create_client_scopes.sql
├── 006_create_consent_records.sql
├── 007_create_audit_logs.sql
├── 008_create_signing_keys.sql
└── 009_create_member_identities.sql
```

### 001_create_schemas.sql

```sql
CREATE SCHEMA IF NOT EXISTS auth;
CREATE SCHEMA IF NOT EXISTS resource;
```

---

## トリガー

### updated_at 自動更新

```sql
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_members_updated_at
    BEFORE UPDATE ON auth.members
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trg_clients_updated_at
    BEFORE UPDATE ON auth.clients
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trg_member_identities_updated_at
    BEFORE UPDATE ON resource.member_identities
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();
```

---

## DB ユーザー（ロール）

```sql
-- 認可サーバー用（Go）
CREATE ROLE jyogi_auth WITH LOGIN PASSWORD 'xxx';
GRANT USAGE ON SCHEMA auth TO jyogi_auth;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA auth TO jyogi_auth;

-- リソースサーバー用（Rust）
CREATE ROLE jyogi_resource WITH LOGIN PASSWORD 'xxx';
GRANT USAGE ON SCHEMA resource TO jyogi_resource;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA resource TO jyogi_resource;
GRANT USAGE ON SCHEMA auth TO jyogi_resource;
GRANT SELECT ON auth.members TO jyogi_resource;
```

リソースサーバーは `auth.members` を読み取り専用で参照可能（メンバー情報の表示用）。
