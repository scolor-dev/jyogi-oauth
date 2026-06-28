use actix_web::{web, HttpResponse};

use crate::auth::AppState;

pub async fn health(state: web::Data<AppState>) -> HttpResponse {
    let pg_status = match sqlx::query("SELECT 1").execute(&state.pool).await {
        Ok(_) => "connected",
        Err(_) => "disconnected",
    };

    let status = if pg_status == "connected" {
        "healthy"
    } else {
        "unhealthy"
    };

    let code = if status == "healthy" { 200 } else { 503 };

    HttpResponse::build(actix_web::http::StatusCode::from_u16(code).unwrap()).json(
        serde_json::json!({
            "status": status,
            "service": "resource-server",
            "version": "0.2.2",
            "dependencies": {
                "postgresql": pg_status
            }
        }),
    )
}
