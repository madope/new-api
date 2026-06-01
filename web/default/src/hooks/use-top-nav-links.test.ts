import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import { buildTopNavLinks } from './top-nav-links.lib.ts'

describe('buildTopNavLinks', () => {
  test('hides model square when pricing requires auth and user is not authenticated', () => {
    const links = buildTopNavLinks({
      t: (key) => key,
      status: {
        HeaderNavModules: JSON.stringify({
          home: true,
          console: true,
          pricing: { enabled: true, requireAuth: true },
          rankings: { enabled: true, requireAuth: false },
          docs: true,
          about: true,
        }),
      },
      isAuthed: false,
    })

    assert.equal(links.some((link) => link.href === '/pricing'), false)
  })

  test('keeps model square visible after login when pricing requires auth', () => {
    const links = buildTopNavLinks({
      t: (key) => key,
      status: {
        HeaderNavModules: JSON.stringify({
          pricing: { enabled: true, requireAuth: true },
        }),
      },
      isAuthed: true,
    })

    assert.equal(links.some((link) => link.href === '/pricing'), true)
  })
})
