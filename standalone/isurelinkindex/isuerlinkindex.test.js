import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import { existsSync, readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const html = readFileSync(
  resolve('standalone/isurelinkindex/isuerlinkindex.html'),
  'utf8'
)
const jsPath = resolve('standalone/isurelinkindex/isuerlinkindex.local.js')
const js = existsSync(jsPath) ? readFileSync(jsPath, 'utf8') : ''
const cssPath = resolve('standalone/isurelinkindex/isuerlinkindex.local.css')
const css = existsSync(cssPath) ? readFileSync(cssPath, 'utf8') : ''

describe('ISURELINK local standalone page', () => {
  test('uses only local assets', () => {
    assert.doesNotMatch(html, /https:\/\/cdn\.tailwindcss\.com/)
    assert.doesNotMatch(html, /https:\/\/fonts\.googleapis\.com/)
    assert.match(html, /<link[^>]+href="\.\/isuerlinkindex\.local\.css"/)
    assert.match(html, /<script[^>]+src="\.\/isuerlinkindex\.local\.js"/)
  })

  test('supports multilingual content and theme sync', () => {
    const source = html + js
    assert.match(source, /const translations = \{/)
    assert.match(source, /zh:/)
    assert.match(source, /en:/)
    assert.match(source, /fr:/)
    assert.match(source, /ja:/)
    assert.match(source, /ru:/)
    assert.match(source, /vi:/)
    assert.match(source, /themeMode/)
    assert.match(source, /window\.addEventListener\('message'/)
  })

  test('login button is wired to a real target path', () => {
    const source = html + js
    assert.match(html, /id="cta-login-button"/)
    assert.match(source, /loginPath/)
    assert.doesNotMatch(html, /href="#"/)
  })

  test('right visual uses an animated gateway routing composition', () => {
    assert.match(html, /class="visual-stage"/)
    assert.match(html, /class="exchange-halo"/)
    assert.match(html, /class="exchange-network-svg"/)
    assert.match(html, /id="exchange-path-1"/)
    assert.match(html, /class="exchange-network-particles"/)
    assert.match(html, /animateMotion dur="11s"/)
    assert.match(html, /class="exchange-lane-group exchange-lane-group-in"/)
    assert.match(html, /class="exchange-lane-group exchange-lane-group-out"/)
    assert.match(html, /class="exchange-lane lane-in lane-in-1"/)
    assert.match(html, /class="exchange-lane lane-out lane-out-3"/)
    assert.match(html, /class="exchange-board glass-panel"/)
    assert.match(html, /class="exchange-chip chip-6"/)
    assert.doesNotMatch(html, /class="exchange-board-symbol"/)
    assert.match(css, /@keyframes lane-flow/)
    assert.match(css, /\.exchange-network-line/)
  })
})
