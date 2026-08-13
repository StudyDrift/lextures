/**
 * Unit tests for structured-data builders (SEO.3).
 * Mirrors pure logic for Node's test runner without Vite.
 */
import assert from 'node:assert/strict'
import { describe, it } from 'node:test'
import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import path from 'node:path'

const __dirname = path.dirname(fileURLToPath(import.meta.url))

// Import compiled-free pure helpers by re-implementing critical paths via generate-site
// and reading source constants from TS is awkward; test the JS validators in generate-site
// and assert source contracts for authors/pricing.

const gen = await import('../../../scripts/generate-site.mjs')
const {
  serializeJsonLd,
  validateJsonLdGraph,
  collectSchemaTypes,
} = gen

const SITE = 'https://lextures.com'

function orgNode() {
  return {
    '@type': 'Organization',
    '@id': `${SITE}/#organization`,
    name: 'Lextures',
    url: `${SITE}/`,
  }
}

function websiteNode() {
  return {
    '@type': 'WebSite',
    '@id': `${SITE}/#website`,
    name: 'Lextures',
    publisher: { '@id': `${SITE}/#organization` },
  }
}

describe('schema graph serialisation (FR-1, FR-3)', () => {
  it('emits a single @graph envelope', () => {
    const out = serializeJsonLd([orgNode(), websiteNode()])
    const parsed = JSON.parse(out)
    assert.equal(parsed['@context'], 'https://schema.org')
    assert.ok(Array.isArray(parsed['@graph']))
    assert.equal(parsed['@graph'].length, 2)
  })

  it('escapes hostile course title (AC-7)', () => {
    const hostile = '</script><img src=x onerror=alert(1)>'
    const out = serializeJsonLd([
      {
        '@type': 'Course',
        '@id': `${SITE}/courses/evil#course`,
        name: hostile,
      },
    ])
    assert.doesNotMatch(out, /<\/script>/i)
    assert.match(out, /\\u003c/)
  })
})

describe('schema graph validation (FR-5)', () => {
  it('accepts a well-formed graph with resolved refs', () => {
    const errors = validateJsonLdGraph([orgNode(), websiteNode()], '/')
    assert.deepEqual(errors, [])
  })

  it('rejects non-absolute @id', () => {
    const errors = validateJsonLdGraph(
      [{ '@type': 'Thing', '@id': '#local' }],
      '/x',
    )
    assert.ok(errors.some(e => e.includes('not absolute')))
  })

  it('rejects dangling @id references', () => {
    const errors = validateJsonLdGraph(
      [
        {
          '@type': 'Article',
          '@id': `${SITE}/blog/x#article`,
          author: { '@id': `${SITE}/authors/missing#person` },
        },
      ],
      '/blog/x',
    )
    assert.ok(errors.some(e => e.includes('dangling')))
  })

  it('rejects missing @id', () => {
    const errors = validateJsonLdGraph([{ '@type': 'Thing' }], '/y')
    assert.ok(errors.some(e => e.includes('missing @id')))
  })
})

describe('collectSchemaTypes', () => {
  it('walks @graph', () => {
    const types = collectSchemaTypes({
      '@context': 'https://schema.org',
      '@graph': [orgNode(), websiteNode()],
    })
    assert.ok(types.includes('Organization'))
    assert.ok(types.includes('WebSite'))
  })
})

describe('pricing source of truth (AC-6)', () => {
  it('institution-pricing tiers are used by software-application source', () => {
    const pricingSrc = readFileSync(
      path.join(__dirname, '../institution-pricing.ts'),
      'utf8',
    )
    const softSrc = readFileSync(
      path.join(__dirname, 'software-application.ts'),
      'utf8',
    )
    const offerSrc = readFileSync(path.join(__dirname, 'offer.ts'), 'utf8')
    assert.match(softSrc, /PRICING_TIERS/)
    assert.match(offerSrc, /PRICING_TIERS/)
    assert.match(pricingSrc, /pricePerStudent: 6/)
    assert.match(offerSrc, /HOMESCHOOL_MONTHLY_USD/)
  })
})

describe('author registry (FR-20)', () => {
  it('authors.ts exports chase-willden with active|retired status type', () => {
    const src = readFileSync(path.join(__dirname, '../authors.ts'), 'utf8')
    assert.match(src, /slug: 'chase-willden'/)
    assert.match(src, /status: 'active'/)
    assert.match(src, /export type AuthorStatus = 'active' \| 'retired'/)
  })

  it('database articles resolve author registry slugs', () => {
    const src = readFileSync(path.join(__dirname, '../content-source.ts'), 'utf8')
    assert.match(src, /value\.author\?\.slug/)
    assert.doesNotMatch(src, /Lextures Team/)
  })
})

describe('sameAs allowlist (FR-7)', () => {
  it('entity sameAs uses SITE_LINKS.github only (no unclaimed review sites)', () => {
    const src = readFileSync(path.join(__dirname, 'entity.ts'), 'utf8')
    assert.match(src, /VERIFIED_SAME_AS/)
    assert.match(src, /SITE_LINKS\.github/)
    // Active (uncommented) G2/Capterra listings must not appear
    const activeLines = src
      .split('\n')
      .filter(l => !l.trim().startsWith('//') && !l.includes('Add once claimed'))
      .join('\n')
    assert.doesNotMatch(activeLines, /g2\.com/)
    assert.doesNotMatch(activeLines, /capterra\.com/)
  })
})
