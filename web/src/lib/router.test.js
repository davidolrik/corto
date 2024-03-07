import { describe, it, expect, beforeEach } from 'vitest'
import { router, navigate } from './router.svelte.js'

function fireHashChange() {
  window.dispatchEvent(new HashChangeEvent('hashchange'))
}

describe('router', () => {
  beforeEach(() => {
    location.hash = ''
    fireHashChange()
  })

  it('defaults to the root path', () => {
    expect(router.path).toBe('/')
  })

  it('navigates by setting the hash', () => {
    navigate('/domains')
    fireHashChange()
    expect(location.hash).toBe('#/domains')
    expect(router.path).toBe('/domains')
  })

  it('tracks external hash changes', () => {
    location.hash = '#/tags'
    fireHashChange()
    expect(router.path).toBe('/tags')
  })
})
