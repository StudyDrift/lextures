# Off-site profile register

Canonical entity data is copied from `www/src/lib/schema/entity.ts`. A URL may enter Organization
`sameAs` only when `status` is `claimed` and `inSameAs` is `yes`. Quarterly review dates are explicit;
credentials are never stored here.

## Canonical entity fields

| field | value |
|---|---|
| name | Lextures |
| legalName | Lextures LLC |
| url | https://lextures.com/ |
| foundingDate | 2024-01-01 |
| description | Adaptive learning management system for K–12, higher education, and homeschool: IRT-routed quizzes, gradebook, roster sync, self-hosting under AGPL-3.0, and a public course marketplace. |
| categories | Adaptive learning; Learning management system; Education software |
| logo | https://lextures.com/logo.svg |

## Profiles

| tier | property | url | status | owner | claimedAt | lastReviewed | nextReview | MFA | inSameAs | notes |
|---|---|---|---|---|---|---|---|---|---|---|
| 1 | GitHub repository | https://github.com/StudyDrift/lextures | claimed | Chase Willden | pre-register | 2026-08-11 | 2026-11-11 | required | yes | Owned public repository; description and website must match canonical fields. |
| 1 | G2 | — | unclaimed | Chase Willden | — | 2026-08-11 | — | required | no | Claim and verify before publishing a URL. |
| 1 | Capterra / GetApp / Software Advice | — | unclaimed | Chase Willden | — | 2026-08-11 | — | required | no | Gartner Digital Markets family; record canonical listing after verification. |
| 1 | LinkedIn company page | — | unclaimed | Chase Willden | — | 2026-08-11 | — | required | no | Do not infer a URL from an unverified search result. |
| 1 | Crunchbase | — | unclaimed | Chase Willden | — | 2026-08-11 | — | required | no | Independent references required before Wikidata. |
| 2 | YouTube | — | unclaimed | Chase Willden | — | 2026-08-11 | — | required | no | Coordinate with SEO.14. |
| 2 | Education directories | — | unclaimed | Chase Willden | — | 2026-08-11 | — | required | no | EdSurge, EdTech directories, Common Sense Education. |
| 2 | Discovery directories | — | unclaimed | Chase Willden | — | 2026-08-11 | — | required | no | Product Hunt, AlternativeTo, Slant, SaaSHub. |
| 2 | 1EdTech | — | not-eligible | Chase Willden | — | 2026-08-11 | — | required | no | Add only if membership or certification is verified. |
| 3 | Wikidata | — | gated | Chase Willden | — | 2026-08-11 | — | required | no | Create only after three independent reliable references exist. |
| 3 | Wikipedia | — | gated | Chase Willden | — | 2026-08-11 | — | required | no | Never self-create; independent notability required. |

