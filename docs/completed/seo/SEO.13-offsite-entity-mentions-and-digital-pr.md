# SEO.13 — Off-Site Entity, Mentions & Digital PR

> Implementation completed 2026-08-11. The repository now provides the press page, brand assets,
> policy, registers, operating templates, and schema/profile CI gate. Off-site profile claims,
> community participation, earned coverage, reviews, and 12-month outcome targets remain ongoing
> human marketing operations and must only be recorded when they actually occur.

> Implementation plan. Source: [docs/plan/seo/research.md §8](research.md#8-off-site-where-ai-citations-actually-come-from)
> and [audit.md](audit.md) F-9.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | SEO.13 |
| **Section** | SEO — Organic & AI-Search Ranking |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | MISSING (no claimed review-site profiles, no directory listings, no community presence, no PR motion, no Wikidata item) |
| **Estimated effort** | L (1–2mo to establish, then continuous) |
| **Owner (proposed)** | Marketing (content lead) + founder |
| **Depends on** | SEO.3, SEO.7, SEO.8, SEO.12 |
| **Unblocks** | SEO.3 FR-7 (`sameAs` needs claimed profiles) |

---

## 1. Problem Statement

**85% of brand mentions in AI answers originate on third-party pages, not the owned domain**, and
~48% of AI citations come from community platforms — Reddit and YouTube foremost — with Google's $60M
Reddit licensing deal putting that content directly in the retrieval path
([research §8](research.md#8-off-site-where-ai-citations-actually-come-from)). Brand mentions
correlate 0.664 with AI-Overview citation versus 0.218 for backlinks. We have no claimed G2 or
Capterra profile, no directory listings, no community presence, no PR motion, and no Wikidata item
(audit F-9) — which also blocks SEO.3's `sameAs` array, because we will not list a profile we have not
claimed. Everything else in this plan set improves what we say about ourselves; this is the plan that
gets other people to say it.

## 2. Goals

- Claim and complete every profile that constitutes our entity footprint, so `sameAs` can be
  populated and the Wikidata item has independent references.
- Establish **genuine, disclosed** presence in the communities where our buyers actually ask for
  recommendations.
- Run a repeatable digital-PR motion built on assets people want to cite (SEO.12 research, SEO.10
  tools, SEO.9 comparisons).
- Generate **+250 referring domains and +400 monthly third-party mentions** in 12 months.
- Do all of it in a way that survives scrutiny — Google extended its spam policies, including
  **inauthentic mentions**, to AI Overviews and AI Mode on 2026-05-15, and the communities themselves
  enforce harder than Google does.

## 3. Non-Goals

- Buying links, sponsored do-follow placements, PBNs, or link exchanges.
- Astroturfing: undisclosed accounts, incentivised reviews, fake community participation. This is a
  hard prohibition (FR-2), not a preference.
- Paid advertising or influencer campaigns (may exist separately; not part of this plan).
- Social-media management as a brand-awareness function beyond what supports mentions and citations.

## 4. Personas & User Stories

- **As a teacher asking Reddit "anyone used Lextures?"**, I want a real, identified response from the
  company plus genuine user replies, so that I can judge honestly.
- **As a buyer**, I want Lextures on the review sites I already use, with real reviews, so that it
  looks like a real company.
- **As an AI assistant**, I want corroborating third-party sources about Lextures, so that I can
  answer questions about it confidently rather than hedging.
- **As a journalist**, I want a contactable company with a press kit and a real spokesperson, so that
  I can quote them on deadline.
- **As our founder**, I want a sustainable amount of community participation, so that this does not
  become a second full-time job.

## 5. Functional Requirements

**Authenticity policy (governs everything below)**

- **FR-1.** Every community post, review response, comment or contribution made on behalf of Lextures
  MUST disclose affiliation in the post itself (not only in a profile), using a consistent form:
  *"I work at Lextures — …"*.
- **FR-2.** The following are prohibited without exception: undisclosed accounts, sockpuppets,
  incentivised or compensated reviews, review-gating (soliciting only from happy customers), fake
  testimonials, paid do-follow links, and any content designed to read as independent when it is not.
- **FR-3.** Every account used for Lextures participation MUST be registered in an owned-accounts
  register with a named owner. Personal accounts used for company purposes MUST disclose affiliation.
- **FR-4.** Review solicitation MUST be **unfiltered**: the request goes to a defined cohort
  (e.g. all customers past 60 days), never selectively to satisfied ones, with no incentive tied to
  sentiment.

**Entity footprint (unblocks SEO.3 `sameAs`)**

- **FR-5.** Claim, complete and verify, in this order:

  | Tier | Property | Why |
  |---|---|---|
  | 1 | G2, Capterra/GetApp/Software Advice | Buyer-facing; heavily cited by assistants for category questions |
  | 1 | LinkedIn company page | Top-2 AI citation driver per 2026 data |
  | 1 | Crunchbase | Entity resolution; a common independent reference for Wikidata |
  | 1 | GitHub org (exists: `StudyDrift/lextures`) | Already real; must be linked and described |
  | 2 | YouTube channel | Community-platform citations; feeds SEO.14 |
  | 2 | EdSurge / EdTech directories / Common Sense Education | Category-specific authority |
  | 2 | Product Hunt, AlternativeTo, Slant, SaaSHub | "Alternatives to X" retrieval surfaces |
  | 2 | 1EdTech member/product directory (if certified) | Standards credibility for institutional buyers |
  | 3 | Wikidata item | Entity home target — created only after ≥3 independent references exist |
  | 3 | Wikipedia | Only if genuinely notable; never self-created |

- **FR-6.** Each profile MUST carry consistent NAP-equivalent data — legal name, URL, founding date,
  description, categories, logo — matching the `Organization` schema exactly (SEO.3 FR-6).
  Inconsistency across profiles is what prevents entity resolution.
- **FR-7.** A profile MUST NOT be added to `sameAs` until it is claimed, verified and complete.
- **FR-8.** Profiles MUST be reviewed quarterly for drift (stale pricing, wrong categories, outdated
  screenshots).

**Community program**

- **FR-9.** Participate where our buyers actually are, with a defined weekly time budget:
  r/edtech, r/Teachers, r/homeschool, r/highereducation, r/instructionaldesign, r/CanvasLMS;
  the Higher Ed Learning Collective, ISTE and state edtech communities; LinkedIn groups; Hacker News
  (for the engineering/self-hosting angle).
- **FR-10.** Participation MUST be **answer-first**: respond to questions where we have real
  expertise, mention the product only where directly relevant, and link only when the link is the
  best available answer. A useful answer with no link outperforms a link with no answer — including
  for our purposes, since a mention without a hyperlink still counts.
- **FR-11.** Weekly cadence: **5 substantive answers** across platforms, owned by a named person, with
  a rotation so it is not one individual's burden.
- **FR-12.** Read subreddit and community rules before first participation; where self-promotion is
  prohibited, we participate without promoting or we do not participate.
- **FR-13.** Monitor brand mentions daily and respond to direct questions about Lextures within 24
  hours, including critical ones — publicly, factually, without defensiveness.

**Digital PR**

- **FR-14.** Each SEO.12 research report MUST have a coordinated outreach push: a target list of 40–60
  journalists, newsletter writers and analysts covering education technology, personalised pitches, an
  embargo option for tier-1 targets, and the press kit from SEO.12 FR-19.
- **FR-15.** Maintain a **press page** at `/press` with boilerplate, logo pack, founder bio and photo,
  fact sheet, recent coverage, and a monitored press contact.
- **FR-16.** Register the founder as an expert source on journalist-request platforms and respond to
  relevant queries with substantive, quotable answers — 2–3 responses per week.
- **FR-17.** Pursue **5 podcast appearances and 5 guest articles per year** on education-technology
  publications, each with a real editorial standard (no paid placements, no syndicated duplicates of
  our own posts).
- **FR-18.** Pursue speaking slots at 2–3 conferences per year (ISTE, EDUCAUSE, state-level events);
  session listings are durable, high-authority mentions.
- **FR-19.** Build **linkable assets deliberately**: the SEO.10 tools and SEO.12 datasets are the
  outreach hooks. Every quarter, one asset is chosen for an outreach push with a target of ≥15
  referring domains.

**Reviews**

- **FR-20.** Run an ongoing, unfiltered review-request motion (FR-4) targeting **≥25 reviews on G2 and
  ≥25 on Capterra within 12 months**, from verified customers.
- **FR-21.** Respond publicly to every review, positive or negative, within 5 business days —
  responses are indexed content and are read by both buyers and assistants.
- **FR-22.** Negative reviews MUST NOT be suppressed or disputed except on factual inaccuracy, using
  the platform's documented process.

## 6. Non-Functional Requirements

- **Performance** — n/a (mostly off-site). The `/press` page follows the SEO.4 static budget.
- **Security** — shared accounts use SSO or a password manager with per-person access and MFA; no
  shared plaintext credentials. Loss of a claimed profile is an availability and reputation risk.
- **Privacy & Compliance** — customer names in case studies, quotes or reviews require written
  consent recorded in the consent ledger ([S04](../standards/S04-unified-consent-preference-ledger.md)).
  Never reference a school's or student's data in public. FTC endorsement rules apply to any
  testimonial or incentivised content — hence FR-2 and FR-4.
- **Accessibility** — the press page and logo pack must include accessible formats; conference and
  webinar materials follow the same WCAG standard as the site.
- **Scalability** — the program is capped by human time; FR-11's 5-answers-per-week and FR-16's 2–3
  responses are deliberately sustainable numbers, not aspirational ones.
- **Reliability** — a mention register (see §8) makes the program measurable and prevents duplicated
  or contradictory outreach.
- **Observability** — track: referring domains, unlinked mentions, review counts and average rating,
  share of AI answers citing third-party sources about us, community-answer engagement, press
  placements. Rolled into the SEO.15 dashboard.
- **Maintainability** — one register file, one owner per channel, quarterly profile review (FR-8).
- **Internationalization** — English-language communities only at launch.
- **Backward compatibility** — n/a.

## 7. Acceptance Criteria

- **AC-1.** *Given* month 2, *When* audited, *Then* all Tier-1 profiles are claimed, verified and
  complete, with data matching the `Organization` schema exactly (AC verified by a field-by-field
  diff).
- **AC-2.** *Given* SEO.3's `sameAs` array, *When* checked, *Then* every entry is a claimed, verified
  profile, and no unclaimed listing appears.
- **AC-3.** *Given* any community post made on our behalf, *When* reviewed, *Then* it discloses
  affiliation in the post body and is recorded in the owned-accounts register.
- **AC-4.** *Given* the review-request motion, *When* audited, *Then* requests went to a defined
  cohort without sentiment filtering and without incentives, evidenced by the request log.
- **AC-5.** *Given* 12 months, *When* measured, *Then* ≥25 G2 reviews and ≥25 Capterra reviews exist
  from verified customers, and every review has a public response within 5 business days.
- **AC-6.** *Given* 12 months, *When* measured, *Then* referring domains have increased by ≥250 and
  monthly third-party brand mentions by ≥400 over the week-3 baseline.
- **AC-7.** *Given* a research report launch, *When* the outreach push completes, *Then* ≥40 targets
  were contacted with personalised pitches and ≥10 distinct publications cited the report.
- **AC-8.** *Given* the Wikidata item, *When* submitted, *Then* it cites ≥3 independent references and
  survives 90 days without deletion nomination.
- **AC-9.** *Given* a critical public mention, *When* it appears, *Then* it receives a factual,
  non-defensive response within 24 hours.
- **AC-10.** *Given* an audit of all off-site activity, *When* conducted, *Then* zero instances of
  undisclosed promotion, incentivised reviews, or purchased links are found.

## 8. Data Model

No production data model. Program registers, versioned in the repo (with no secrets):

```
docs/plan/seo/offsite/
  profiles.md          # property, URL, status, owner, claimedAt, lastReviewed, in-sameAs?
  accounts.md          # platform, handle, owner, disclosure form, purpose
  mentions.md          # date, source, URL, type (link|unlinked), sentiment, campaign
  outreach/<campaign>.md  # targets, pitch, status, outcome
  press-coverage.md    # date, outlet, URL, headline stat cited
```

Mention record shape:

```
| date | source | url | type | linked | sentiment | campaign | notes |
```

## 9. API Surface

None. Third-party tooling (mention monitoring, review platform APIs) is read-only and operated
outside the product.

## 10. UI / UX

- **New pages:** `/press` (FR-15) with boilerplate, fact sheet, logo pack, founder bio/photo, coverage
  list, press contact.
- **Modified:** `/about` (SEO.3 FR-18) links to `/press` and to every claimed profile; footer gains a
  Press link under Company.
- **Flows**
  1. Journalist reads a research report → `/press` → downloads assets → contacts us → publishes.
  2. Buyer sees a G2 review → clicks through → `/compare/lextures-vs-canvas` → `/request-information`.
  3. Community member asks about us on Reddit → disclosed answer with a help-article link → visit.
- **States** — coverage list empty at launch: show the press contact and boilerplate rather than an
  empty "As seen in" strip (a fake-looking logo wall is worse than none).
- **Responsive** — logo pack downloads and the fact sheet table must work on mobile.
- **Accessibility** — logo pack includes SVG plus PNG with described usage; founder photo has
  meaningful alt text; the fact sheet is a real table.
- **Copy & i18n** — `www.press.*`; boilerplate reviewed by legal and kept identical across every
  profile (FR-6).

## 11. AI / ML Considerations

- **This plan targets the 85%.** Owned-site work (SEO.1–SEO.11) makes us retrievable; this plan makes
  us *corroborated*. Assistants hedge on entities with no third-party grounding, and the hedge is what
  keeps us out of recommendation lists.
- **Consistency is the mechanism.** Identical name, description, founding date and categories across
  every profile (FR-6) is what lets a model resolve the mentions to one entity; drift produces two
  weak entities instead of one strong one.
- **Unlinked mentions count.** The mention register (§8) deliberately tracks `linked: false` rows,
  because the correlation data says mentions — not links — predict AI citation.
- **AI may draft outreach; a human sends it.** Personalisation is the entire value of a pitch, and
  mass-generated pitches damage relationships permanently. No automated posting to community platforms
  under any circumstances.
- **Monitoring must include AI answers themselves**: SEO.15's prompt harness records which third-party
  sources assistants cite about us, which tells us where to earn the next mention.

## 12. Integration Points

- **External:** G2, Capterra/GetApp/Software Advice, LinkedIn, Crunchbase, YouTube, Reddit, Hacker
  News, Product Hunt, AlternativeTo, Wikidata, journalist-request platforms, mention-monitoring tool.
- **Internal modules touched:** `www/src/pages/press-page.tsx` (new), `www/src/lib/site-links.ts`,
  `www/src/lib/schema/organization.ts` (`sameAs`), `www/src/components/site-footer.tsx`.
- **Events:** press-kit downloads → GA4.

## 13. Dependencies & Sequencing

- **Must ship after:** [SEO.3](SEO.3-structured-data-and-entity-graph.md) (the schema that profiles
  must match), [SEO.7](SEO.7-help-center-expansion.md) (community answers need canonical URLs to
  link), [SEO.8](SEO.8-editorial-engine-and-content-calendar.md) (content to promote),
  [SEO.12](SEO.12-original-research-and-data-program.md) (the asset outreach is built on).
- **Must ship before:** [SEO.3](SEO.3-structured-data-and-entity-graph.md) FR-7's completion — this is
  a two-way dependency: schema defines the canonical data, this plan claims the profiles, then
  `sameAs` is populated. Sequence: SEO.3 core → SEO.13 Tier 1 → SEO.3 `sameAs` + Wikidata.
- **Shared infra:** mention-monitoring subscription; press contact address; consent process for
  customer quotes.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Community participation reads as promotion and gets us banned | **H** | H | FR-1 disclosure, FR-10 answer-first, FR-12 rules-first; a ban is effectively permanent, so err toward participating less |
| Someone shortcuts to undisclosed promotion under pressure | M | **H** | FR-2 absolute prohibition + FR-3 account register + AC-10 audit; make it an explicit written policy people can point to when asked |
| Review solicitation drifts into gating | M | H | FR-4 defined cohort + request log audited in AC-4; FTC exposure makes this a legal issue, not a tactics one |
| Program depends entirely on the founder's time | **H** | M | FR-11 capped cadence with rotation; prioritise durable assets (research, tools) over per-post effort |
| Wikidata item deleted for non-notability | M | M | FR-5 Tier 3 gated on ≥3 independent references; never self-create a Wikipedia article |
| Negative reviews or hostile threads | M | M | FR-21/FR-22 respond factually, never suppress; a well-handled negative review is credible evidence for buyers |
| Outreach produces nothing and demoralises the team | M | M | FR-19 quarterly asset-led pushes with a ≥15-domain target; measure per campaign and stop what does not work |

## 15. Rollout Plan

- **Feature flag:** none.
- **Sequencing**
  1. **Weeks 1–2:** write and sign off the authenticity policy (FR-1–FR-4); create the registers; audit
     existing unclaimed listings.
  2. **Weeks 3–4:** claim and complete all Tier-1 profiles; align copy to the `Organization` schema;
     notify SEO.3 to populate `sameAs`.
  3. **Weeks 5–6:** ship `/press`; register the founder on journalist-request platforms; start the
     weekly community cadence.
  4. **Weeks 7–8:** Tier-2 profiles; begin the review motion (FR-20).
  5. **Month 3:** first asset-led outreach push (SEO.10 tool or SEO.9 comparison).
  6. **Month 4:** research report #1 outreach (SEO.12) — the largest push of the year.
  7. **Month 6:** Wikidata item once ≥3 independent references exist; quarterly profile review begins.
- **Dogfood:** the founder does the first two weeks of community participation personally, to
  calibrate tone before delegating.
- **GA criteria:** AC-1…AC-10 across 12 months; program reviewed quarterly with a stop/continue
  decision per channel.
- **Rollback:** channels can be dropped individually; claimed profiles are kept regardless (an
  unmaintained claimed profile still beats an unclaimed one, provided FR-8 review continues).

## 16. Test Plan

Mostly process; the testable parts:

- **Unit** — `sameAs` array validated against `profiles.md` status (a profile not marked `claimed`
  cannot appear in schema); press-page schema validation.
- **Integration** — build fails if `sameAs` contains a URL absent from the claimed-profiles register
  (AC-2).
- **End-to-end** — `/press` renders server-side; logo pack and fact sheet download.
- **Security** — verify no credentials in the registers; verify MFA on every claimed profile;
  quarterly access review.
- **Accessibility** — axe on `/press`; alt text on founder photo and logo previews.
- **Performance / load** — `/press` within the static budget.
- **Manual exploratory** — quarterly authenticity audit (AC-10): sample 20 off-site posts and reviews
  and verify disclosure and non-incentivisation; quarterly profile-drift review (FR-8).

## 17. Documentation & Training

- `docs/plan/seo/offsite/authenticity-policy.md` — the prohibitions, the disclosure form, what to do
  when someone asks you to bend it, and who to escalate to.
- `www/docs/community-participation-guide.md` — where we participate, how to answer, when to link,
  platform-specific rules.
- Press kit and media-response runbook (shared with SEO.12).
- Onboarding: anyone who will post on behalf of Lextures reads and acknowledges the authenticity
  policy first.

## 18. Open Questions

1. Who owns community participation day-to-day after the founder's first two weeks?
2. What is the budget for a mention-monitoring tool and review-platform presence (G2 listings have
   paid tiers — we should confirm what the free tier permits)?
3. Do we build a customer-reference program to source reviews, quotes and case studies with consent?
4. Which conferences are worth the travel budget, and who speaks?
5. Is the founder comfortable being the public face (needed for FR-16/FR-17/FR-18 and for SEO.3's
   founder `Person` entity)?

## 19. References

- Audit findings: [F-9](audit.md#f-9-no-brand-entity-anywhere), [F-11](audit.md#f-11-zero-named-authors)
- Research: [§8 Off-site](research.md#8-off-site-where-ai-citations-actually-come-from),
  [§4 Entity SEO](research.md#4-entity-seo-is-the-highest-roieffort-ratio-available),
  [§7 spam-policy extension](research.md#7-content-strategy-concentration-beats-volume-utility-beats-pages)
- External: [Google — Spam policies (link spam, inauthentic mentions)](https://developers.google.com/search/docs/essentials/spam-policies),
  [FTC — Endorsement guides](https://www.ftc.gov/business-guidance/resources/ftcs-endorsement-guides-what-people-are-asking),
  [Wikidata notability](https://www.wikidata.org/wiki/Wikidata:Notability),
  [Reddit — self-promotion guidelines](https://support.reddithelp.com/hc/en-us/articles/205926439)
- Related plans: [SEO.3](SEO.3-structured-data-and-entity-graph.md),
  [SEO.9](SEO.9-comparison-alternatives-and-integration-pages.md),
  [SEO.10](SEO.10-programmatic-utility-pages.md),
  [SEO.12](SEO.12-original-research-and-data-program.md),
  [SEO.15](SEO.15-measurement-search-console-and-ai-share-of-voice.md),
  [S04 — consent ledger](../standards/S04-unified-consent-preference-ledger.md)
