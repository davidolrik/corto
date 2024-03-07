import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import Login from './Login.svelte'
import { auth } from '../auth.svelte.js'
import { getToken } from '../api.js'

async function fillAndSubmit() {
  await fireEvent.input(screen.getByLabelText('Username'), { target: { value: 'mandse' } })
  await fireEvent.input(screen.getByLabelText('Password'), { target: { value: 'secret' } })
  await fireEvent.click(screen.getByRole('button', { name: 'Log in' }))
}

describe('Login', () => {
  beforeEach(() => {
    localStorage.clear()
    auth.token = null
    auth.tenants = []
    vi.unstubAllGlobals()
  })

  it('has no tenant field', () => {
    render(Login)

    expect(screen.queryByLabelText('Tenant ID')).toBe(null)
  })

  it('authenticates and stores token, tenant, and memberships', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => ({
        ok: true,
        status: 200,
        json: async () => ({
          token: 'v4.public.fresh',
          username: 'mandse',
          tenant_slug: 'olrik-links',
          tenant_name: 'Olrik Links',
          tenants: [
            { slug: 'olrik-links', name: 'Olrik Links', is_admin: true },
            { slug: 'other', name: 'Other', is_admin: false },
          ],
        }),
      }))
    )

    render(Login)
    await fillAndSubmit()

    expect(getToken()).toBe('v4.public.fresh')
    expect(auth.token).toBe('v4.public.fresh')
    expect(auth.username).toBe('mandse')
    expect(auth.tenantName).toBe('Olrik Links')
    expect(auth.tenantSlug).toBe('olrik-links')
    expect(auth.tenants.length).toBe(2)
    expect(localStorage.getItem('corto_tenant_slug')).toBe('olrik-links')
  })

  it('shows the server error on failed login', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => ({
        ok: false,
        status: 401,
        json: async () => ({ title: 'Unauthorized', detail: 'invalid credentials' }),
      }))
    )

    render(Login)
    await fillAndSubmit()

    expect(await screen.findByText('invalid credentials')).toBeTruthy()
    expect(auth.token).toBe(null)
  })
})
