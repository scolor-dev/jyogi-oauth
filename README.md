# jyogi-oauth

OAuth 2.0 / OpenID Connect authorization server, resource server, and Vue dashboard SPA.

## Services

- `auth-server/`: Go authorization server and dashboard API
- `resource-server/`: Rust resource server for member identity APIs
- `web/`: Vue 3 dashboard SPA
- `nginx/`: local and production reverse proxy configuration
- `docs/`: generated project documentation

## Local Development

Generate ES256 keys on first setup:

```bash
mkdir -p keys
openssl ecparam -genkey -name prime256v1 -noout -out keys/private.pem
openssl ec -in keys/private.pem -pubout -out keys/public.pem
```

For JWT key rotation, use directory mode instead of the single-key paths:

```bash
mkdir -p keys/jwt
openssl ecparam -genkey -name prime256v1 -noout -out keys/jwt/key-2026-06.private.pem
openssl ec -in keys/jwt/key-2026-06.private.pem -pubout -out keys/jwt/key-2026-06.public.pem
```

Set `AUTH_JWT_KEYS_DIR=/keys/jwt` and `AUTH_JWT_ACTIVE_KID=key-2026-06`. The auth server signs new tokens with the active private key and publishes every `*.public.pem` in JWKS so old tokens continue to verify during rotation.

Build the SPA and start all services:

```bash
cd web && npm install && npm run build && cd ..
docker compose up -d --build
```

The local entrypoint is `http://localhost`.

## Verification

```bash
GOCACHE="$PWD/.cache/go-build" go test ./...
cd resource-server && cargo check
cd web && npm run build
```

## Production Config

Use `config/.env.prod.example` as the template for `config/.env.prod`.
The real `.env.prod` and `keys/` must stay out of Git.

Production compose and nginx overrides are kept in:

- `docker-compose.prod.yml`
- `nginx/prod.conf`
