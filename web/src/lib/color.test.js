import { describe, it, expect } from 'vitest'
import { readableOnDark, darkTint } from './color.js'

describe('darkTint', () => {
  it('darkens a color toward black for use as a pill background', () => {
    expect(darkTint('#22d3ee')).toBe('#0a3f47') // cyan, 30% kept
    expect(darkTint('#ff6600')).toBe('#4d1f00') // orange
    expect(darkTint('#000000')).toBe('#000000')
  })

  it('returns empty for unparsable values', () => {
    expect(darkTint('')).toBe('')
    expect(darkTint('orange')).toBe('')
  })
})

describe('readableOnDark', () => {
  it('keeps colors that are bright enough for a dark background', () => {
    expect(readableOnDark('#ffeb3b')).toBe('#ffeb3b') // yellow
    expect(readableOnDark('#ff6600')).toBe('#ff6600') // orange
    expect(readableOnDark('#22d3ee')).toBe('#22d3ee') // cyan
  })

  it('lightens colors that would vanish on a dark background', () => {
    const lightened = readableOnDark('#1a1a66') // dark navy
    expect(lightened).not.toBe('#1a1a66')
    expect(lightened).toMatch(/^#[0-9a-f]{6}$/)

    // The result must be bright enough to read
    const value = parseInt(lightened.slice(1), 16)
    const r = (value >> 16) & 0xff
    const g = (value >> 8) & 0xff
    const b = value & 0xff
    expect((r * 299 + g * 587 + b * 114) / 1000).toBeGreaterThanOrEqual(100)
  })

  it('returns empty for unparsable values', () => {
    expect(readableOnDark('')).toBe('')
    expect(readableOnDark('orange')).toBe('')
  })
})
