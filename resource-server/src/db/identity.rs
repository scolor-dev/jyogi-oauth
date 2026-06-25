use chrono::{DateTime, Utc};
use sqlx::PgPool;
use uuid::Uuid;

#[derive(Debug, serde::Serialize, sqlx::FromRow)]
pub struct MemberIdentity {
    pub id: Uuid,
    pub member_id: Uuid,
    pub display_name: String,
    pub avatar_url: Option<String>,
    pub theme_color: String,
    pub tagline: Option<String>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

pub async fn get_by_member_id(
    pool: &PgPool,
    member_id: Uuid,
) -> Result<Option<MemberIdentity>, sqlx::Error> {
    sqlx::query_as::<_, MemberIdentity>(
        "SELECT id, member_id, display_name, avatar_url, theme_color, tagline, created_at, updated_at
         FROM resource.member_identities WHERE member_id = $1",
    )
    .bind(member_id)
    .fetch_optional(pool)
    .await
}

pub async fn upsert(
    pool: &PgPool,
    member_id: Uuid,
    display_name: &str,
    avatar_url: Option<&str>,
    theme_color: &str,
    tagline: Option<&str>,
) -> Result<MemberIdentity, sqlx::Error> {
    sqlx::query_as::<_, MemberIdentity>(
        "INSERT INTO resource.member_identities (member_id, display_name, avatar_url, theme_color, tagline)
         VALUES ($1, $2, $3, $4, $5)
         ON CONFLICT (member_id) DO UPDATE SET
           display_name = EXCLUDED.display_name,
           avatar_url = EXCLUDED.avatar_url,
           theme_color = EXCLUDED.theme_color,
           tagline = EXCLUDED.tagline
         RETURNING id, member_id, display_name, avatar_url, theme_color, tagline, created_at, updated_at",
    )
    .bind(member_id)
    .bind(display_name)
    .bind(avatar_url)
    .bind(theme_color)
    .bind(tagline)
    .fetch_one(pool)
    .await
}

pub async fn get_by_member_ids(
    pool: &PgPool,
    member_ids: &[Uuid],
) -> Result<Vec<MemberIdentity>, sqlx::Error> {
    sqlx::query_as::<_, MemberIdentity>(
        "SELECT id, member_id, display_name, avatar_url, theme_color, tagline, created_at, updated_at
         FROM resource.member_identities WHERE member_id = ANY($1)",
    )
    .bind(member_ids)
    .fetch_all(pool)
    .await
}
