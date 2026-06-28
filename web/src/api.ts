import type { AuditLog, ConsentInfo, ConsentRequestInfo, Identity, Member, OAuthClient, Scope, SessionInfo } from './types'

const BASE = ''

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const opts: RequestInit = {
    method,
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
  }
  if (body !== undefined) opts.body = JSON.stringify(body)

  const res = await fetch(BASE + path, opts)
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: { message: res.statusText } }))
    const error = typeof err === 'object' && err !== null && 'error' in err ? err.error : undefined
    throw {
      status: res.status,
      ...(typeof error === 'object' && error !== null ? error : { message: res.statusText }),
    }
  }
  if (res.status === 204) return undefined as T
  return res.json()
}

export const api = {
  login: (username: string, password: string) =>
    request<{ redirect_to: string }>('POST', '/oauth/login', { username, password }),

  logout: () =>
    request<{ status: string }>('POST', '/oauth/logout'),

  getMe: () =>
    request<{ member: Member; identity: Identity | null }>('GET', '/oauth/me'),

  updateIdentity: (data: { display_name: string; avatar_url?: string | null; theme_color: string; tagline?: string | null }) =>
    request<Identity>('PUT', '/oauth/me/identity', data),

  changePassword: (currentPassword: string, newPassword: string) =>
    request<{ status: string }>('PUT', '/oauth/me/password', { current_password: currentPassword, new_password: newPassword }),

  getConsents: () =>
    request<{ consents: ConsentInfo[] }>('GET', '/oauth/me/consents'),

  revokeConsent: (clientId: string) =>
    request<void>('DELETE', `/oauth/me/consents/${clientId}`),

  getConsentInfo: () =>
    request<ConsentRequestInfo>('GET', '/oauth/consent'),

  submitConsent: (approved: boolean, scopes: string[]) =>
    request<{ redirect_to: string }>('POST', '/oauth/consent', { approved, scopes }),

  getSessions: () =>
    request<{ sessions: SessionInfo[] }>('GET', '/oauth/me/sessions'),

  revokeSession: (sessionId: string) =>
    request<void>('DELETE', `/oauth/me/sessions/${sessionId}`),

  getMyClients: () =>
    request<{ clients: OAuthClient[]; total: number }>('GET', '/oauth/me/clients'),

  createClient: (data: { name: string; client_type: OAuthClient['client_type']; redirect_uris: string[]; allowed_grant_types: OAuthClient['allowed_grant_types']; description?: string; icon_url?: string }) =>
    request<{ client: OAuthClient; client_secret?: string }>('POST', '/oauth/me/clients', data),

  updateClient: (id: string, data: { name?: string; description?: string; icon_url?: string; redirect_uris?: string[]; allowed_grant_types?: OAuthClient['allowed_grant_types'] }) =>
    request<OAuthClient>('PUT', `/oauth/me/clients/${id}`, data),

  deleteClient: (id: string) =>
    request<void>('DELETE', `/oauth/me/clients/${id}`),

  rotateClientSecret: (id: string) =>
    request<{ client_id: string; client_secret: string; message: string }>('POST', `/oauth/me/clients/${id}/rotate-secret`),

  // Admin API
  admin: {
    listMembers: (page = 1, perPage = 20) =>
      request<{ members: Member[]; total: number; page: number; per_page: number }>('GET', `/oauth/admin/members?page=${page}&per_page=${perPage}`),

    createMember: (data: { username: string; email: string; password?: string; must_change_password?: boolean }) =>
      request<{ member: Member; temporary_password?: string }>('POST', '/oauth/admin/members', data),

    updateMember: (id: string, data: { username?: string; email?: string; role?: Member['role']; is_active?: boolean; password?: string }) =>
      request<Member>('PUT', `/oauth/admin/members/${id}`, data),

    deleteMember: (id: string) =>
      request<void>('DELETE', `/oauth/admin/members/${id}`),

    resetPassword: (id: string) =>
      request<{ temporary_password: string; message: string }>('POST', `/oauth/admin/members/${id}/reset-password`),

    listClients: (page = 1, perPage = 20) =>
      request<{ clients: OAuthClient[]; total: number; page: number; per_page: number }>('GET', `/oauth/admin/clients?page=${page}&per_page=${perPage}`),

    rotateClientSecret: (id: string) =>
      request<{ client_id: string; client_secret: string; message: string }>('POST', `/oauth/admin/clients/${id}/rotate-secret`),

    deleteAdminClient: (id: string) =>
      request<void>('DELETE', `/oauth/admin/clients/${id}`),

    listScopes: () =>
      request<{ scopes: Scope[]; total: number }>('GET', '/oauth/admin/scopes'),

    createScope: (data: { name: string; description?: string; is_default?: boolean }) =>
      request<Scope>('POST', '/oauth/admin/scopes', data),

    updateScope: (id: string, data: { name?: string; description?: string; is_default?: boolean }) =>
      request<Scope>('PUT', `/oauth/admin/scopes/${id}`, data),

    deleteScope: (id: string) =>
      request<void>('DELETE', `/oauth/admin/scopes/${id}`),

    listAuditLogs: (params: { page?: number; perPage?: number; action?: string; memberId?: string; from?: string; to?: string } = {}) => {
      const query = new URLSearchParams()
      if (params.page) query.set('page', String(params.page))
      if (params.perPage) query.set('per_page', String(params.perPage))
      if (params.action) query.set('action', params.action)
      if (params.memberId) query.set('member_id', params.memberId)
      if (params.from) query.set('from', params.from)
      if (params.to) query.set('to', params.to)
      const suffix = query.toString() ? `?${query.toString()}` : ''
      return request<{ items: AuditLog[]; total: number; page: number; per_page: number }>('GET', `/oauth/admin/audit-logs${suffix}`)
    },
  },
}
