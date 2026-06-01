import type { SystemStatus } from '../types'

function toBooleanWithDefault(value: unknown, fallback: boolean): boolean {
  if (value === undefined || value === null) return fallback
  if (typeof value === 'boolean') return value
  if (typeof value === 'number') return value !== 0
  if (typeof value === 'string') {
    const normalized = value.trim().toLowerCase()
    if (normalized === 'true' || normalized === '1') return true
    if (normalized === 'false' || normalized === '0') return false
  }
  return fallback
}

export function isRegisterEnabled(status?: SystemStatus | Record<string, unknown> | null): boolean {
  const directValue = status?.register_enabled
  const nestedValue =
    status && 'data' in status && status.data && typeof status.data === 'object'
      ? (status.data as Record<string, unknown>).register_enabled
      : undefined

  return toBooleanWithDefault(directValue ?? nestedValue, true)
}

export function getCachedStatus(): Record<string, unknown> | null {
  try {
    if (typeof window === 'undefined') return null
    const saved = window.localStorage.getItem('status')
    return saved ? (JSON.parse(saved) as Record<string, unknown>) : null
  } catch {
    return null
  }
}
