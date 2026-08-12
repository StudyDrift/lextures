import assert from 'node:assert/strict'
import test from 'node:test'
import { buildFeeds } from './feeds.mjs'

test('feeds escape hostile content, exclude noindex, and use canonical URLs', () => {
  const { rss, json, itemCount } = buildFeeds([{ path: '/blog/a', title: 'A & B', description: '<script>x</script> Safe & sound', content: 'Full', author: 'Ada', date: '2026-01-02', canonicalOverride: 'https://example.com/a' }, { path: '/blog/hidden', title: 'Hidden', description: 'No', author: 'Ada', date: '2026-01-03', noindex: true }])
  assert.equal(itemCount, 1)
  assert.match(rss, /A &amp; B/)
  assert.doesNotMatch(rss, /<script>|Hidden/)
  assert.match(rss, /https:\/\/example.com\/a/)
  assert.equal(JSON.parse(json).items[0].url, 'https://example.com/a')
})
