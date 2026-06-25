use actix_web::{web, HttpRequest, HttpResponse};
use sqlx::PgPool;
use std::sync::Arc;
use tokio::sync::RwLock;
use uuid::Uuid;

use crate::jwt::{self, Claims, JwksCache};

pub struct AppState {
    pub pool: PgPool,
    pub jwks_cache: Arc<RwLock<JwksCache>>,
    pub jwt_issuer: String,
}

pub fn has_scope(claims: &Claims, required: &str) -> bool {
    claims
        .scope
        .as_ref()
        .map(|s| s.split(' ').any(|sc| sc == required))
        .unwrap_or(false)
}

pub async fn extract_and_verify(
    req: &HttpRequest,
    state: &web::Data<AppState>,
) -> Result<Claims, HttpResponse> {
    let auth_header = req
        .headers()
        .get("Authorization")
        .and_then(|v| v.to_str().ok())
        .unwrap_or("");

    if !auth_header.starts_with("Bearer ") {
        return Err(HttpResponse::Unauthorized()
            .insert_header(("WWW-Authenticate", "Bearer realm=\"jyogi-oauth\""))
            .json(serde_json::json!({
                "error": {"code": "unauthorized", "message": "Bearer token required"}
            })));
    }

    let token = &auth_header[7..];
    jwt::verify_token(token, &state.jwks_cache, &state.jwt_issuer)
        .await
        .map_err(|e| {
            HttpResponse::Unauthorized()
                .insert_header(("WWW-Authenticate", "Bearer realm=\"jyogi-oauth\""))
                .json(serde_json::json!({
                    "error": {"code": "unauthorized", "message": e}
                }))
        })
}

pub fn require_scope(claims: &Claims, scope: &str) -> Result<(), HttpResponse> {
    if !has_scope(claims, scope) {
        return Err(HttpResponse::Forbidden().json(serde_json::json!({
            "error": {"code": "insufficient_scope", "message": format!("Required scope: {scope}")}
        })));
    }
    Ok(())
}

pub fn extract_member_id(claims: &Claims) -> Result<Uuid, HttpResponse> {
    claims
        .sub
        .as_deref()
        .and_then(|s| Uuid::parse_str(s).ok())
        .ok_or_else(|| {
            HttpResponse::BadRequest().json(serde_json::json!({
                "error": {"code": "bad_request", "message": "Invalid subject in token"}
            }))
        })
}
