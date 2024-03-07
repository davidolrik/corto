import { describe, it, expect, beforeEach, vi } from 'vitest'
import { api, login, getToken, setToken, clearToken, ApiError } from './api.js'

function mockFetch(status, body) {
  return vi.fn(async () => ({
    ok: status >= 200 && status < 300,
    status,
    json: async () => body,
  }))
}

describe('token storage', () => {
  beforeEach(() => localStorage.clear())

  it('round-trips the token through localStorage', () => {
    expect(getToken()).toBe(null)
    setToken('v4.public.token')
    expect(getToken()).toBe('v4.public.token')
    clearToken()
    expect(getToken()).toBe(null)
  })
})

describe('api', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.unstubAllGlobals()
  })

  it('sends the Authorization header when a token is set', async () => {
    const fetch = mockFetch(200, [])
    vi.stubGlobal('fetch', fetch)
    setToken('v4.public.token')

    await api('/api/domains')

    const [url, options] = fetch.mock.calls[0]
    expect(url).toBe('/api/domains')
    expect(options.headers['Authorization']).toBe('Bearer v4.public.token')
  })

  it('sends JSON bodies with content type', async () => {
    const fetch = mockFetch(201, {})
    vi.stubGlobal('fetch', fetch)

    await api('/api/domains', { method: 'POST', body: { fqdn: 'go.example.com' } })

    const [, options] = fetch.mock.calls[0]
    expect(options.method).toBe('POST')
    expect(options.headers['Content-Type']).toBe('application/json')
    expect(JSON.parse(options.body)).toEqual({ fqdn: 'go.example.com' })
  })

  it('returns parsed JSON on success', async () => {
    vi.stubGlobal('fetch', mockFetch(200, [{ slug: 'promo' }]))
    expect(await api('/api/short-codes')).toEqual([{ slug: 'promo' }])
  })

  it('returns null for 204 responses', async () => {
    vi.stubGlobal('fetch', mockFetch(204, null))
    expect(await api('/api/domains/x', { method: 'DELETE' })).toBe(null)
  })

  it('throws ApiError with the server detail on failure', async () => {
    vi.stubGlobal('fetch', mockFetch(500, { title: 'Internal Server Error', detail: 'boom' }))
    await expect(api('/api/domains')).rejects.toThrow('boom')
    await expect(api('/api/domains')).rejects.toBeInstanceOf(ApiError)
  })

  it('clears the token and emits an event on 401', async () => {
    vi.stubGlobal('fetch', mockFetch(401, { title: 'Unauthorized' }))
    setToken('v4.public.token')
    const unauthorized = vi.fn()
    window.addEventListener('corto:unauthorized', unauthorized)

    await expect(api('/api/domains')).rejects.toThrow()

    expect(getToken()).toBe(null)
    expect(unauthorized).toHaveBeenCalled()
  })
})

describe('login', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.unstubAllGlobals()
  })

  it('posts credentials and stores the returned token', async () => {
    const fetch = mockFetch(200, { token: 'v4.public.fresh', is_admin: true })
    vi.stubGlobal('fetch', fetch)

    const result = await login('mandse', 'secret')

    const [url, options] = fetch.mock.calls[0]
    expect(url).toBe('/api/auth/login')
    expect(JSON.parse(options.body)).toEqual({
      username: 'mandse',
      password: 'secret',
    })
    expect(result.is_admin).toBe(true)
    expect(getToken()).toBe('v4.public.fresh')
  })
})
