import assert from 'node:assert/strict';
import { describe, test } from 'node:test';

import { isRegisterEnabled } from './register-entry.js';

describe('isRegisterEnabled', () => {
  test('defaults to true when status is missing', () => {
    assert.equal(isRegisterEnabled(undefined), true);
    assert.equal(isRegisterEnabled(null), true);
  });

  test('returns false when register_enabled is false', () => {
    assert.equal(isRegisterEnabled({ register_enabled: false }), false);
  });

  test('supports cached string values', () => {
    assert.equal(isRegisterEnabled({ register_enabled: 'false' }), false);
    assert.equal(isRegisterEnabled({ register_enabled: 'true' }), true);
  });
});
