import type { SystemStatus } from '@/features/auth/types'

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

export function isNoticeButtonEnabled(
  status?: SystemStatus | Record<string, unknown> | null
): boolean {
  const directValue = status?.notice_button_enabled
  const nestedValue =
    status && 'data' in status && status.data && typeof status.data === 'object'
      ? (status.data as Record<string, unknown>).notice_button_enabled
      : undefined

  return toBooleanWithDefault(directValue ?? nestedValue, true)
}
