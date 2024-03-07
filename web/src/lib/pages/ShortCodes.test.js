import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, fireEvent, within } from '@testing-library/svelte'
import ShortCodes from './ShortCodes.svelte'

const shortCodeList = [
  {
    public_id: 'sc-1',
    slug: 'promo',
    title: 'Spring promo',
    description: 'Landing page for the spring campaign',
    target_url: 'https://example.com/landing',
    domains: ['go.example.com', 'links.example.com'],
    tags: ['marketing', 'stack'],
    visits: 42,
    visits_this_week: 5,
    visits_by_domain: { 'go.example.com': 30, 'links.example.com': 12 },
    visits_by_campaign: { direct: 2, spring: 40 },
    visits_by_country: { DK: 28, unknown: 5, GB: 9 },
    forward_query: false,
    is_crawlable: false,
    created_at: '2026-06-11T10:00:00Z',
    updated_at: '2026-06-11T10:00:00Z',
  },
]

const tagList = [
  { public_id: 'tag-1', slug: 'marketing', color: '#ff6600', description: '' },
  { public_id: 'tag-2', slug: 'stack', color: '#2a1045', description: '' },
]

function stubFetch() {
  vi.stubGlobal(
    'fetch',
    vi.fn(async (url) => ({
      ok: true,
      status: 200,
      json: async () => {
        if (url.includes('/api/short-codes')) {
          return shortCodeList
        }
        if (url.includes('/api/tags')) {
          return tagList
        }
        return []
      },
    }))
  )
}

async function expandStats() {
  const { container } = render(ShortCodes)
  await screen.findByText('Spring promo', { exact: false })
  await fireEvent.click(screen.getByRole('button', { name: /promo/ }))
  return container
}

describe('ShortCodes', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.unstubAllGlobals()
    stubFetch()
  })

  it('shows visits this week and in total for each link', async () => {
    render(ShortCodes)

    await screen.findByText('Spring promo', { exact: false })

    expect(screen.getByText('5')).toBeTruthy()
    expect(screen.getByText('this week')).toBeTruthy()
    expect(screen.getByText('42')).toBeTruthy()
    expect(screen.getByText('total')).toBeTruthy()
  })

  it('keeps rows to configuration chips only', async () => {
    render(ShortCodes)

    await screen.findByText('Spring promo', { exact: false })

    // Plain domain and tag chips, no per-domain counts and no stat chips
    const domainChip = screen.getByText('go.example.com')
    expect(domainChip.classList.contains('domain')).toBe(true)
    const tagChip = screen.getByText('#marketing')
    expect(tagChip).toBeTruthy()
    // Badge style: readable tag color on border and text over a dark tint of
    // the raw color; orange is bright enough to stay unchanged
    expect(tagChip.style.borderColor).toBe('rgb(255, 102, 0)')
    expect(tagChip.style.color).toBe('rgb(255, 102, 0)')
    expect(tagChip.style.backgroundColor).toBe('rgb(77, 31, 0)')

    // A dark tag color gets lightened on border and text so both stay visible
    const darkChip = screen.getByText('#stack')
    expect(darkChip.style.borderColor).toBe('rgb(170, 159, 181)')
    expect(darkChip.style.color).toBe('rgb(170, 159, 181)')
    expect(darkChip.style.backgroundColor).toBe('rgb(13, 5, 21)')
    expect(screen.queryByText(/go\.example\.com · 30/)).toBe(null)
    expect(screen.queryByText(/spring · 40/)).toBe(null)
    expect(screen.queryByText(/DK · 28/)).toBe(null)
  })

  it('expands the row into a map and stats sidebar inside the row', async () => {
    const container = await expandStats()

    // The per-link world map is lazy-loaded into the expansion
    expect(await screen.findByRole('img', { name: 'Visits by country' })).toBeTruthy()

    // The expansion lives inside the row card, which is highlighted
    const row = container.querySelector('.row')
    expect(row.querySelector('.row-stats')).not.toBe(null)
    expect(row.classList.contains('expanded')).toBe(true)

    const sidebar = within(container.querySelector('.row-stats-sidebar'))
    // Domains
    expect(sidebar.getByText('go.example.com')).toBeTruthy()
    expect(sidebar.getByText('30')).toBeTruthy()
    // Campaigns as plain labels, biggest first
    const spring = sidebar.getByText('spring')
    const direct = sidebar.getByText('direct')
    expect(spring.classList.contains('chip')).toBe(false)
    expect(spring.compareDocumentPosition(direct) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
    // Countries with flags, unknown without
    expect(sidebar.getByText('🇩🇰 DK')).toBeTruthy()
    expect(sidebar.getByText('🇬🇧 GB')).toBeTruthy()
    expect(sidebar.getByText('unknown')).toBeTruthy()
  })

  it('collapses the stats on a second click', async () => {
    const container = await expandStats()
    await fireEvent.click(screen.getByRole('button', { name: /promo/ }))

    expect(container.querySelector('.row-stats')).toBe(null)
  })

  it('does not open the edit form when the row is clicked', async () => {
    await expandStats()

    expect(screen.queryByLabelText('Slug')).toBe(null)
  })

  it('does not collapse when clicking inside the expansion', async () => {
    const container = await expandStats()
    await screen.findByRole('img', { name: 'Visits by country' })

    await fireEvent.click(container.querySelector('.row-stats-sidebar'))

    expect(container.querySelector('.row-stats')).not.toBe(null)
  })

  it('copies the domain specific link from its pill without expanding the row', async () => {
    const writeText = vi.fn(async () => {})
    Object.defineProperty(navigator, 'clipboard', { value: { writeText }, configurable: true })

    const { container } = render(ShortCodes)
    await screen.findByText('Spring promo', { exact: false })

    await fireEvent.click(screen.getByRole('button', { name: 'go.example.com' }))
    expect(writeText).toHaveBeenCalledWith('https://go.example.com/promo')
    expect(screen.getByRole('button', { name: 'Copied' })).toBeTruthy()

    await fireEvent.click(screen.getByRole('button', { name: 'links.example.com' }))
    expect(writeText).toHaveBeenCalledWith('https://links.example.com/promo')

    expect(container.querySelector('.row-stats')).toBe(null)
    expect(screen.queryByRole('button', { name: 'Copy' })).toBe(null)
  })

  it('shows the description on the row', async () => {
    render(ShortCodes)

    await screen.findByText('Spring promo', { exact: false })

    expect(screen.getByText('Landing page for the spring campaign')).toBeTruthy()
  })

  it('opens the edit modal from the edit button without expanding the row', async () => {
    const { container } = render(ShortCodes)
    await screen.findByText('Spring promo', { exact: false })

    await fireEvent.click(screen.getByRole('button', { name: 'Edit' }))

    const slugInput = screen.getByLabelText('Slug')
    expect(slugInput.value).toBe('promo')
    expect(screen.getByLabelText('Description').value).toBe('Landing page for the spring campaign')
    expect(container.querySelector('.row-stats')).toBe(null)
  })
})
