import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import Domains from './Domains.svelte'

const domainList = [
  {
    public_id: 'dom-1',
    fqdn: 'go.example.com',
    fallback_url: 'https://example.com',
    description: 'Primary short link domain',
    created_at: '2026-06-11T10:00:00Z',
    updated_at: '2026-06-11T10:00:00Z',
  },
]

function stubFetch() {
  const fetch = vi.fn(async (url, options = {}) => ({
    ok: true,
    status: 200,
    json: async () => ((options.method || 'GET') === 'GET' ? domainList : {}),
  }))
  vi.stubGlobal('fetch', fetch)
  return fetch
}

describe('Domains', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.unstubAllGlobals()
  })

  it('shows the description in the list', async () => {
    stubFetch()
    render(Domains)

    await screen.findByText('go.example.com')

    expect(screen.getByText('Primary short link domain')).toBeTruthy()
  })

  it('opens the edit form in a modal from the edit button', async () => {
    stubFetch()
    render(Domains)

    await screen.findByText('go.example.com')
    await fireEvent.click(screen.getByRole('button', { name: 'Edit' }))

    const fqdnInput = screen.getByLabelText('Domain (FQDN)')
    expect(fqdnInput.value).toBe('go.example.com')
    expect(screen.getByLabelText('Description').value).toBe('Primary short link domain')

    const dialog = fqdnInput.closest('dialog')
    expect(dialog).not.toBe(null)
    expect(dialog.open).toBe(true)
  })

  it('closes the modal on cancel', async () => {
    stubFetch()
    render(Domains)

    await screen.findByText('go.example.com')
    await fireEvent.click(screen.getByRole('button', { name: 'Edit' }))
    await fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))

    expect(screen.queryByLabelText('Domain (FQDN)')).toBe(null)
  })

  it('does not open the edit form when delete is clicked', async () => {
    stubFetch()
    vi.stubGlobal('confirm', vi.fn(() => false))
    render(Domains)

    await screen.findByText(/go\.example\.com/)
    await fireEvent.click(screen.getByRole('button', { name: 'Delete' }))

    expect(screen.queryByLabelText('Domain (FQDN)')).toBe(null)
  })
})
