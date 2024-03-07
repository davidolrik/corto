import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import Profile from './Profile.svelte'

function stubFetch({ passwordStatus = 204 } = {}) {
  const fetch = vi.fn(async (url, options = {}) => {
    if (url.includes('/api/profile/password')) {
      return {
        ok: passwordStatus < 400,
        status: passwordStatus,
        json: async () => ({ title: 'Forbidden', detail: 'current password is incorrect' }),
      }
    }
    return {
      ok: true,
      status: 200,
      json: async () => ({ user_id: 'user-1', username: 'mandse' }),
    }
  })
  vi.stubGlobal('fetch', fetch)
  return fetch
}

async function fillAndSubmit() {
  await fireEvent.input(screen.getByLabelText('Current password'), { target: { value: 'old-password' } })
  await fireEvent.input(screen.getByLabelText('New password'), { target: { value: 'brand-new-password' } })
  await fireEvent.click(screen.getByRole('button', { name: 'Change password' }))
}

describe('Profile', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.unstubAllGlobals()
  })

  it('shows the username', async () => {
    stubFetch()
    render(Profile)

    expect(await screen.findByText('mandse')).toBeTruthy()
  })

  it('changes the password and confirms', async () => {
    const fetch = stubFetch()
    render(Profile)
    await screen.findByText('mandse')

    await fillAndSubmit()

    const call = fetch.mock.calls.find(([url]) => url.includes('/api/profile/password'))
    expect(call[1].method).toBe('PUT')
    expect(JSON.parse(call[1].body)).toEqual({
      current_password: 'old-password',
      new_password: 'brand-new-password',
    })
    expect(await screen.findByText('Password changed')).toBeTruthy()
  })

  it('shows the server error for a wrong current password', async () => {
    stubFetch({ passwordStatus: 403 })
    render(Profile)
    await screen.findByText('mandse')

    await fillAndSubmit()

    expect(await screen.findByText('current password is incorrect')).toBeTruthy()
  })
})
