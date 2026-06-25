-- jyogi-oauth: Consolidated schema initialization

CREATE SCHEMA IF NOT EXISTS auth;
CREATE SCHEMA IF NOT EXISTS resource;

-- Utility function
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

------------------------------------------------------------------------
-- auth schema
------------------------------------------------------------------------

CREATE TABLE auth.members (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username      VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    email         VARCHAR(255) NOT NULL UNIQUE,
    role          VARCHAR(50)  NOT NULL DEFAULT 'member',
    is_active     BOOLEAN      NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT chk_role CHECK (role IN ('member', 'moderator', 'admin'))
);

CREATE INDEX idx_members_username  ON auth.members (username);
CREATE INDEX idx_members_email     ON auth.members (email);
CREATE INDEX idx_members_is_active ON auth.members (is_active);
CREATE INDEX idx_members_role      ON auth.members (role);

CREATE TRIGGER trg_members_updated_at
    BEFORE UPDATE ON auth.members
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TABLE auth.clients (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id           VARCHAR(255) NOT NULL UNIQUE,
    client_secret_hash  VARCHAR(255),
    name                VARCHAR(255) NOT NULL,
    description         TEXT,
    client_type         VARCHAR(50)  NOT NULL DEFAULT 'confidential',
    redirect_uris       TEXT[]       NOT NULL,
    allowed_grant_types TEXT[]       NOT NULL DEFAULT '{authorization_code}',
    is_active           BOOLEAN      NOT NULL DEFAULT true,
    created_by          UUID REFERENCES auth.members(id),
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT chk_client_type CHECK (client_type IN ('confidential', 'public'))
);

CREATE INDEX idx_clients_client_id ON auth.clients (client_id);
CREATE INDEX idx_clients_is_active ON auth.clients (is_active);

CREATE TRIGGER trg_clients_updated_at
    BEFORE UPDATE ON auth.clients
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TABLE auth.scopes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    is_default  BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_scopes_name ON auth.scopes (name);

CREATE TABLE auth.client_scopes (
    client_id UUID NOT NULL REFERENCES auth.clients(id) ON DELETE CASCADE,
    scope_id  UUID NOT NULL REFERENCES auth.scopes(id)  ON DELETE CASCADE,
    PRIMARY KEY (client_id, scope_id)
);

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

CREATE INDEX idx_audit_member_id  ON auth.audit_logs (member_id);
CREATE INDEX idx_audit_action     ON auth.audit_logs (action);
CREATE INDEX idx_audit_created_at ON auth.audit_logs (created_at);

CREATE TABLE auth.signing_keys (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kid         VARCHAR(100) NOT NULL UNIQUE,
    algorithm   VARCHAR(10)  NOT NULL DEFAULT 'ES256',
    public_key  TEXT NOT NULL,
    private_key TEXT NOT NULL,
    is_active   BOOLEAN     NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at  TIMESTAMPTZ,

    CONSTRAINT chk_algorithm CHECK (algorithm IN ('ES256', 'RS256'))
);

------------------------------------------------------------------------
-- resource schema
------------------------------------------------------------------------

CREATE TABLE resource.member_identities (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    member_id    UUID         NOT NULL UNIQUE,
    display_name VARCHAR(100) NOT NULL,
    avatar_url   TEXT,
    theme_color  VARCHAR(7)   NOT NULL DEFAULT '#000000',
    tagline      VARCHAR(8),
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_member_identities_member_id ON resource.member_identities (member_id);

CREATE TRIGGER trg_member_identities_updated_at
    BEFORE UPDATE ON resource.member_identities
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

------------------------------------------------------------------------
-- DB roles
------------------------------------------------------------------------

DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'jyogi_auth') THEN
        CREATE ROLE jyogi_auth WITH LOGIN PASSWORD 'auth_dev_password';
    END IF;
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'jyogi_resource') THEN
        CREATE ROLE jyogi_resource WITH LOGIN PASSWORD 'resource_dev_password';
    END IF;
END
$$;

GRANT USAGE ON SCHEMA auth TO jyogi_auth;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA auth TO jyogi_auth;

GRANT USAGE ON SCHEMA resource TO jyogi_resource;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA resource TO jyogi_resource;
GRANT USAGE ON SCHEMA auth TO jyogi_resource;
GRANT SELECT ON auth.members TO jyogi_resource;

------------------------------------------------------------------------
-- Seed data
------------------------------------------------------------------------

INSERT INTO auth.scopes (name, description, is_default) VALUES
    ('openid',         'OpenID Connect 認証',                                      true),
    ('read',           'リソースの読み取り',                                       true),
    ('write',          'リソースの書き込み',                                       false),
    ('admin',          '管理者権限',                                               false),
    ('identity',       'メンバーの基本識別情報の読み取り（表示名、アイコン、カラー、タグライン）', true),
    ('identity:write', 'メンバーの基本識別情報の更新',                              false),
    ('profile',        'メンバーの詳細プロフィール（name, preferred_username）',     false);

-- Root admin member (password: 65536)
INSERT INTO auth.members (username, password_hash, email, role) VALUES (
    'root',
    '$argon2id$v=19$m=65536,t=3,p=2$l+/v4N5mfUGIqcbQ1hXetg$VkEH3NL3qVVuLiVpHyx7bFbjeqdMIo0AWyKGIP7yiCo',
    'root@jyogi.internal',
    'admin'
);
