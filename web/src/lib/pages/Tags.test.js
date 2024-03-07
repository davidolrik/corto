import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import Tags from './Tags.svelte'

const tagList = [
  {
    public_id: 'tag-1',
    slug: 'marketing',
    color: '#ff6600',
    description: 'Campaign links',
    created_at: '2026-06-12T10:00:00Z',
    updated_at: '2026-06-12T10:00:00Z',
  },
]

function stubFetch() {
  vi.stubGlobal(
    'fetch',
    vi.fn(async (url, options = {}) => ({
      ok: true,
      status: 200,
      json: async () => ((options.method || 'GET') === 'GET' ? tagList : {}),
    }))
  )
}

describe('Tags', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.unstubAllGlobals()
    stubFetch()
  })

  it('colors the left edge of the row with the tag color', async () => {
    const { container } = render(Tags)

    await screen.findByText('#marketing')

    expect(screen.getByText('Campaign links')).toBeTruthy()
    expect(container.querySelector('.tag-swatch')).toBe(null)

    const row = container.querySelector('.row')
    expect(row.classList.contains('tag-row')).toBe(true)
    expect(row.style.borderLeftColor).toBe('rgb(255, 102, 0)')
  })

  it('prefills color and description in the edit form', async () => {
    render(Tags)

    await screen.findByText('#marketing')
    await fireEvent.click(screen.getByRole('button', { name: 'Edit' }))

    expect(screen.getByLabelText('Color').value).toBe('#ff6600')
    expect(screen.getByLabelText('Description').value).toBe('Campaign links')
  })
})
