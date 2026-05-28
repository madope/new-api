import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const defaultEmbeddedSignIn = readFileSync(
  resolve('web/default/src/features/auth/sign-in/embedded-sign-in.tsx'),
  'utf8'
)
const classicApp = readFileSync(resolve('web/classic/src/App.jsx'), 'utf8')

describe('Embedded auth routes', () => {
  test('default embedded sign-in links to embedded sign-up', () => {
    assert.match(defaultEmbeddedSignIn, /to='\/embedded-sign-up'/)
  })

  test('classic app exposes embedded sign-up route', () => {
    assert.match(classicApp, /path='\/embedded-sign-up'/)
  })
})
