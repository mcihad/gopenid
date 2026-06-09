export type Base = {
  ID: number
  CreatedAt: string
  UpdatedAt: string
}

export type Department = Base & {
  name: string
  description: string
}

export type Role = Base & {
  name: string
  description: string
}

export type Group = Base & {
  name: string
  description: string
}

export type ClientRole = Base & {
  clientId: number
  name: string
  description: string
}

export type Client = Base & {
  clientId: string
  clientSecret: string
  name: string
  description: string
  homeUrl: string
  logoUrl: string
  redirectUris: string
  tokenTtlSeconds: number
  refreshTtlSeconds: number
  roles?: ClientRole[]
}

export type User = Base & {
  email: string
  name: string
  active: boolean
  blocked: boolean
  blockedReason: string
  phone: string
  title: string
  avatarUrl: string
  lastLoginAt?: string | null
  departmentId?: number
  department?: Department
  departments?: Department[]
  groups?: Group[]
  roles: Role[]
  authorizedClients?: Client[]
  clientRoles?: ClientRole[]
}

export type PolicyType = 'ip' | 'time'
export type PolicyEffect = 'allow' | 'deny'
export type PolicySubject = 'client' | 'group' | 'user'

export type Policy = Base & {
  name: string
  description: string
  type: PolicyType
  effect: PolicyEffect
  ipCidrs: string
  daysOfWeek: number[] | null
  startTime: string
  endTime: string
}

export type PolicyAssignment = Base & {
  policyId: number
  subjectType: PolicySubject
  subjectId: number
}

export type AuditLog = Base & {
  userId?: number | null
  email: string
  clientId: string
  event: string
  success: boolean
  message: string
  ip: string
  userAgent: string
  device: string
  browser: string
  os: string
}

export type SessionInfo = Base & {
  userId: number
  clientId: string
  scope: string
  expiresAt: string
  revoked: boolean
  revokedAt?: string | null
}

export type Resource = Department | Role | Group
