import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const html = readFileSync(resolve('standalone/isurelink-home.html'), 'utf8')

describe('ISURELINK standalone home page', () => {
  test('uses the embedded sign-in route by default', () => {
    assert.match(html, /params\.get\('path'\)\s*\|\|\s*'\/embedded-sign-in'/)
    assert.doesNotMatch(html, /params\.get\('path'\)\s*\|\|\s*'\/sign-in'/)
  })

  test('keeps only the requested hero copy and removes extra chrome text', () => {
    assert.match(html, />ISURELINK</)
    assert.match(
      html,
      /一站式大模型 API 网关，提供专业稳定的大模型 API 服务/
    )
    assert.doesNotMatch(html, /Secure Sign-In/)
    assert.doesNotMatch(html, /未设置目标站点/)
    assert.doesNotMatch(html, /缺少目标站点地址/)
    assert.doesNotMatch(html, /Model Gateway Interface/)
    assert.doesNotMatch(html, /Unified Access/)
    assert.doesNotMatch(html, /Stable Routing/)
    assert.doesNotMatch(html, /Operational Ready/)
  })

  test('uses a plain light background and keeps subtitle on one line', () => {
    assert.doesNotMatch(html, /background-image:\s*\n?\s*linear-gradient\(var\(--grid\) 1px, transparent 1px\)/)
    assert.match(html, /\.hero-subtitle\s*\{[\s\S]*white-space:\s*nowrap;/)
  })
})
