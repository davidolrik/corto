import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, fireEvent, within } from '@testing-library/svelte'
import ShortCodes from './ShortCodes.svelte'

const shortCodeList = [
  {
    public_id: 'sc-1',
    slug: 'promo',
    title: 'Spring promo',
    description: 'Landing page for the spring campaign',
    max_visits: 100,
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

function stubFetch(list = shortCodeList) {
  vi.stubGlobal(
    'fetch',
    vi.fn(async (url) => ({
      ok: true,
      status: 200,
      json: async () => {
        if (url.includes('/api/short-codes')) {
          return list
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

  it('shows visits this week and the total against the limit', async () => {
    render(ShortCodes)

    await screen.findByText('Spring promo', { exact: false })

    expect(screen.getByText('5')).toBeTruthy()
    expect(screen.getByText('this week')).toBeTruthy()
    const total = screen.getByText('42 / 100')
    expect(total).toBeTruthy()
    expect(total.classList.contains('exhausted')).toBe(false)
    expect(screen.getByText('total')).toBeTruthy()
  })

  it('shows a plain total for unlimited links', async () => {
    stubFetch([{ ...shortCodeList[0], max_visits: null }])
    render(ShortCodes)

    await screen.findByText('Spring promo', { exact: false })

    expect(screen.getByText('42')).toBeTruthy()
    expect(screen.queryByText('42 / 100')).toBe(null)
  })

  it('marks an exhausted link', async () => {
    stubFetch([{ ...shortCodeList[0], visits: 100 }])
    render(ShortCodes)

    await screen.findByText('Spring promo', { exact: false })

    const total = screen.getByText('100 / 100')
    expect(total.classList.contains('exhausted')).toBe(true)
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
    expect(screen.getByLabelText('Max visits').value).toBe('100')
    expect(container.querySelector('.row-stats')).toBe(null)
  })
})

// Four codes sharing domains and tags so cumulative AND filtering can be
// exercised. grepmaste.rs spans the first three; mand.se only two of them.
const filterList = [
  {
    ...shortCodeList[0],
    public_id: 'sc-1',
    slug: 'dns',
    title: 'DNS service',
    domains: ['grepmaste.rs', 'mand.se'],
    tags: ['affiliate'],
  },
  {
    ...shortCodeList[0],
    public_id: 'sc-2',
    slug: 'arc',
    title: 'Arc browser',
    domains: ['grepmaste.rs'],
    tags: ['stack'],
  },
  {
    ...shortCodeList[0],
    public_id: 'sc-3',
    slug: 'tower',
    title: 'Git Tower',
    domains: ['grepmaste.rs', 'mand.se'],
    tags: ['affiliate', 'stack'],
  },
  {
    ...shortCodeList[0],
    public_id: 'sc-4',
    slug: 'other',
    title: 'Other thing',
    domains: ['other.org'],
    tags: ['editorial'],
  },
]

// Set the typeahead box to the given query
function type(value) {
  return fireEvent.input(screen.getByLabelText('Filter links'), { target: { value } })
}

// Scope queries to the open suggestion menu (returns null when it is closed)
function menu() {
  const el = document.querySelector('.typeahead-menu')
  return el ? within(el) : null
}

describe('ShortCodes filter', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.unstubAllGlobals()
    stubFetch(filterList)
  })

  it('renders a filter input above the list', async () => {
    render(ShortCodes)
    await screen.findByText('DNS service', { exact: false })

    expect(screen.getByLabelText('Filter links')).toBeTruthy()
  })

  it('suggests domains and tags matching the typed text', async () => {
    render(ShortCodes)
    await screen.findByText('DNS service', { exact: false })

    await type('grep')
    expect(menu().getByRole('button', { name: 'grepmaste.rs' })).toBeTruthy()
    expect(menu().queryByRole('button', { name: 'other.org' })).toBe(null)

    await type('stac')
    expect(menu().getByRole('button', { name: '#stack' })).toBeTruthy()
  })

  it('filters the list and shows a token when a domain is selected', async () => {
    const { container } = render(ShortCodes)
    await screen.findByText('DNS service', { exact: false })

    await type('grep')
    await fireEvent.click(menu().getByRole('button', { name: 'grepmaste.rs' }))

    // rendered as a domain chip with a remove control
    expect(container.querySelector('.filter-tokens .chip.domain')).not.toBe(null)
    expect(screen.getByRole('button', { name: 'Remove domain grepmaste.rs' })).toBeTruthy()

    // the code on a different domain is gone, the rest remain
    expect(screen.getByText('DNS service', { exact: false })).toBeTruthy()
    expect(screen.getByText('Arc browser', { exact: false })).toBeTruthy()
    expect(screen.getByText('Git Tower', { exact: false })).toBeTruthy()
    expect(screen.queryByText('Other thing', { exact: false })).toBe(null)
  })

  it('ANDs every selected token, dropping codes missing any of them', async () => {
    render(ShortCodes)
    await screen.findByText('DNS service', { exact: false })

    await type('grep')
    await fireEvent.click(menu().getByRole('button', { name: 'grepmaste.rs' }))
    await type('affil')
    await fireEvent.click(menu().getByRole('button', { name: '#affiliate' }))
    await type('stac')
    await fireEvent.click(menu().getByRole('button', { name: '#stack' }))

    // only the code carrying grepmaste.rs AND affiliate AND stack survives
    expect(screen.getByText('Git Tower', { exact: false })).toBeTruthy()
    expect(screen.queryByText('DNS service', { exact: false })).toBe(null) // lacks #stack
    expect(screen.queryByText('Arc browser', { exact: false })).toBe(null) // lacks #affiliate
  })

  it('requires all selected domains (domain AND domain)', async () => {
    render(ShortCodes)
    await screen.findByText('DNS service', { exact: false })

    await type('grep')
    await fireEvent.click(menu().getByRole('button', { name: 'grepmaste.rs' }))
    expect(screen.getByText('Arc browser', { exact: false })).toBeTruthy()

    await type('mand')
    await fireEvent.click(menu().getByRole('button', { name: 'mand.se' }))

    // Arc lives only on grepmaste.rs, so requiring mand.se too removes it
    expect(screen.queryByText('Arc browser', { exact: false })).toBe(null)
    expect(screen.getByText('DNS service', { exact: false })).toBeTruthy()
    expect(screen.getByText('Git Tower', { exact: false })).toBeTruthy()
  })

  it('only suggests values reachable in the current filtered set', async () => {
    render(ShortCodes)
    await screen.findByText('DNS service', { exact: false })

    await type('grep')
    await fireEvent.click(menu().getByRole('button', { name: 'grepmaste.rs' }))

    // other.org belongs to an excluded code, so it is not offered
    await type('other')
    expect(menu()).toBe(null)

    // mand.se is on the filtered codes, so it is offered
    await type('mand')
    expect(menu().getByRole('button', { name: 'mand.se' })).toBeTruthy()
  })

  it('does not re-suggest an already selected domain', async () => {
    render(ShortCodes)
    await screen.findByText('DNS service', { exact: false })

    await type('grep')
    await fireEvent.click(menu().getByRole('button', { name: 'grepmaste.rs' }))

    await type('grep')
    expect(menu()).toBe(null)
  })

  it('removes a filter when its token close button is clicked', async () => {
    render(ShortCodes)
    await screen.findByText('DNS service', { exact: false })

    await type('grep')
    await fireEvent.click(menu().getByRole('button', { name: 'grepmaste.rs' }))
    expect(screen.queryByText('Other thing', { exact: false })).toBe(null)

    await fireEvent.click(screen.getByRole('button', { name: 'Remove domain grepmaste.rs' }))
    expect(screen.getByText('Other thing', { exact: false })).toBeTruthy()
  })

  it('adds the first suggestion on Enter', async () => {
    render(ShortCodes)
    await screen.findByText('DNS service', { exact: false })

    const input = screen.getByLabelText('Filter links')
    await type('grep')
    await fireEvent.keyDown(input, { key: 'Enter' })

    expect(screen.getByRole('button', { name: 'Remove domain grepmaste.rs' })).toBeTruthy()
    expect(screen.queryByText('Other thing', { exact: false })).toBe(null)
  })

  it('moves the highlight with the arrow keys and commits with Enter', async () => {
    render(ShortCodes)
    await screen.findByText('DNS service', { exact: false })

    const input = screen.getByLabelText('Filter links')
    // "st" matches the domain grepmaste.rs and the tag stack, in that order
    await type('st')
    await fireEvent.keyDown(input, { key: 'ArrowDown' }) // grepmaste.rs
    await fireEvent.keyDown(input, { key: 'ArrowDown' }) // #stack
    await fireEvent.keyDown(input, { key: 'Enter' })

    expect(screen.getByRole('button', { name: 'Remove tag stack' })).toBeTruthy()
  })

  it('commits the highlighted suggestion with Tab', async () => {
    render(ShortCodes)
    await screen.findByText('DNS service', { exact: false })

    const input = screen.getByLabelText('Filter links')
    await type('st')
    await fireEvent.keyDown(input, { key: 'ArrowDown' }) // grepmaste.rs
    await fireEvent.keyDown(input, { key: 'Tab' })

    expect(screen.getByRole('button', { name: 'Remove domain grepmaste.rs' })).toBeTruthy()
  })

  it('pops the last token on Backspace when the box is empty', async () => {
    render(ShortCodes)
    await screen.findByText('DNS service', { exact: false })

    const input = screen.getByLabelText('Filter links')
    await type('grep')
    await fireEvent.keyDown(input, { key: 'Enter' })
    expect(screen.getByRole('button', { name: 'Remove domain grepmaste.rs' })).toBeTruthy()

    await fireEvent.keyDown(input, { key: 'Backspace' })
    expect(screen.queryByRole('button', { name: 'Remove domain grepmaste.rs' })).toBe(null)
    expect(screen.getByText('Other thing', { exact: false })).toBeTruthy()
  })
})
