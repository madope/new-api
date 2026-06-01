import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import { isRegisterEnabled } from './register-entry.ts'

describe('isRegisterEnabled', () => {
  test('defaults to true when status is missing', () => {
    assert.equal(isRegisterEnabled(undefined), true)
    assert.equal(isRegisterEnabled(null), true)
  })

  test('returns false when register_enabled is false', () => {
    assert.equal(isRegisterEnabled({ register_enabled: false }), false)
  })

  test('returns false when nested data.register_enabled is false', () => {
    assert.equal(
      isRegisterEnabled({ data: { register_enabled: false } }),
      false
    )
  })

  test('supports string values from cached payloads', () => {
    assert.equal(isRegisterEnabled({ register_enabled: 'false' }), false)
    assert.equal(isRegisterEnabled({ register_enabled: 'true' }), true)
  })
})
