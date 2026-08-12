import { test } from 'node:test'
import assert from 'node:assert/strict'
import { renderMarkdownLite } from './markdown-lite.ts'

test('renderMarkdownLite escapes HTML and renders paragraphs', () => {
  const html = renderMarkdownLite('Hello <script>alert(1)</script>\n\nWorld')
  assert.match(html, /&lt;script&gt;/)
  assert.doesNotMatch(html, /<script>/)
  assert.match(html, /<p>/)
})

test('renderMarkdownLite supports links and bold', () => {
  const html = renderMarkdownLite('See [docs](/docs) and **bold**')
  assert.match(html, /href="\/docs"/)
  assert.match(html, /<strong>bold<\/strong>/)
})
