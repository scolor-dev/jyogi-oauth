# セキュリティ設計

## 概要

部内 OAuth 基盤のセキュリティ方針。
外部公開は Cloudflare Tunnel 経由で行い、HTTPS 終端は Cloudflare が担当する。

---

## 通信の暗号化

### 構成

```
クライアント →(HTTPS)→ Cloudflare →(Tunnel/暗号化)→ Nginx (:80 HTTP)
```

| 区間 | プロトコル | 暗号化 |
|---|---|---|
| クライアント → Cloudflare | HTTPS | Cloudflare が TLS 終端 |
| Cloudflare → Nginx | Cloudflare Tunnel | トンネル内で暗号化 |
| Nginx → Go / Rust | HTTP | Docker 内部ネットワーク（外部非公開） |
| Go / Rust → Redis / PostgreSQL | TCP | Docker 内部ネットワーク（外部非公開） |

### Nginx の設定

- Nginx は **HTTP (:80) のみ** でリッスン
- 証明書管理は不要（Cloudflare が担当）
- Cloudflare が付与するヘッダーを信頼する設定を追加

```nginx
# Cloudflare の実 IP を取得
set_real_ip_from 173.245.48.0/20;
set_real_ip_from 103.21.244.0/22;
set_real_ip_from 103.22.200.0/22;
set_real_ip_from 103.31.4.0/22;
set_real_ip_from 141.101.64.0/18;
set_real_ip_from 108.162.192.0/18;
set_real_ip_from 190.93.240.0/20;
set_real_ip_from 188.114.96.0/20;
set_real_ip_from 197.234.240.0/22;
set_real_ip_from 198.41.128.0/17;
set_real_ip_from 162.158.0.0/15;
set_real_ip_from 104.16.0.0/13;
set_real_ip_from 104.24.0.0/14;
set_real_ip_from 172.64.0.0/13;
set_real_ip_from 131.0.72.0/22;
real_ip_header CF-Connecting-IP;
```

---

## CORS（Cross-Origin Resource Sharing）

複数サブドメインからのリクエストを許可する。

### 設定方針

```nginx
# Nginx で CORS ヘッダーを付与
location /oauth/ {
    # 許可するオリジンのパターン
    set $cors_origin "";
    if ($http_origin ~* "^https://([a-z0-9-]+\.)?example\.internal$") {
        set $cors_origin $http_origin;
    }

    add_header Access-Control-Allow-Origin $cors_origin always;
    add_header Access-Control-Allow-Methods "GET, POST, PUT, DELETE, OPTIONS" always;
    add_header Access-Control-Allow-Headers "Authorization, Content-Type, X-Request-ID" always;
    add_header Access-Control-Allow-Credentials "true" always;
    add_header Access-Control-Max-Age "86400" always;

    # プリフライトリクエスト
    if ($request_method = OPTIONS) {
        return 204;
    }

    proxy_pass http://auth-server:8080;
}
```

| 設定 | 値 | 説明 |
|---|---|---|
| Allow-Origin | `*.example.internal` パターン | サブドメインを動的に許可 |
| Allow-Credentials | `true` | Cookie（セッション）の送信を許可 |
| Allow-Methods | GET, POST, PUT, DELETE, OPTIONS | 許可する HTTP メソッド |
| Allow-Headers | Authorization, Content-Type, X-Request-ID | 許可するカスタムヘッダー |
| Max-Age | 86400 | プリフライトキャッシュ（24時間） |

---

## CSRF 対策

### OAuth エンドポイント

| エンドポイント | 対策 |
|---|---|
| /oauth/authorize | `state` パラメータで CSRF 防止（OAuth 2.0 仕様） |
| /oauth/token | クライアント認証（client_secret）で保護 |
| /oauth/login | SameSite Cookie + Origin ヘッダー検証 |
| /oauth/consent | セッション Cookie + CSRF トークン |

### セッション Cookie

```
Set-Cookie: jyogi_sid={session_id};
    HttpOnly;
    Secure;
    SameSite=Lax;
    Path=/oauth;
    Domain=.example.internal;
    Max-Age=86400
```

| 属性 | 値 | 説明 |
|---|---|---|
| HttpOnly | Yes | JavaScript からアクセス不可（XSS 対策） |
| Secure | Yes | HTTPS 経由でのみ送信（Cloudflare が HTTPS 終端） |
| SameSite | Lax | 同一サイトからのリクエストでのみ Cookie を送信 |
| Path | /oauth | 認可サーバーのパスでのみ Cookie を送信 |
| Domain | .example.internal | サブドメイン間で共有 |

---

## パスワードセキュリティ

### ハッシュ方式

**argon2id** を使用する。

| パラメータ | 値 | 説明 |
|---|---|---|
| algorithm | argon2id | Argon2 のハイブリッド版（側チャネル攻撃耐性 + GPU 耐性） |
| memory | 65536 KB (64MB) | メモリコスト |
| iterations | 3 | 反復回数 |
| parallelism | 2 | 並列度 |
| salt_length | 16 bytes | ランダムソルト長 |
| hash_length | 32 bytes | 出力ハッシュ長 |

### 保存形式

```
$argon2id$v=19$m=65536,t=3,p=2$<salt_base64>$<hash_base64>
```

