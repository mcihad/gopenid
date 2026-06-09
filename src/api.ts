import type { Department, Role, User, Client, ClientRole } from './types'

const tokenKey = 'gopenid.token'

export const auth = {
  get token() {
    return localStorage.getItem(tokenKey) ?? ''
  },
  isTokenValid() {
    const token = localStorage.getItem(tokenKey)
    if (!token) return false
    const payload = decodePayload(token)
    if (!payload?.exp) return false
    return payload.exp * 1000 > Date.now() + 10_000
  },
  set(token: string) {
    localStorage.setItem(tokenKey, token)
  },
  clear() {
    localStorage.removeItem(tokenKey)
    window.dispatchEvent(new Event('gopenid:logout'))
  },
}

async function request<T>(url: string, init: RequestInit = {}): Promise<T> {
  if (url.startsWith('/api/admin') && !auth.isTokenValid()) {
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
  if (!res.ok) throw new Error((await res.text()) || res.statusText)
  if (res.status === 204) return undefined as T
  return res.json() as Promise<T>
}

export async function login(email: string, password: string) {
  const data = await request<{ access_token: string }>('/api/auth/login', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  })
  auth.set(data.access_token)
}

export const api = {
  departments: crud<Department>('/api/admin/departments'),
  roles: crud<Role>('/api/admin/roles'),
  users: {
    list: () => request<User[]>('/api/admin/users'),
    create: (body: unknown) =>
      request<User>('/api/admin/users', { method: 'POST', body: JSON.stringify(body) }),
    update: (id: number, body: unknown) =>
      request<User>(`/api/admin/users/${id}`, { method: 'PUT', body: JSON.stringify(body) }),
    remove: (id: number) => request<void>(`/api/admin/users/${id}`, { method: 'DELETE' }),
  },
  clients: {
    ...crud<Client>('/api/admin/clients'),
    roles: {
      list: (clientId: number) => request<ClientRole[]>(`/api/admin/clients/${clientId}/roles`),
      create: (clientId: number, body: unknown) => request<ClientRole>(`/api/admin/clients/${clientId}/roles`, { method: 'POST', body: JSON.stringify(body) }),
      update: (clientId: number, roleId: number, body: unknown) => request<ClientRole>(`/api/admin/clients/${clientId}/roles/${roleId}`, { method: 'PUT', body: JSON.stringify(body) }),
      remove: (clientId: number, roleId: number) => request<void>(`/api/admin/clients/${clientId}/roles/${roleId}`, { method: 'DELETE' }),
    }
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
