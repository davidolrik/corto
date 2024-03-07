// countryFlag returns the flag emoji for a two letter ISO country code (any
// case), or an empty string for anything else (like the "unknown" bucket).
export function countryFlag(code) {
  if (!/^[A-Za-z]{2}$/.test(code || '')) {
    return ''
  }
  return [...code.toUpperCase()].map((c) => String.fromCodePoint(127397 + c.charCodeAt(0))).join('')
}
