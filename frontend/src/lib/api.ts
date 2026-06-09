import type {
  AuditLog,
  Client,
  ClientRole,
  Department,
  Group,
  Policy,
  PolicyAssignment,
  Role,
  SessionInfo,
  User,
} from './types'

const tokenKey = 'gopenid.token'
const refreshKey = 'gopenid.refresh'
const userKey = 'gopenid.user'

export type SessionUser = {
  id: number
  email: string
  name: string
  title?: string
  roles: string[]
}

export const auth = {
  get token() {
    return localStorage.getItem(tokenKey) ?? ''
  },
  get refreshToken() {
    return localStorage.getItem(refreshKey) ?? ''
  },
  get user(): SessionUser | null {
    const raw = localStorage.getItem(userKey)
    if (!raw) return null
    try {
      return JSON.parse(raw) as SessionUser
    } catch {
      return null
    }
  },
  isTokenValid() {
    const token = localStorage.getItem(tokenKey)
    if (!token) return false
    const payload = decodePayload(token)
    if (!payload?.exp) return false
    return payload.exp * 1000 > Date.now() + 10_000
  },
  set(token: string, refresh?: string, user?: SessionUser) {
    localStorage.setItem(tokenKey, token)
    if (refresh) localStorage.setItem(refreshKey, refresh)
    if (user) localStorage.setItem(userKey, JSON.stringify(user))
  },
  clear() {
    localStorage.removeItem(tokenKey)
    localStorage.removeItem(refreshKey)
    localStorage.removeItem(userKey)
    window.dispatchEvent(new Event('gopenid:logout'))
  },
}

async function request<T>(url: string, init: RequestInit = {}): Promise<T> {
  const guarded = url.startsWith('/api/admin') || url.startsWith('/api/me')
  if (guarded && !auth.isTokenValid()) {
    auth.clear()
    throw new Error('Oturum süresi doldu. Lütfen tekrar giriş yapın.')
  }
  const res = await fetch(url, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      ...(auth.token ? { Authorization: `Bearer ${auth.token}` } : {}),
      ...init.headers,
    },
  })
  if (res.status === 401) auth.clear()
  if (!res.ok) throw new Error(extractError(await res.text()) || res.statusText)
  if (res.status === 204) return undefined as T
  return res.json() as Promise<T>
}

function extractError(body: string): string {
  try {
    const parsed = JSON.parse(body) as { error?: string }
    return parsed.error ?? body
  } catch {
    return body
  }
}

export async function login(email: string, password: string) {
  const data = await request<{ access_token: string; refresh_token: string; user: SessionUser }>(
    '/api/auth/login',
    { method: 'POST', body: JSON.stringify({ email, password }) },
  )
  auth.set(data.access_token, data.refresh_token, data.user)
}

export async function logout() {
  try {
    await request<void>('/api/auth/logout', { method: 'POST' })
  } finally {
    auth.clear()
  }
}

export const me = {
  profile: () => request<User>('/api/me'),
  update: (body: { name: string; phone: string; title: string; avatarUrl: string }) =>
    request<User>('/api/me', { method: 'PUT', body: JSON.stringify(body) }),
  changePassword: (body: { currentPassword: string; newPassword: string }) =>
    request<{ message: string }>('/api/me/password', { method: 'POST', body: JSON.stringify(body) }),
  roles: () => request<Role[]>('/api/me/roles'),
  departments: () => request<Department[]>('/api/me/departments'),
  groups: () => request<Group[]>('/api/me/groups'),
  clients: () => request<Client[]>('/api/me/clients'),
  sessions: () => request<SessionInfo[]>('/api/me/sessions'),
}

export const api = {
  departments: crud<Department>('/api/admin/departments'),
  roles: crud<Role>('/api/admin/roles'),
  groups: crud<Group>('/api/admin/groups'),
  users: {
    ...crud<User>('/api/admin/users'),
    block: (id: number, reason: string) =>
      request<{ message: string }>(`/api/admin/users/${id}/block`, {
        method: 'POST',
        body: JSON.stringify({ reason }),
      }),
    unblock: (id: number) =>
      request<{ message: string }>(`/api/admin/users/${id}/unblock`, { method: 'POST' }),
    revokeSessions: (id: number) =>
      request<{ message: string }>(`/api/admin/users/${id}/revoke-sessions`, { method: 'POST' }),
  },
  clients: {
    ...crud<Client>('/api/admin/clients'),
    roles: {
      list: (clientId: number) => request<ClientRole[]>(`/api/admin/clients/${clientId}/roles`),
      create: (clientId: number, body: unknown) =>
        request<ClientRole>(`/api/admin/clients/${clientId}/roles`, {
          method: 'POST',
          body: JSON.stringify(body),
        }),
      update: (clientId: number, roleId: number, body: unknown) =>
        request<ClientRole>(`/api/admin/clients/${clientId}/roles/${roleId}`, {
          method: 'PUT',
          body: JSON.stringify(body),
        }),
      remove: (clientId: number, roleId: number) =>
        request<void>(`/api/admin/clients/${clientId}/roles/${roleId}`, { method: 'DELETE' }),
    },
  },
  policies: {
    ...crud<Policy>('/api/admin/policies'),
    assignments: {
      list: (policyId: number) =>
        request<PolicyAssignment[]>(`/api/admin/policies/${policyId}/assignments`),
      create: (policyId: number, body: { subjectType: string; subjectId: number }) =>
        request<PolicyAssignment>(`/api/admin/policies/${policyId}/assignments`, {
          method: 'POST',
          body: JSON.stringify(body),
        }),
      remove: (policyId: number, assignmentId: number) =>
        request<void>(`/api/admin/policies/${policyId}/assignments/${assignmentId}`, {
          method: 'DELETE',
        }),
    },
  },
  auditLogs: (params: { userId?: number; event?: string; limit?: number } = {}) => {
    const query = new URLSearchParams()
    if (params.userId) query.set('userId', String(params.userId))
    if (params.event) query.set('event', params.event)
    query.set('limit', String(params.limit ?? 100))
    return request<AuditLog[]>(`/api/admin/audit-logs?${query.toString()}`)
  },
}

function crud<T>(base: string) {
  return {
    list: () => request<T[]>(base),
    create: (body: unknown) => request<T>(base, { method: 'POST', body: JSON.stringify(body) }),
    update: (id: number, body: unknown) =>
      request<T>(`${base}/${id}`, { method: 'PUT', body: JSON.stringify(body) }),
    remove: (id: number) => request<void>(`${base}/${id}`, { method: 'DELETE' }),
  }
}

function decodePayload(token: string): { exp?: number } | null {
  try {
    const payload = token.split('.')[1]
    const json = atob(payload.replace(/-/g, '+').replace(/_/g, '/'))
    return JSON.parse(json) as { exp?: number }
  } catch {
    return null
  }
}