### パスワードポリシー

| ルール | 値 |
|---|---|
| 最小長 | 8文字 |
| 最大長 | 128文字 |
| 必須文字 | 英大文字・英小文字・数字のうち2種以上 |
| 禁止 | ログインユーザー名と同一 |

---

## Rate Limit

### ブルートフォース対策

| 対象 | 上限 | ウィンドウ | 超過時 |
|---|---|---|---|
| ログイン試行（ユーザー名単位） | 5回 | 5分 | 一時ロック + 429 |
| /token エンドポイント（クライアント単位） | 20回 | 60秒 | 429 |
| 未認証リクエスト（IP 単位） | 30回 | 60秒 | 429 |
| 認証済みリクエスト（メンバー単位） | 100回 | 60秒 | 429 |

### レスポンスヘッダー

```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1719200060
Retry-After: 30
```

---

## トークンセキュリティ

### アクセストークン（JWT）

| 項目 | 設計 |
|---|---|
| 形式 | JWT（ES256 署名） |
| 有効期間 | 15分 |
| Redis 保存 | しない（自己完結型） |
| 失効方法 | 自然失効のみ（最大15分待ち） |

### リフレッシュトークン

| 項目 | 設計 |
|---|---|
| 形式 | ランダム文字列（64 bytes、Base64url エンコード） |
| 有効期間 | 7日 |
| Redis 保存 | する（SHA-256 ハッシュをキーに保存） |
| ローテーション | 使用のたびに新しいトークンを発行、旧トークンは削除 |
| 再利用検知 | 旧トークンが使用された場合、そのメンバーの全トークンを無効化 |

### 認可コード

| 項目 | 設計 |
|---|---|
| 形式 | ランダム文字列（32 bytes、Base64url エンコード） |
| 有効期間 | 10分 |
| 使用回数 | 1回限り（取得後即削除） |
| PKCE | 必須（S256 のみ） |

---

## 入力バリデーション

### 共通

| 項目 | 対策 |
|---|---|
| SQL インジェクション | プリペアドステートメント / パラメータバインディング |
| XSS | フロントは Vue の自動エスケープ。API は JSON レスポンスのみ |
| redirect_uri | 完全一致で検証（部分一致やワイルドカード不可） |
| scope | 登録済みスコープとの照合 |
| client_id | UUID 形式の検証 |

### redirect_uri の検証

```
登録済み: https://app.example.internal/callback
リクエスト: https://app.example.internal/callback  → OK
リクエスト: https://app.example.internal/callback?  → NG
リクエスト: https://evil.example.com/callback       → NG
```

---

## HTTP セキュリティヘッダー

Nginx で付与する。

```nginx
add_header X-Content-Type-Options "nosniff" always;
add_header X-Frame-Options "DENY" always;
add_header X-XSS-Protection "0" always;
add_header Referrer-Policy "strict-origin-when-cross-origin" always;
add_header Content-Security-Policy "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'" always;
add_header Permissions-Policy "camera=(), microphone=(), geolocation=()" always;
```

| ヘッダー | 値 | 説明 |
|---|---|---|
| X-Content-Type-Options | nosniff | MIME タイプスニッフィング防止 |
| X-Frame-Options | DENY | iframe での埋め込み禁止（クリックジャッキング対策） |
| X-XSS-Protection | 0 | ブラウザの XSS フィルタ無効化（CSP に委任） |
| Referrer-Policy | strict-origin-when-cross-origin | リファラ漏洩防止 |
| Content-Security-Policy | default-src 'self' | 外部リソースの読み込み制限 |
| Permissions-Policy | camera=(), microphone=() | デバイスアクセス無効化 |

---

## 監査ログ

PostgreSQL の `auth.audit_logs` テーブルに記録する。

### 記録対象

| イベント | ログレベル |
|---|---|
| ログイン成功 | INFO |
| ログイン失敗 | WARN |
| 同意付与/撤回 | INFO |
| トークン発行 | INFO |
| トークン失効 | INFO |
| クライアント作成/更新/削除 | INFO |
| メンバー作成/更新/無効化 | INFO |
| Rate Limit 超過 | WARN |
| 不正なリフレッシュトークン再利用検知 | ERROR |

### ログに含める情報

```json
{
  "action": "login_success",
  "member_id": "uuid",
  "client_id": "my-app",
  "ip_address": "192.168.1.100",
  "user_agent": "Mozilla/5.0...",
  "timestamp": "2026-06-24T09:00:00Z",
  "details": {}
}
```

---

## Docker ネットワークセキュリティ

### ポート公開

| コンテナ | 外部公開 | 説明 |
|---|---|---|
| nginx | :80 のみ | Cloudflare Tunnel 経由でアクセス |
| auth-server | 非公開 (expose のみ) | Nginx からの内部通信のみ |
| resource-server | 非公開 (expose のみ) | Nginx からの内部通信のみ |
| redis | 非公開 | Docker 内部ネットワークのみ |
| postgres | 非公開 | Docker 内部ネットワークのみ |

### ネットワーク分離

```yaml
networks:
  frontend:
    # nginx, auth-server, resource-server
  backend:
    # auth-server, resource-server, redis, postgres
```

nginx は backend に直接アクセスできない。auth-server / resource-server が中継する。
