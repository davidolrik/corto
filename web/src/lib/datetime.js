// Conversions between API timestamps (RFC 3339) and the value format of
// <input type="datetime-local">, which is always local time.

const pad = (n) => String(n).padStart(2, '0')

export function toDateTimeInput(iso) {
  if (!iso) {
    return ''
  }
  const d = new Date(iso)
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

export function fromDateTimeInput(value) {
  if (!value) {
    return undefined
  }
  return new Date(value).toISOString()
}
