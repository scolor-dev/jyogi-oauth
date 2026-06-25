mod auth;
mod config;
mod db;
mod handler;
mod jwt;

use actix_web::{web, App, HttpServer};
use std::sync::Arc;
use std::time::Duration;
use tokio::sync::RwLock;

#[actix_web::main]
async fn main() -> std::io::Result<()> {
    env_logger::init();

    let cfg = config::Config::from_env().expect("Failed to load config");

    let pool = db::create_pool(&cfg.database_url)
        .await
        .expect("Failed to connect to database");

    let jwks_cache = match jwt::JwksCache::new(&cfg.jwks_url, Duration::from_secs(cfg.jwks_cache_ttl)).await {
        Ok(cache) => Arc::new(RwLock::new(cache)),
        Err(e) => {
            log::warn!("Failed to fetch JWKS on startup: {e}. Will retry on first request.");
            Arc::new(RwLock::new(
                jwt::JwksCache::new_empty(&cfg.jwks_url, Duration::from_secs(cfg.jwks_cache_ttl)),
            ))
        }
    };

    let state = web::Data::new(auth::AppState {
        pool,
        jwks_cache,
        jwt_issuer: cfg.jwt_issuer,
    });

    log::info!("Resource server starting on :{}", cfg.port);

    HttpServer::new(move || {
        App::new()
            .app_data(state.clone())
            .route("/api/health", web::get().to(handler::health::health))
            .route("/api/v1/members/me/identity", web::get().to(handler::identity::get_my_identity))
            .route("/api/v1/members/me/identity", web::put().to(handler::identity::upsert_my_identity))
            .route("/api/v1/members/identities", web::get().to(handler::identity::get_batch_identities))
            .route("/api/v1/members/{member_id}/identity", web::get().to(handler::identity::get_member_identity))
    })
    .bind(("0.0.0.0", cfg.port))?
    .run()
    .await
}
