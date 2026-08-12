import assert from 'node:assert/strict'
import { describe, it } from 'node:test'
import { validateHreflang } from './hreflang.mjs'

const alternates = `<link rel="alternate" hreflang="en" href="https://lextures.com/pricing" /><link rel="alternate" hreflang="fr-CA" href="https://lextures.com/fr-ca/pricing" /><link rel="alternate" hreflang="x-default" href="https://lextures.com/pricing" />`
const sitemapLinks = alternates.replaceAll('<link ', '<xhtml:link ')
function fixture(localizedHtml = alternates) {
  return {
    manifest: { urls: [
      { path: '/pricing', locale: 'en', robots: 'index,follow', sitemap: true },
      { path: '/fr-ca/pricing', locale: 'fr-CA', translationOf: '/pricing', robots: 'index,follow', sitemap: true },
    ] },
    htmlByPath: new Map([['/pricing', alternates], ['/fr-ca/pricing', localizedHtml]]),
    sitemapXmls: [`<urlset xmlns:xhtml="http://www.w3.org/1999/xhtml"><url><loc>https://lextures.com/pricing</loc>${sitemapLinks}</url><url><loc>https://lextures.com/fr-ca/pricing</loc>${sitemapLinks}</url></urlset>`],
  }
}

describe('hreflang CI check', () => {
  it('accepts reciprocal HTML and sitemap clusters', () => assert.deepEqual(validateHreflang(fixture()), []))
  it('names a non-reciprocal pair', () => {
    const broken = alternates.replace('<link rel="alternate" hreflang="en" href="https://lextures.com/pricing" />', '')
    assert.ok(validateHreflang(fixture(broken)).some(item => item.message.includes('Non-reciprocal') && item.message.includes('/pricing')))
  })
  it('fails when HTML and sitemap differ', () => {
    assert.ok(validateHreflang(fixture(alternates.replace('fr-CA', 'fr-FR'))).some(item => item.message.includes('do not match')))
  })
})
