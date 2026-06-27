const BASE = ''

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const opts: RequestInit = {
    method,
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
  }
  if (body) opts.body = JSON.stringify(body)

  const res = await fetch(BASE + path, opts)
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: { message: res.statusText } }))
    throw { status: res.status, ...err.error }
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
    request<{ member: any; identity: any }>('GET', '/oauth/me'),

  updateIdentity: (data: { display_name: string; avatar_url?: string | null; theme_color: string; tagline?: string | null }) =>
    request<any>('PUT', '/oauth/me/identity', data),

  changePassword: (currentPassword: string, newPassword: string) =>
    request<{ status: string }>('PUT', '/oauth/me/password', { current_password: currentPassword, new_password: newPassword }),

  getConsents: () =>
    request<{ consents: any[] }>('GET', '/oauth/me/consents'),

  revokeConsent: (clientId: string) =>
    request<void>('DELETE', `/oauth/me/consents/${clientId}`),

  getConsentInfo: () =>
    request<{ client_name: string; requested_scopes: any[] }>('GET', '/oauth/consent'),

  submitConsent: (approved: boolean, scopes: string[]) =>
    request<{ redirect_to: string }>('POST', '/oauth/consent', { approved, scopes }),

  getMyClients: () =>
    request<{ clients: any[]; total: number }>('GET', '/oauth/me/clients'),

  createClient: (data: { name: string; client_type: string; redirect_uris: string[]; allowed_grant_types: string[]; description?: string }) =>
    request<{ client: any; client_secret?: string }>('POST', '/oauth/me/clients', data),

  updateClient: (id: string, data: { name?: string; redirect_uris?: string[]; allowed_grant_types?: string[] }) =>
    request<any>('PUT', `/oauth/me/clients/${id}`, data),

  deleteClient: (id: string) =>
    request<void>('DELETE', `/oauth/me/clients/${id}`),

  // Admin API
  admin: {
    listMembers: (page = 1, perPage = 20) =>
      request<{ members: any[]; total: number; page: number; per_page: number }>('GET', `/oauth/admin/members?page=${page}&per_page=${perPage}`),

    createMember: (data: { username: string; password: string; email: string }) =>
      request<any>('POST', '/oauth/admin/members', data),

    getMember: (id: string) =>
      request<any>('GET', `/oauth/admin/members/${id}`),

    updateMember: (id: string, data: { username?: string; email?: string; role?: string; is_active?: boolean; password?: string }) =>
      request<any>('PUT', `/oauth/admin/members/${id}`, data),

    deleteMember: (id: string) =>
      request<void>('DELETE', `/oauth/admin/members/${id}`),

    resetPassword: (id: string) =>
      request<{ temporary_password: string; message: string }>('POST', `/oauth/admin/members/${id}/reset-password`),

    listClients: (page = 1, perPage = 20) =>
      request<{ clients: any[]; total: number; page: number; per_page: number }>('GET', `/oauth/admin/clients?page=${page}&per_page=${perPage}`),

    deleteAdminClient: (id: string) =>
      request<void>('DELETE', `/oauth/admin/clients/${id}`),
  },
}
