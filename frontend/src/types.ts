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

export type ClientRole = Base & {
  clientId: number
  name: string
  description: string
}

export type Client = Base & {
  clientId: string
  clientSecret: string
  name: string
  redirectUris: string
  roles?: ClientRole[]
}

export type User = Base & {
  email: string
  name: string
  active: boolean
  departmentId?: number
  department?: Department
  roles: Role[]
  authorizedClients?: Client[]
  clientRoles?: ClientRole[]
}

export type Resource = Department | Role
