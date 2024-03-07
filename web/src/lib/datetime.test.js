import { describe, it, expect } from 'vitest'
import { toDateTimeInput, fromDateTimeInput } from './datetime.js'

describe('toDateTimeInput', () => {
  it('returns empty string for empty values', () => {
    expect(toDateTimeInput(null)).toBe('')
    expect(toDateTimeInput(undefined)).toBe('')
    expect(toDateTimeInput('')).toBe('')
  })

  it('formats an ISO timestamp as a local datetime-local value', () => {
    const iso = new Date(2026, 5, 11, 14, 30).toISOString()
    expect(toDateTimeInput(iso)).toBe('2026-06-11T14:30')
  })
})

describe('fromDateTimeInput', () => {
  it('returns undefined for empty values', () => {
    expect(fromDateTimeInput('')).toBe(undefined)
    expect(fromDateTimeInput(null)).toBe(undefined)
  })

  it('converts a datetime-local value to an ISO timestamp', () => {
    const iso = fromDateTimeInput('2026-06-11T14:30')
    expect(new Date(iso).getTime()).toBe(new Date(2026, 5, 11, 14, 30).getTime())
  })

  it('round-trips with toDateTimeInput', () => {
    expect(toDateTimeInput(fromDateTimeInput('2026-06-11T14:30'))).toBe('2026-06-11T14:30')
  })
})
