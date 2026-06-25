use actix_web::{web, HttpRequest, HttpResponse};
use serde::Deserialize;
use uuid::Uuid;

use crate::auth::{self, AppState};
use crate::db;

pub async fn get_my_identity(
    req: HttpRequest,
    state: web::Data<AppState>,
) -> HttpResponse {
    let claims = match auth::extract_and_verify(&req, &state).await {
        Ok(c) => c,
        Err(e) => return e,
    };
    if let Err(e) = auth::require_scope(&claims, "identity") {
        return e;
    }

    let member_id = match auth::extract_member_id(&claims) {
        Ok(id) => id,
        Err(e) => return e,
    };

    match db::identity::get_by_member_id(&state.pool, member_id).await {
        Ok(Some(identity)) => HttpResponse::Ok().json(identity),
        Ok(None) => HttpResponse::NotFound().json(serde_json::json!({
            "error": {"code": "identity_not_found", "message": "Identity has not been created yet"}
        })),
        Err(e) => {
            log::error!("DB error: {e}");
            HttpResponse::InternalServerError().json(serde_json::json!({
                "error": {"code": "internal_error", "message": "Database error"}
            }))
        }
    }
}

#[derive(Deserialize)]
pub struct UpsertIdentityRequest {
    pub display_name: String,
    pub avatar_url: Option<String>,
    pub theme_color: String,
    pub tagline: Option<String>,
}

pub async fn upsert_my_identity(
    req: HttpRequest,
    state: web::Data<AppState>,
    body: web::Json<UpsertIdentityRequest>,
) -> HttpResponse {
    let claims = match auth::extract_and_verify(&req, &state).await {
        Ok(c) => c,
        Err(e) => return e,
    };
    if !auth::has_scope(&claims, "identity:write") && !auth::has_scope(&claims, "write") {
        return HttpResponse::Forbidden().json(serde_json::json!({
            "error": {"code": "insufficient_scope", "message": "Required scope: identity:write or write"}
        }));
    }

    let member_id = match auth::extract_member_id(&claims) {
        Ok(id) => id,
        Err(e) => return e,
    };

    if body.display_name.is_empty() || body.display_name.len() > 100 {
        return HttpResponse::BadRequest().json(serde_json::json!({
            "error": {"code": "validation_error", "message": "display_name must be 1-100 characters"}
        }));
    }

    if let Some(ref tagline) = body.tagline {
        if tagline.is_empty() || tagline.chars().count() > 8 {
            return HttpResponse::BadRequest().json(serde_json::json!({
                "error": {"code": "validation_error", "message": "tagline must be 1-8 characters"}
            }));
        }
    }

    let color_re = regex_lite::Regex::new(r"^#[0-9A-Fa-f]{6}$").unwrap();
    if !color_re.is_match(&body.theme_color) {
        return HttpResponse::BadRequest().json(serde_json::json!({
            "error": {"code": "validation_error", "message": "theme_color must be a hex color code (#RRGGBB)"}
        }));
    }

    match db::identity::upsert(
        &state.pool,
        member_id,
        &body.display_name,
        body.avatar_url.as_deref(),
        &body.theme_color,
        body.tagline.as_deref(),
    )
    .await
    {
        Ok(identity) => HttpResponse::Ok().json(identity),
        Err(e) => {
            log::error!("DB error: {e}");
            HttpResponse::InternalServerError().json(serde_json::json!({
                "error": {"code": "internal_error", "message": "Database error"}
            }))
        }
    }
}

pub async fn get_member_identity(
    req: HttpRequest,
    state: web::Data<AppState>,
    path: web::Path<Uuid>,
) -> HttpResponse {
    let claims = match auth::extract_and_verify(&req, &state).await {
        Ok(c) => c,
        Err(e) => return e,
    };
    if let Err(e) = auth::require_scope(&claims, "identity") {
        return e;
    }

    let member_id = path.into_inner();

    match db::identity::get_by_member_id(&state.pool, member_id).await {
        Ok(Some(identity)) => HttpResponse::Ok().json(identity),
        Ok(None) => HttpResponse::NotFound().json(serde_json::json!({
            "error": {"code": "identity_not_found", "message": "Identity not found"}
        })),
        Err(e) => {
            log::error!("DB error: {e}");
            HttpResponse::InternalServerError().json(serde_json::json!({
                "error": {"code": "internal_error", "message": "Database error"}
            }))
        }
    }
}

#[derive(Deserialize)]
pub struct BatchQuery {
    pub ids: String,
}

pub async fn get_batch_identities(
    req: HttpRequest,
    state: web::Data<AppState>,
    query: web::Query<BatchQuery>,
) -> HttpResponse {
    let claims = match auth::extract_and_verify(&req, &state).await {
        Ok(c) => c,
        Err(e) => return e,
    };
    if let Err(e) = auth::require_scope(&claims, "identity") {
        return e;
    }

    let ids: Vec<Uuid> = query
        .ids
        .split(',')
        .filter_map(|s| Uuid::parse_str(s.trim()).ok())
        .take(50)
        .collect();

    if ids.is_empty() {
        return HttpResponse::Ok().json(serde_json::json!({"data": []}));
    }

    match db::identity::get_by_member_ids(&state.pool, &ids).await {
        Ok(identities) => HttpResponse::Ok().json(serde_json::json!({"data": identities})),
        Err(e) => {
            log::error!("DB error: {e}");
            HttpResponse::InternalServerError().json(serde_json::json!({
                "error": {"code": "internal_error", "message": "Database error"}
            }))
        }
    }
}
