# Redis データ設計

## 接続情報

```
Host: 127.0.0.1
Port: 6379
DB 0: 認可サーバー用（認可コード、リフレッシュトークン、セッション）
DB 1: Rate Limit 用
```

---

## キー設計

### 命名規則

```
{サービス}:{データ種別}:{識別子}
```

例: `auth:code:abc123`, `auth:refresh:xyz789`

### キー一覧

| キー | TTL | 用途 |
|---|---|---|
| `auth:code:{code}` | 600秒 | 認可コード |
| `auth:refresh:{hash}` | 604800秒 | リフレッシュトークン |
| `auth:session:{id}` | 86400秒 | メンバーセッション |
| `auth:member_refreshes:{member_id}` | なし | メンバー別リフレッシュトークン管理 |
| `ratelimit:*` | 60〜300秒 | Rate Limit |

※ アクセストークンは JWT（自己完結型）のため Redis に保存しない。
失効制御はリフレッシュトークンの削除で行い、JWT は最大15分で自然失効する。

---

## 認可コード

認可エンドポイントで発行し、トークンエンドポイントで消費（1回限り）。

**キー:** `auth:code:{認可コード}`

**TTL:** 600 秒（10分）

**値（JSON）:**

```json
{
  "client_id": "my-app",
  "member_id": "550e8400-e29b-41d4-a716-446655440000",
  "redirect_uri": "https://my-app.example/callback",
  "scope": "read write",
  "code_challenge": "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
  "code_challenge_method": "S256",
  "issued_at": 1719196400
}
```

**操作:**

| 操作 | コマンド | タイミング |
|---|---|---|
| 保存 | `SET auth:code:{code} {json} EX 600` | /authorize 成功時 |
| 取得+削除 | `GET` → `DEL` (トランザクション) | /token リクエスト時 |

※ 認可コードは1回使い切り。取得後すぐに削除し再利用を防ぐ。

---

## リフレッシュトークン

アクセストークン（JWT）の再発行に使用する。失効制御の唯一の手段。

**キー:** `auth:refresh:{トークンのSHA-256ハッシュ}`

**TTL:** 604800 秒（7日）

**値（JSON）:**

```json
{
  "member_id": "550e8400-e29b-41d4-a716-446655440000",
  "client_id": "my-app",
  "scope": "read write",
  "issued_at": 1719196400,
  "expires_at": 1719801200
}
```

**操作:**

| 操作 | コマンド | タイミング |
|---|---|---|
| 保存 | `SET auth:refresh:{hash} {json} EX 604800` | /token 発行時 |
| 取得 | `GET auth:refresh:{hash}` | refresh_token grant 時 |
| 削除 | `DEL auth:refresh:{hash}` | /revoke 時、ローテーション時 |

### リフレッシュトークンローテーション

```
1. 旧リフレッシュトークンを GET で取得
2. 旧リフレッシュトークンを DEL で削除
3. 新アクセストークン（JWT）+ 新リフレッシュトークンを発行・保存
```

旧トークンは再利用不可。再利用を検知した場合、そのメンバーの全リフレッシュトークンを無効化する。

### トークン失効の仕組み

```
管理者が「このメンバーのアクセスを停止したい」場合:

1. auth:member_refreshes:{member_id} から全リフレッシュトークンハッシュを取得
2. 各 auth:refresh:{hash} を DEL
3. auth:member_refreshes:{member_id} を DEL

→ リフレッシュ不可になり、現在の JWT は最大15分で自然失効
```

---

## セッション

ログイン状態の管理。Cookie にセッション ID を保持し、Redis で状態管理。

**キー:** `auth:session:{セッションID}`

**TTL:** 86400 秒（24時間）

**値（JSON）:**

```json
{
  "member_id": "550e8400-e29b-41d4-a716-446655440000",
  "username": "tanaka",
  "ip_address": "192.168.1.100",
  "user_agent": "Mozilla/5.0...",
  "created_at": 1719196400,
  "last_accessed_at": 1719200000,
  "oauth_params": {
    "client_id": "my-app",
    "redirect_uri": "https://my-app.example/callback",
    "scope": "read write",
    "state": "random-state",
    "code_challenge": "xxx",
    "code_challenge_method": "S256"
  }
}
```

`oauth_params` は認可フロー中のみ一時的に保持。ログイン→同意→認可コード発行の間パラメータを引き継ぐ。

**操作:**

| 操作 | コマンド | タイミング |
|---|---|---|
| 作成 | `SET auth:session:{id} {json} EX 86400` | ログイン成功時 |
| 取得 | `GET auth:session:{id}` | 各リクエスト時 |
| 更新 | `SET auth:session:{id} {json} EX 86400` | last_accessed_at 更新 |
| 削除 | `DEL auth:session:{id}` | ログアウト時 |

---

## メンバー別リフレッシュトークン管理

特定メンバーの全リフレッシュトークンを一括失効させるためのインデックス。

**キー:** `auth:member_refreshes:{member_id}`

**型:** SET

**値:** リフレッシュトークンハッシュの集合

**操作:**

| 操作 | コマンド | タイミング |
|---|---|---|
| 追加 | `SADD auth:member_refreshes:{member_id} {refresh_hash}` | リフレッシュトークン発行時 |
| 一覧 | `SMEMBERS auth:member_refreshes:{member_id}` | 管理画面、一括失効時 |
| 削除 | `SREM auth:member_refreshes:{member_id} {refresh_hash}` | ローテーション時 |
| 全削除 | `DEL auth:member_refreshes:{member_id}` + 各 refresh DEL | 全トークン失効時 |

※ TTL なし。リフレッシュトークン失効時に SREM で整合性を保つ。

---

## Rate Limit

Sliding Window 方式で Rate Limit を実装する。

**キー:** `ratelimit:{対象}:{識別子}`

**DB:** 1（Rate Limit 専用）

### メンバー単位

**キー:** `ratelimit:member:{member_id}`

**TTL:** 60 秒

```
INCR ratelimit:member:{member_id}
→ 1 の場合: EXPIRE ratelimit:member:{member_id} 60
→ 100 超過: リクエスト拒否 (429)
```

### IP 単位（未認証リクエスト）

**キー:** `ratelimit:ip:{ip_address}`

**TTL:** 60 秒

```
INCR ratelimit:ip:{ip_address}
→ 1 の場合: EXPIRE ratelimit:ip:{ip_address} 60
→ 30 超過: リクエスト拒否 (429)
```

### ログイン試行（ブルートフォース対策）

**キー:** `ratelimit:login:{username}`

**TTL:** 300 秒（5分）

```
INCR ratelimit:login:{username}
→ 5 超過: ログイン一時ロック
```

### Rate Limit 設定一覧

| 対象 | キー | 上限 | ウィンドウ |
|---|---|---|---|
| 認証済みメンバー | `ratelimit:member:{member_id}` | 100 req | 60秒 |
| 未認証 IP | `ratelimit:ip:{ip}` | 30 req | 60秒 |
| ログイン試行 | `ratelimit:login:{username}` | 5 回 | 300秒 |
| /token エンドポイント | `ratelimit:token:{client_id}` | 20 req | 60秒 |

---

## Redis 設定（推奨）

```conf
# メモリ上限
maxmemory 64mb

# 上限到達時の挙動（TTL付きキーを優先削除）
maxmemory-policy volatile-ttl

# 永続化（AOF）
appendonly yes
appendfsync everysec

# DB 数
databases 2
```
