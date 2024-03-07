// darkTint returns a dark shade of the given #rrggbb color (30% of the color
// blended with black) for use as a pill background behind colored text.
// Returns an empty string for unparsable values.
export function darkTint(hexColor) {
  const match = /^#([0-9a-fA-F]{6})$/.exec(hexColor || '')
  if (!match) {
    return ''
  }
  const value = parseInt(match[1], 16)
  const r = Math.round(((value >> 16) & 0xff) * 0.3)
  const g = Math.round(((value >> 8) & 0xff) * 0.3)
  const b = Math.round((value & 0xff) * 0.3)
  return `#${((r << 16) | (g << 8) | b).toString(16).padStart(6, '0')}`
}

// readableOnDark returns the given #rrggbb color, lightened when necessary so
// it stays readable on the dark chip background. Returns an empty string for
// unparsable values so callers can fall back to CSS defaults.
export function readableOnDark(hexColor) {
  const match = /^#([0-9a-fA-F]{6})$/.exec(hexColor || '')
  if (!match) {
    return ''
  }
  const value = parseInt(match[1], 16)
  let r = (value >> 16) & 0xff
  let g = (value >> 8) & 0xff
  let b = value & 0xff

  // YIQ perceived brightness; bright colors pass through unchanged
  if ((r * 299 + g * 587 + b * 114) / 1000 >= 100) {
    return hexColor
  }

  // Blend toward white until readable
  r = Math.round(r + (255 - r) * 0.6)
  g = Math.round(g + (255 - g) * 0.6)
  b = Math.round(b + (255 - b) * 0.6)
  return `#${((r << 16) | (g << 8) | b).toString(16).padStart(6, '0')}`
}
