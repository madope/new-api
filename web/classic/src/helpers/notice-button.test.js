import assert from 'node:assert/strict';
import { describe, test } from 'node:test';

import { isNoticeButtonEnabled } from './notice-button.js';

describe('isNoticeButtonEnabled', () => {
  test('defaults to true when status is missing', () => {
    assert.equal(isNoticeButtonEnabled(undefined), true);
    assert.equal(isNoticeButtonEnabled(null), true);
  });

  test('returns false when notice_button_enabled is false', () => {
    assert.equal(isNoticeButtonEnabled({ notice_button_enabled: false }), false);
  });

  test('returns false when nested data.notice_button_enabled is false', () => {
    assert.equal(
      isNoticeButtonEnabled({ data: { notice_button_enabled: false } }),
      false,
    );
  });

  test('supports cached string values', () => {
    assert.equal(isNoticeButtonEnabled({ notice_button_enabled: 'false' }), false);
    assert.equal(isNoticeButtonEnabled({ notice_button_enabled: 'true' }), true);
  });
});
