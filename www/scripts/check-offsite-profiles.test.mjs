import assert from 'node:assert/strict'
import { describe, it } from 'node:test'
import { claimedSameAsUrls, validateSameAs } from './check-offsite-profiles.mjs'

const links = `export const SITE_LINKS = { github: 'https://github.com/StudyDrift/lextures' }`
const entity = `export const VERIFIED_SAME_AS = [SITE_LINKS.github]`

describe('off-site claimed-profile gate', () => {
  it('only reads rows that are both claimed and approved for sameAs', () => {
    const markdown = [
      '| tier | property | url | status | owner | claimedAt | lastReviewed | nextReview | MFA | inSameAs | notes |',
      '| 1 | GitHub | https://github.com/StudyDrift/lextures | claimed | Owner | date | date | date | yes | yes | ok |',
      '| 1 | G2 | https://g2.com/products/x | unclaimed | Owner | — | date | — | yes | no | pending |',
    ].join('\n')
    assert.deepEqual(claimedSameAsUrls(markdown), ['https://github.com/StudyDrift/lextures'])
  })

  it('rejects a sameAs URL absent from the claimed register', () => {
    const result = validateSameAs({ profilesMarkdown: '# empty', entitySource: entity, siteLinksSource: links })
    assert.deepEqual(result.missingFromRegister, ['https://github.com/StudyDrift/lextures'])
  })

  it('accepts exact register/schema agreement', () => {
    const markdown = '| 1 | GitHub | https://github.com/StudyDrift/lextures | claimed | Owner | date | date | date | yes | yes | ok |'
    const result = validateSameAs({ profilesMarkdown: markdown, entitySource: entity, siteLinksSource: links })
    assert.deepEqual(result.missingFromRegister, [])
    assert.deepEqual(result.missingFromSchema, [])
  })
})
