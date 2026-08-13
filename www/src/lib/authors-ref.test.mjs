import assert from 'node:assert/strict'
import { describe, it } from 'node:test'

/**
 * Mirrors www/src/lib/authors.ts author-ref helpers so SSG stays resilient when
 * the content API embeds author objects on article payloads.
 */
function authorSlugFrom(ref) {
  if (typeof ref === 'string') return ref
  if (ref && typeof ref === 'object' && typeof ref.slug === 'string') return ref.slug
  return ''
}

function authorDisplayName(ref, registry = new Map()) {
  const slug = authorSlugFrom(ref)
  const registered = slug ? registry.get(slug)?.name : undefined
  if (typeof registered === 'string' && registered) return registered
  if (ref && typeof ref === 'object' && typeof ref.name === 'string' && ref.name) return ref.name
  return slug
}

describe('author ref coercion (SSG byline)', () => {
  it('extracts slugs from API author objects', () => {
    assert.equal(authorSlugFrom({ slug: 'chase-willden', name: 'Chase Willden' }), 'chase-willden')
    assert.equal(authorSlugFrom('chase-willden'), 'chase-willden')
    assert.equal(authorSlugFrom(null), '')
  })

  it('never returns a non-string display name for InitialsAvatar', () => {
    const name = authorDisplayName({ slug: 'chase-willden', name: 'Chase Willden' })
    assert.equal(typeof name, 'string')
    assert.equal(name, 'Chase Willden')
    assert.equal(name.split(/\s+/)[0], 'Chase')
  })

  it('prefers registry names when the slug is known', () => {
    const registry = new Map([['chase-willden', { name: 'Chase Willden' }]])
    assert.equal(authorDisplayName('chase-willden', registry), 'Chase Willden')
  })
})
