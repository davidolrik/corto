const TOKEN_KEY = 'corto_token'

export class ApiError extends Error {
  constructor(message, status) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

export function getToken() {
  return localStorage.getItem(TOKEN_KEY)
}

export function setToken(token) {
  localStorage.setItem(TOKEN_KEY, token)
}

export function clearToken() {
  localStorage.removeItem(TOKEN_KEY)
}

// api performs an authenticated JSON request against the corto API. A 401
// clears the stored token and emits 'corto:unauthorized' so the app can
// return to the login screen.
export async function api(path, { method = 'GET', body } = {}) {
  const headers = {}
  const token = getToken()
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }
  if (body !== undefined) {
    headers['Content-Type'] = 'application/json'
  }

  const response = await fetch(path, {
    method,
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })

  if (response.status === 401) {
    clearToken()
    window.dispatchEvent(new CustomEvent('corto:unauthorized'))
  }

  if (!response.ok) {
    let message = `request failed with status ${response.status}`
    try {
      const problem = await response.json()
      message = problem.detail || problem.title || message
    } catch {
      // Not a JSON problem response, keep the generic message
    }
    throw new ApiError(message, response.status)
  }

  if (response.status === 204) {
    return null
  }
  return response.json()
}

export async function login(username, password) {
  const result = await api('/api/auth/login', {
    method: 'POST',
    body: { username, password },
  })
  setToken(result.token)
  return result
}
