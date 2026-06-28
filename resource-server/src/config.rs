use std::env;

pub struct Config {
    pub host: String,
    pub port: u16,
    pub database_url: String,
    pub database_max_connections: u32,
    pub jwks_url: String,
    pub jwks_cache_ttl: u64,
    pub jwt_issuer: String,
    pub log_level: String,
}

impl Config {
    pub fn from_env() -> Result<Self, String> {
        let database_url =
            env::var("RESOURCE_DATABASE_URL").map_err(|_| "RESOURCE_DATABASE_URL is required")?;
        let jwks_url =
            env::var("RESOURCE_JWKS_URL").map_err(|_| "RESOURCE_JWKS_URL is required")?;

        Ok(Config {
            host: env::var("RESOURCE_SERVER_HOST").unwrap_or_else(|_| "0.0.0.0".to_string()),
            port: env::var("RESOURCE_SERVER_PORT")
                .unwrap_or_else(|_| "8081".to_string())
                .parse()
                .map_err(|_| "RESOURCE_SERVER_PORT must be a number")?,
            database_url,
            database_max_connections: env::var("RESOURCE_DATABASE_MAX_CONNECTIONS")
                .unwrap_or_else(|_| "10".to_string())
                .parse()
                .map_err(|_| "RESOURCE_DATABASE_MAX_CONNECTIONS must be a number")?,
            jwks_url,
            jwks_cache_ttl: env::var("RESOURCE_JWKS_CACHE_TTL")
                .unwrap_or_else(|_| "3600".to_string())
                .parse()
                .unwrap_or(3600),
            jwt_issuer: env::var("RESOURCE_JWT_ISSUER")
                .unwrap_or_else(|_| "http://localhost".to_string()),
            log_level: env::var("RESOURCE_LOG_LEVEL").unwrap_or_else(|_| "info".to_string()),
        })
    }
}
