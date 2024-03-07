import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import Dashboard from './Dashboard.svelte'
import { auth } from '../auth.svelte.js'

const stats = {
  links: 3,
  domains: 2,
  tags: 4,
  visits: 120,
  visits_this_week: 17,
  visits_by_country: { DK: 100, GB: 15, unknown: 5 },
}

function stubFetch() {
  vi.stubGlobal(
    'fetch',
    vi.fn(async () => ({
      ok: true,
      status: 200,
      json: async () => stats,
    }))
  )
}

describe('Dashboard', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.unstubAllGlobals()
    stubFetch()
    auth.tenantName = 'Olrik Links'
  })

  it('shows the tenant name as the page title', async () => {
    render(Dashboard)

    expect(await screen.findByRole('heading', { name: 'Olrik Links' })).toBeTruthy()
  })

  it('shows the stat cards', async () => {
    render(Dashboard)

    await screen.findByText('120')
    expect(screen.getByText('17')).toBeTruthy()
    expect(screen.getByText('3')).toBeTruthy()
    expect(screen.getByText('2')).toBeTruthy()
    expect(screen.getByText('4')).toBeTruthy()
  })

  it('paints visited countries on the world map', async () => {
    const { container } = render(Dashboard)

    await screen.findByText('120')

    // The map component is lazy-loaded
    const map = await screen.findByRole('img', { name: 'Visits by country' })
    expect(map).toBeTruthy()

    const visited = container.querySelectorAll('path.visited')
    expect(visited.length).toBe(2) // DK and GB; "unknown" is not on the map

    const denmark = [...container.querySelectorAll('path')].find((p) =>
      p.getAttribute('aria-label')?.startsWith('Denmark')
    )
    expect(denmark.getAttribute('aria-label')).toBe('Denmark: 100 visits')
    expect(denmark.classList.contains('visited')).toBe(true)
  })

  it('shows an html tooltip with flag and stats on hover', async () => {
    const { container } = render(Dashboard)

    await screen.findByText('120')
    await screen.findByRole('img', { name: 'Visits by country' })

    const denmark = [...container.querySelectorAll('path')].find((p) =>
      p.getAttribute('aria-label')?.startsWith('Denmark')
    )

    await fireEvent.mouseEnter(denmark, { clientX: 50, clientY: 60 })

    const tooltip = container.querySelector('.map-tooltip')
    expect(tooltip).not.toBe(null)
    expect(tooltip.textContent).toContain('🇩🇰')
    expect(tooltip.textContent).toContain('Denmark')
    // 100 of 120 total visits
    expect(tooltip.textContent).toContain('100 visits · 83%')

    await fireEvent.mouseLeave(denmark)
    expect(container.querySelector('.map-tooltip')).toBe(null)
  })
})
