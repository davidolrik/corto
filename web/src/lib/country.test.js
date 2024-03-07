import { describe, it, expect } from 'vitest'
import { countryFlag } from './country.js'

describe('countryFlag', () => {
  it('returns the flag emoji for a two letter country code', () => {
    expect(countryFlag('DK')).toBe('🇩🇰')
    expect(countryFlag('GB')).toBe('🇬🇧')
    expect(countryFlag('dk')).toBe('🇩🇰') // map location ids are lowercase
  })

  it('returns empty for anything that is not a country code', () => {
    expect(countryFlag('unknown')).toBe('')
    expect(countryFlag('')).toBe('')
    expect(countryFlag(undefined)).toBe('')
  })
})
