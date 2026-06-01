import assert from 'node:assert/strict';
import { describe, test } from 'node:test';

import { buildMainNavLinks } from './navigation-links.js';

describe('buildMainNavLinks', () => {
  const t = (key) => key;

  test('hides model marketplace before login when pricing requires auth', () => {
    const links = buildMainNavLinks({
      t,
      docsLink: '',
      headerNavModules: {
        home: true,
        console: true,
        pricing: { enabled: true, requireAuth: true },
        docs: true,
        about: true,
      },
      isAuthed: false,
    });

    assert.equal(links.some((link) => link.itemKey === 'pricing'), false);
  });

  test('shows model marketplace after login when pricing requires auth', () => {
    const links = buildMainNavLinks({
      t,
      docsLink: '',
      headerNavModules: {
        pricing: { enabled: true, requireAuth: true },
      },
      isAuthed: true,
    });

    assert.equal(links.some((link) => link.itemKey === 'pricing'), true);
  });
});
