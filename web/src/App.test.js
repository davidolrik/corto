import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, fireEvent, within } from '@testing-library/svelte'
import App from './App.svelte'
import { auth } from './lib/auth.svelte.js'

function stubFetch() {
  vi.stubGlobal(
    'fetch',
    vi.fn(async (url) => ({
      ok: true,
      status: 200,
      json: async () => {
        if (url.includes('/api/version')) {
          return { version: '1.2.3' }
        }
        if (url.includes('/api/stats')) {
          return { links: 0, domains: 0, tags: 0, visits: 0, visits_this_week: 0, visits_by_country: {} }
        }
        return []
      },
    }))
  )
}

describe('App', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.unstubAllGlobals()
    stubFetch()
    auth.token = null
    auth.username = ''
    auth.tenantName = ''
    auth.tenantSlug = ''
    auth.tenants = []
  })

  it('labels the profile link with the username', async () => {
    auth.token = 'v4.public.token'
    auth.username = 'mandse'
    auth.tenantName = 'Olrik Links'

    render(App)

    const profileLink = screen.getByRole('link', { name: 'mandse' })
    expect(profileLink.getAttribute('href')).toBe('#/profile')
  })

  it('shows a footer with the version, also on the login screen', async () => {
    render(App)

    expect(await screen.findByText('Corto v1.2.3')).toBeTruthy()
  })

  it('brands the nav with only the link emoji', async () => {
    auth.token = 'v4.public.token'
    auth.tenantName = 'Olrik Links'

    render(App)

    const brand = screen.getByLabelText('Corto')
    expect(brand.textContent.trim()).toBe('🔗')
  })

  it('shows the tenant slug as plain text for a single tenant', async () => {
    auth.token = 'v4.public.token'
    auth.username = 'mandse'
    auth.tenantName = 'Olrik Links'
    auth.tenantSlug = 'olrik-links'
    auth.tenants = [{ slug: 'olrik-links', name: 'Olrik Links', is_admin: true }]

    render(App)

    expect(screen.queryByLabelText('Switch tenant')).toBe(null)
    expect(screen.getByText('olrik-links')).toBeTruthy()
  })

  it('switches tenants from the dropdown next to the username', async () => {
    auth.token = 'v4.public.token'
    auth.username = 'mandse'
    auth.tenantName = 'Olrik Links'
    auth.tenantSlug = 'olrik-links'
    auth.tenants = [
      { slug: 'olrik-links', name: 'Olrik Links', is_admin: true },
      { slug: 'acme', name: 'Acme', is_admin: false },
    ]
    const fetch = vi.fn(async (url) => ({
      ok: true,
      status: 200,
      json: async () => {
        if (url.includes('/api/auth/tenant')) {
          return {
            token: 'v4.public.acme',
            tenant_slug: 'acme',
            tenant_name: 'Acme',
            tenants: auth.tenants,
          }
        }
        if (url.includes('/api/version')) {
          return { version: '1.2.3' }
        }
        return { links: 0, domains: 0, tags: 0, visits: 0, visits_this_week: 0, visits_by_country: {} }
      },
    }))
    vi.stubGlobal('fetch', fetch)

    render(App)

    // The switcher is the tenant slug shown before the profile link, as
    // "olrik-links / mandse"
    const summary = screen.getByLabelText('Switch tenant')
    expect(summary.textContent.trim()).toBe('olrik-links')
    const userArea = summary.closest('.user-area')
    expect(userArea).not.toBe(null)
    expect(within(userArea).getByText('/')).toBeTruthy()
    expect(within(userArea).getByRole('link', { name: 'mandse' })).toBeTruthy()

    await fireEvent.click(screen.getByRole('button', { name: 'acme' }))

    const switchCall = fetch.mock.calls.find(([url]) => url.includes('/api/auth/tenant'))
    expect(switchCall).toBeTruthy()
    expect(JSON.parse(switchCall[1].body)).toEqual({ tenant: 'acme' })
    expect(auth.tenantSlug).toBe('acme')
    expect(auth.tenantName).toBe('Acme')
  })
})
