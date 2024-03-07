import { api, getToken, setToken, clearToken } from './api.js'

const USERNAME_KEY = 'corto_username'
const TENANT_NAME_KEY = 'corto_tenant_name'
const TENANT_SLUG_KEY = 'corto_tenant_slug'
const TENANTS_KEY = 'corto_tenants'

function storedTenants() {
  try {
    return JSON.parse(localStorage.getItem(TENANTS_KEY) || '[]')
  } catch {
    return []
  }
}

export const auth = $state({
  token: getToken(),
  username: localStorage.getItem(USERNAME_KEY) || '',
  tenantName: localStorage.getItem(TENANT_NAME_KEY) || '',
  tenantSlug: localStorage.getItem(TENANT_SLUG_KEY) || '',
  tenants: storedTenants(),
})

// setAuthenticated stores a login or tenant switch result.
export function setAuthenticated(result) {
  setToken(result.token)
  auth.token = result.token
  auth.username = result.username || ''
  auth.tenantName = result.tenant_name || ''
  auth.tenantSlug = result.tenant_slug || ''
  auth.tenants = result.tenants || []
  localStorage.setItem(USERNAME_KEY, auth.username)
  localStorage.setItem(TENANT_NAME_KEY, auth.tenantName)
  localStorage.setItem(TENANT_SLUG_KEY, auth.tenantSlug)
  localStorage.setItem(TENANTS_KEY, JSON.stringify(auth.tenants))
}

// switchTenant exchanges the token for one bound to another tenant.
export async function switchTenant(slug) {
  const result = await api('/api/auth/tenant', { method: 'POST', body: { tenant: slug } })
  setAuthenticated(result)
}

export function logout() {
  clearToken()
  localStorage.removeItem(USERNAME_KEY)
  localStorage.removeItem(TENANT_NAME_KEY)
  localStorage.removeItem(TENANT_SLUG_KEY)
  localStorage.removeItem(TENANTS_KEY)
  auth.token = null
  auth.username = ''
  auth.tenantName = ''
  auth.tenantSlug = ''
  auth.tenants = []
}

// The API client clears the stored token on 401; drop the in-memory copy too
// so the app returns to the login screen.
window.addEventListener('corto:unauthorized', () => {
  auth.token = null
})
