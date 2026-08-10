import { defineStore } from 'pinia'
import { adminAPI, type AdminAuthzPolicy, type AdminLoginRequest } from '@/api/admin'

const TOKEN_KEY = 'admin_token'
const ROLES_KEY = 'admin_roles'
const PERMISSIONS_KEY = 'admin_permissions'
const IS_SUPER_KEY = 'admin_is_super'

function readArrayStorage(key: string): string[] {
  const raw = localStorage.getItem(key)
  if (!raw) {
    return []
  }
  try {
    const parsed = JSON.parse(raw)
    if (Array.isArray(parsed)) {
      return parsed.filter((item): item is string => typeof item === 'string')
    }
  } catch {
    return []
  }
  return []
}

function normalizeObjectPath(path: string): string {
  const normalized = String(path || '').trim()
  if (!normalized) {
    return '/'
  }
  if (normalized.startsWith('/api/v1/')) {
    return normalized.replace('/api/v1', '')
  }
  if (normalized === '/api/v1') {
    return '/'
  }
  return normalized.startsWith('/') ? normalized : `/${normalized}`
}

function normalizePermissionKey(action: string, object: string): string {
  return `${String(action || '').trim().toUpperCase()}:${normalizeObjectPath(object)}`
}

function parsePermissionKey(permission: string): { action: string; object: string } {
  const splitIndex = permission.indexOf(':')
  if (splitIndex <= 0) {
    return { action: '*', object: '/' }
  }
  return {
    action: permission.slice(0, splitIndex).trim().toUpperCase(),
    object: normalizeObjectPath(permission.slice(splitIndex + 1)),
  }
}

function escapeRegex(input: string): string {
  return input.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

function matchObject(requiredObject: string, grantedObject: string): boolean {
  const required = normalizeObjectPath(requiredObject)
  const granted = normalizeObjectPath(grantedObject)
  if (granted === '*' || granted === '/*') {
    return true
  }
  const pattern = `^${escapeRegex(granted).replace(/\\\*/g, '.*').replace(/:[^/]+/g, '[^/]+')}$`
  return new RegExp(pattern).test(required)
}

function buildPermissionKeys(policies: AdminAuthzPolicy[]): string[] {
  const set = new Set<string>()
  policies.forEach((item) => {
    set.add(normalizePermissionKey(item.action, item.object))
  })
  return Array.from(set)
}

export const useAdminAuthStore = defineStore('adminAuth', {
  state: () => ({
    loading: false,
    token: localStorage.getItem(TOKEN_KEY) || '',
    isSuper: localStorage.getItem(IS_SUPER_KEY) === '1',
    roles: readArrayStorage(ROLES_KEY),
    permissions: readArrayStorage(PERMISSIONS_KEY),
    permissionsLoaded: false,
  }),
  actions: {
    async login(payload: AdminLoginRequest) {
      this.loading = true
      try {
        const response = await adminAPI.login(payload)
        const { token } = response.data.data
        this.token = token
        localStorage.setItem(TOKEN_KEY, token)
        await this.loadAuthz()
      } finally {
        this.loading = false
      }
    },

    async loadAuthz() {
      if (!this.token) {
        this.clearAuthzCache()
        return
      }
      const response = await adminAPI.getAuthzMe()
      const payload = response.data.data
      this.isSuper = Boolean(payload?.is_super)
      this.roles = Array.isArray(payload?.roles) ? payload.roles : []
      this.permissions = Array.isArray(payload?.policies) ? buildPermissionKeys(payload.policies) : []
      this.permissionsLoaded = true

      localStorage.setItem(IS_SUPER_KEY, this.isSuper ? '1' : '0')
      localStorage.setItem(ROLES_KEY, JSON.stringify(this.roles))
      localStorage.setItem(PERMISSIONS_KEY, JSON.stringify(this.permissions))
    },

    hasPermission(permission?: string) {
      if (!permission) {
        return true
      }
      if (this.isSuper) {
        return true
      }
      const required = parsePermissionKey(permission)
      return this.permissions.some((item) => {
        const granted = parsePermissionKey(item)
        const methodMatch = granted.action === '*' || granted.action === required.action
        if (!methodMatch) {
          return false
        }
        return matchObject(required.object, granted.object)
      })
    },

    clearAuthzCache() {
      this.isSuper = false
      this.roles = []
      this.permissions = []
      this.permissionsLoaded = false
      localStorage.removeItem(IS_SUPER_KEY)
      localStorage.removeItem(ROLES_KEY)
      localStorage.removeItem(PERMISSIONS_KEY)
    },

    logout() {
      this.token = ''
      localStorage.removeItem(TOKEN_KEY)
      this.clearAuthzCache()
    },
  },
})
