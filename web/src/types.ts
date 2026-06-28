export type Role = 'member' | 'moderator' | 'admin'
export type ClientType = 'public' | 'confidential'
export type GrantType = 'authorization_code' | 'refresh_token' | 'client_credentials'

export interface Member {
  id: string
  username: string
  email: string
  role: Role
  must_change_password: boolean
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface Identity {
  member_id: string
  display_name: string
  avatar_url: string | null
  theme_color: string
  tagline: string | null
  updated_at: string
}

export interface OAuthClient {
  id: string
  client_id: string
  name: string
  description?: string | null
  icon_url?: string | null
  client_type: ClientType
  redirect_uris: string[]
  allowed_grant_types: GrantType[]
  is_active: boolean
  created_by?: string
  created_at: string
  updated_at: string
}

export interface ConsentInfo {
  client_id: string
  client_name: string
  scopes: string[]
  granted_at: string
}

export interface ScopeInfo {
  name: string
  description?: string
}

export interface Scope {
  id: string
  name: string
  description?: string | null
  is_default: boolean
  created_at: string
}

export interface ConsentRequestInfo {
  client_name: string
  client_icon_url?: string | null
  client_description?: string | null
  requested_scopes: ScopeInfo[]
}

export interface AuditLog {
  id: string
  member_id: string | null
  client_id: string | null
  action: string
  ip_address: string | null
  user_agent: string | null
  details: Record<string, unknown> | null
  created_at: string
}

export interface SessionInfo {
  session_id: string
  ip_address: string
  user_agent: string
  created_at: number
  last_accessed_at: number
  is_current?: boolean
}

export interface ApiError {
  status?: number
  code?: string
  message?: string
}

export function getErrorMessage(error: unknown, fallback: string): string {
  if (typeof error === 'object' && error !== null && 'message' in error) {
    const message = (error as { message?: unknown }).message
    if (typeof message === 'string' && message !== '') return message
  }
  return fallback
}
