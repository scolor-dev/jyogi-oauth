use base64::engine::general_purpose::URL_SAFE_NO_PAD;
use base64::Engine;
use jsonwebtoken::{Algorithm, DecodingKey, Validation};
use serde::Deserialize;
use std::collections::HashMap;
use std::sync::Arc;
use std::time::{Duration, Instant};
use tokio::sync::RwLock;

#[derive(Debug, Deserialize)]
pub struct JwksResponse {
    pub keys: Vec<Jwk>,
}

#[derive(Debug, Deserialize)]
pub struct Jwk {
    pub kty: String,
    pub kid: Option<String>,
    pub alg: Option<String>,
    pub crv: Option<String>,
    pub x: Option<String>,
    pub y: Option<String>,
}

#[allow(dead_code)]
#[derive(Debug, Clone, Deserialize)]
pub struct Claims {
    pub iss: Option<String>,
    pub sub: Option<String>,
    pub aud: Option<serde_json::Value>,
    pub exp: Option<u64>,
    pub iat: Option<u64>,
    pub jti: Option<String>,
    pub client_id: Option<String>,
    pub scope: Option<String>,
    pub username: Option<String>,
    pub grant_type: Option<String>,
}

pub struct JwksCache {
    keys: HashMap<String, DecodingKey>,
    jwks_url: String,
    last_fetched: Instant,
    ttl: Duration,
}

impl JwksCache {
    pub fn new_empty(jwks_url: &str, ttl: Duration) -> Self {
        JwksCache {
            keys: HashMap::new(),
            jwks_url: jwks_url.to_string(),
            last_fetched: Instant::now() - ttl - Duration::from_secs(1),
            ttl,
        }
    }

    pub async fn new(jwks_url: &str, ttl: Duration) -> Result<Self, Box<dyn std::error::Error>> {
        let mut cache = JwksCache {
            keys: HashMap::new(),
            jwks_url: jwks_url.to_string(),
            last_fetched: Instant::now(),
            ttl,
        };
        cache.refresh().await?;
        Ok(cache)
    }

    pub async fn refresh(&mut self) -> Result<(), Box<dyn std::error::Error>> {
        let resp: JwksResponse = reqwest::get(&self.jwks_url).await?.json().await?;

        let mut keys = HashMap::new();
        for jwk in &resp.keys {
            if jwk.kty != "EC" {
                continue;
            }
            if jwk.alg.as_deref() != Some("ES256") || jwk.crv.as_deref() != Some("P-256") {
                continue;
            }
            let kid = match &jwk.kid {
                Some(k) => k.clone(),
                None => continue,
            };
            let x = match &jwk.x {
                Some(v) => URL_SAFE_NO_PAD.decode(v)?,
                None => continue,
            };
            let y = match &jwk.y {
                Some(v) => URL_SAFE_NO_PAD.decode(v)?,
                None => continue,
            };

            // Build uncompressed EC point: 0x04 || x || y
            let mut point = Vec::with_capacity(1 + x.len() + y.len());
            point.push(0x04);
            point.extend_from_slice(&x);
            point.extend_from_slice(&y);

            let decoding_key = DecodingKey::from_ec_der(&point);
            keys.insert(kid, decoding_key);
        }

        self.keys = keys;
        self.last_fetched = Instant::now();
        log::info!("JWKS refreshed: {} keys loaded", self.keys.len());
        Ok(())
    }

    pub fn is_expired(&self) -> bool {
        self.last_fetched.elapsed() > self.ttl
    }

    pub fn get_key(&self, kid: &str) -> Option<&DecodingKey> {
        self.keys.get(kid)
    }
}

pub async fn verify_token(
    token: &str,
    cache: &Arc<RwLock<JwksCache>>,
    issuer: &str,
) -> Result<Claims, String> {
    let header =
        jsonwebtoken::decode_header(token).map_err(|e| format!("Invalid JWT header: {e}"))?;

    let kid = header.kid.ok_or("JWT missing kid")?;

    {
        let cache_read = cache.read().await;
        if cache_read.is_expired() {
            drop(cache_read);
            let mut cache_write = cache.write().await;
            if cache_write.is_expired() {
                cache_write
                    .refresh()
                    .await
                    .map_err(|e| format!("Failed to refresh JWKS: {e}"))?;
            }
        }
    }

    let cache_read = cache.read().await;
    let key = match cache_read.get_key(&kid) {
        Some(k) => k.clone(),
        None => {
            drop(cache_read);
            let mut cache_write = cache.write().await;
            if let Err(e) = cache_write.refresh().await {
                log::warn!("Failed to refresh JWKS: {e}");
            }
            cache_write
                .get_key(&kid)
                .cloned()
                .ok_or_else(|| format!("Unknown kid: {kid}"))?
        }
    };

    let mut validation = Validation::new(Algorithm::ES256);
    validation.set_issuer(&[issuer]);
    validation.validate_aud = false;

    let token_data = jsonwebtoken::decode::<Claims>(token, &key, &validation)
        .map_err(|e| format!("JWT validation failed: {e}"))?;

    Ok(token_data.claims)
}
