# MKTC.7 — Abuse Prevention, Analytics, Docs & Rollout

> Implementation plan. Source: [docs/plan/marketplace/README.md](../../plan/marketplace/README.md). Part of the MKTC Course Coupon Codes epic.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | MKTC.7 |
| **Section** | Marketplace |
| **Severity** | MAJOR |
| **Markets** | HS (primary) · HE · K12 |
| **Status (today)** | COMPLETE |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Commerce / Growth squad (backend + web) with Security review |
| **Depends on** | MKTC.1, MKTC.2, MKTC.3, MKTC.4, MKTC.5, MKTC.6 |
| **Unblocks** | — (closes the epic) |

---

## 1. Problem Statement

MKTC.1–MKTC.6 build a working discount system. Shipping it to everyone without hardening would put
a money-moving endpoint in front of the internet: a short coupon code is guessable, an unmonitored
discount is an unbounded liability, and a creator who cannot see how a code performed will not
create another. This story closes the epic — brute-force defences on the apply endpoint, the
creator-facing performance view, the operational dashboards and alerts, the help documentation, and
the staged flip of `ffCourseCoupons` from default OFF to default ON.

## 2. Goals

- Make code guessing economically pointless: entropy floor on generated codes, layered rate limits,
  attempt logging, and an automatic cool-down on sustained failure.
- Give creators a small, honest performance view per coupon (claims, revenue, discount given).
- Give operators dashboards, alerts and a runbook that can answer "why did this discount happen?"
  in under five minutes.
- Publish learner and creator documentation covering every failure reason.
- Flip `ffCourseCoupons` to default ON with a documented GA bar and a one-switch rollback.

## 3. Non-Goals

- New discount capabilities (stacking, org-wide coupons, gift cards, per-recipient unique codes) —
  the epic's non-goals stand.
- A full marketing-analytics product; this is a per-coupon summary, not attribution modelling.
- Fraud scoring or ML detection; the controls here are deterministic rate limits and caps.
- Changing the iOS behaviour decided in MKTC.6.

## 4. Personas & User Stories

- **As a course creator**, I want to see that LAUNCH25 drove 34 sales and $180 of discount, so that
  I know whether to run it again.
- **As a platform operator**, I want an alert when someone is spraying codes at a course, so that I
  can respond before a working code is found.
- **As a platform operator**, I want a dashboard that reconciles discounts against Stripe, so that
  month-end has no surprises.
- **As a support agent**, I want a runbook entry for "my code didn't work", so that I can answer
  without escalating.
- **As a learner**, I want a help page that explains why a code failed, so that I do not open a
  ticket at all.
- **As a security reviewer**, I want the enumeration threat modelled and mitigated in writing.

## 5. Functional Requirements

### Abuse prevention

- **FR-1.** The coupon preview endpoint MUST enforce three layered limits: per (user, course)
  **15/min and 60/hour**, per user across all courses **100/hour**, and per IP **200/hour**.
  Exceeding any returns `429 RATE_LIMITED` with a `Retry-After` header.
- **FR-2.** After **10 consecutive failed** apply attempts by the same user on the same course
  within an hour, the system MUST impose a **15-minute cool-down** on that pair, returning `429`
  with a distinct client-visible reason so the UI can explain it. A successful apply resets the
  counter.
- **FR-3.** The system MUST log every failed apply attempt to a bounded audit trail
  (`billing.coupon_attempts`, retained 30 days) recording `user_id`, `course_id`, hashed code,
  `reason`, IP prefix (/24 IPv4, /48 IPv6), and timestamp. The **raw** code MUST NOT be stored for
  `not_found` results — a salted hash is stored instead, so the log cannot be mined for near-miss
  guesses.
- **FR-4.** Client-side code generation (MKTC.4 FR-8) MUST use ≥ 8 characters from a 32-symbol
  unambiguous alphabet (≥ 40 bits). The **server** MUST reject creation of codes shorter than 4
  characters (already enforced) and MUST warn on the create response
  (`warnings: ["low_entropy"]`) for codes under 6 characters or drawn from a dictionary word list,
  which MKTC.4's dialog surfaces as a non-blocking hint.
- **FR-5.** The system MUST expose a platform-level **discount ceiling** setting
  (`coupon_max_percent_off`, default 100) so an operator can cap creator discounts if abuse
  emerges; creation above the ceiling returns `422`.
- **FR-6.** The system MUST alert when a single coupon's redemptions exceed a configurable
  velocity (default 50 redemptions in 10 minutes) — the signature of a leaked "unlimited" code.

### Analytics

- **FR-7.** The creator coupon table (MKTC.4) MUST gain a **Performance** column set per coupon:
  claims, gross revenue at list price, total discount given, and net charged — computed from
  `coupon_redemptions` with `status='redeemed'` and read through a new
  `GET /api/v1/courses/{course_code}/coupons/summary` endpoint returning all coupons' figures in one
  request.
- **FR-8.** Refunded purchases MUST be excluded from net revenue and shown separately as
  `refundedCount`, so the numbers reconcile with the earnings ledger.
- **FR-9.** The redemptions drawer MUST gain a **CSV export** of that coupon's redemptions
  (learner name, email, status, amounts, dates), rate limited to 5 exports/hour/user, with the
  export recorded in the audit log. (Resolves MKTC.4 §18 Q4 in favour of shipping it, since
  creators need it for co-op reconciliation.)
- **FR-10.** Platform metrics MUST include: `coupon_apply_total{result}`,
  `coupon_apply_cooldown_total`, `coupon_created_total{discount_type}`,
  `coupon_redeemed_total`, `coupon_discount_cents_total`, `coupon_free_grant_total`,
  `coupon_clamped_to_free_total`, `coupon_released_total{reason}`,
  `coupon_reservation_expired_total`, `coupon_web_redirect_total{platform}`.
- **FR-11.** A Grafana dashboard MUST chart: apply attempts by result, discount given per day,
  redemption velocity per coupon (top 10), reservation expiry rate, and free-grant rate.
- **FR-12.** Alerts MUST fire for: sustained `not_found` rate above 30 % over 15 minutes
  (enumeration), redemption velocity breach (FR-6), `coupon_redeemed_total` diverging from
  coupon-tagged `marketplace_purchase_completed` over 1 hour (ledger drift), and reservation expiry
  rate above 50 % over 1 hour (checkout breakage).

### Documentation & rollout

- **FR-13.** Learner help page "Using a coupon code" MUST be published, covering entry, share links,
  the nine failure reasons, mobile behaviour (including the iOS partial-discount case), and refunds.
- **FR-14.** Creator help page "Create and share coupon codes" MUST be published, covering the four
  fields, share links, pause vs archive, what a coupon costs in revenue share, and the performance
  view.
- **FR-15.** `docs/runbooks/coupons.md` MUST be completed with: the reconciliation queries, how to
  release a stuck reservation, how to kill a leaked code, how to investigate "the discount didn't
  apply", and how to disable the feature.
- **FR-16.** `ffCourseCoupons` MUST flip from default **OFF** to default **ON** in
  `repos/platformconfig` only after the GA bar in §15 is met, in a separate, revertable commit.
- **FR-17.** A threat model note for coupon enumeration and discount abuse MUST be added to the
  security documentation set and reviewed by the security owner before the flag flips.

## 6. Non-Functional Requirements

- **Performance** — Rate-limit checks MUST add < 5 ms to the apply path (in-memory buckets with the
  same shape as `checkBillingCheckoutRateLimit`, backed by Redis when configured so limits hold
  across replicas). The summary endpoint MUST be one aggregate query, p95 < 200 ms for 100 coupons.
  CSV export streams rather than buffering.
- **Security** — Attempt logs store hashed codes for unknown codes (FR-3). Rate limits are enforced
  server-side only. The discount ceiling is platform-admin-only. Export is authorization-checked and
  audited. No control here may be bypassed by an unauthenticated caller because apply already
  requires a session.
- **Privacy & Compliance** — IP prefixes (not full addresses) are stored, for 30 days, documented in
  the RoPA/data-map (S-series). CSV export is a bulk egress of learner name and email to course
  staff who already have roster access; it is audit-logged and rate limited. The attempt log must be
  covered by the retention/deletion engine and by DSAR export.
- **Accessibility** — The performance column and CSV control follow MKTC.4's rules: real table
  semantics, text alternatives for every number, and a discernible name on the export button that
  includes the coupon code. Alert/monitor UIs are operator-facing (Grafana) and out of scope for
  WCAG conformance claims.
- **Scalability** — Limits are per-key buckets; Redis-backed when `REDIS_URL` is configured (reuse
  `internal/redisclient`), in-memory otherwise, matching the existing billing limiter's behaviour.
- **Reliability** — A limiter outage MUST fail **open** for legitimate purchases but MUST still
  enforce the coupon's own caps (which live in Postgres), so a Redis blip cannot cause oversell —
  only extra guessing capacity.
- **Observability** — This story *is* the observability story; every metric in FR-10 must have a
  dashboard panel and every alert an owning runbook section.
- **Maintainability** — Limits live in one place (`server/internal/service/billing/coupon_limits.go`)
  with named constants, not scattered magic numbers.
- **Internationalization** — New help pages and the cool-down message are translated; the CSV header
  row is localized while the data stays machine-parseable (ISO dates, minor units).
- **Backward compatibility** — Flipping the flag default changes behaviour for tenants that never
  touched the setting; the release notes and the admin panel copy must say so. Tenants who
  explicitly set it OFF keep OFF (the merge honours an explicit DB value).

## 7. Acceptance Criteria

- **AC-1.** *Given* I submit 16 codes in a minute for one course, *When* the 16th is submitted,
  *Then* I get `429` with `Retry-After` and the UI shows the cool-down copy.
- **AC-2.** *Given* 10 consecutive failed applies on a course, *When* I try an 11th — even a valid
  code — *Then* I get `429` with the cool-down reason for 15 minutes; *and given* a success before
  the 10th, *then* the counter resets.
- **AC-3.** *Given* a failed apply for an unknown code, *When* the attempt is logged, *Then* the
  stored value is a salted hash, the raw code appears nowhere in the row, and the IP is stored as a
  prefix.
- **AC-4.** *Given* the discount ceiling is set to 50 %, *When* a creator tries to create a 75 %
  coupon, *Then* they get `422` naming the ceiling.
- **AC-5.** *Given* a coupon with 34 redemptions and 2 refunds, *When* I open the coupon table,
  *Then* it shows 34 claims, the total discount given, net charged excluding refunds, and 2
  refunded.
- **AC-6.** *Given* a coupon's summary figures, *When* compared with `billing.earnings_ledger` for
  the same course and period, *Then* net charged reconciles to within rounding, asserted by an
  integration test.
- **AC-7.** *Given* I export a coupon's redemptions, *When* the CSV downloads, *Then* it contains
  one row per redemption with ISO dates and minor-unit amounts, and the export is recorded in the
  audit log.
- **AC-8.** *Given* 6 exports in an hour, *When* the 6th is requested, *Then* it is rate limited.
- **AC-9.** *Given* an enumeration burst (>30 % `not_found` over 15 minutes), *When* the alert
  evaluates, *Then* it fires and links to the runbook section.
- **AC-10.** *Given* 60 redemptions of one coupon in 10 minutes, *When* the velocity alert
  evaluates, *Then* it fires.
- **AC-11.** *Given* the help pages are published, *When* a learner searches "coupon", *Then* both
  pages are findable and every one of the nine reasons has a matching explanation.
- **AC-12.** *Given* the GA commit, *When* a tenant with no explicit setting loads platform
  features, *Then* `ffCourseCoupons` is true; *and given* a tenant that set it false explicitly,
  *Then* it stays false.
- **AC-13.** *Given* Redis is unavailable, *When* an apply request arrives, *Then* it is served
  (fail-open on the limiter) and the coupon's own redemption cap is still enforced correctly.
- **AC-14.** *Given* the security threat-model note, *When* the security owner reviews it, *Then*
  sign-off is recorded before the flag flip commit merges.

## 8. Data Model

```sql
-- MKTC.7 — bounded attempt log for enumeration detection.
CREATE TABLE IF NOT EXISTS billing.coupon_attempts (
    id         BIGSERIAL PRIMARY KEY,
    user_id    UUID REFERENCES "user".users (id) ON DELETE CASCADE,
    course_id  UUID REFERENCES course.courses (id) ON DELETE CASCADE,
    code_hash  TEXT NOT NULL,          -- salted hash; raw code never stored for unknown codes
    reason     TEXT NOT NULL,
    ip_prefix  TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_coupon_attempts_user_course
    ON billing.coupon_attempts (user_id, course_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_coupon_attempts_recent
    ON billing.coupon_attempts (created_at DESC);

COMMENT ON TABLE billing.coupon_attempts IS
    'Bounded 30-day log of failed coupon applications for enumeration detection (plan MKTC.7).';

-- Platform-level discount ceiling.
ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS coupon_max_percent_off NUMERIC(5,2);

COMMENT ON COLUMN settings.platform_app_settings.coupon_max_percent_off IS
    'Optional cap on creator coupon percent discounts; NULL/100 = uncapped (plan MKTC.7).';
```

Retention: a scheduled job deletes `coupon_attempts` rows older than 30 days, registered alongside
the MKTC.1 reservation sweeper. No backfill.

## 9. API Surface

| Verb | Path | Change |
|---|---|---|
| GET | `/api/v1/courses/{course_code}/coupons/summary` | **new** — per-coupon performance figures |
| GET | `/api/v1/courses/{course_code}/coupons/{coupon_id}/redemptions.csv` | **new** — streamed export |
| POST | `/api/v1/marketplace/courses/{slug}/coupon/preview` | 429 body gains `reason: "cooldown"` and `Retry-After` |
| PUT | `/api/v1/platform/settings` | accepts `couponMaxPercentOff` |

```ts
type CouponSummaryRow = {
  couponId: string
  code: string
  redeemedCount: number
  refundedCount: number
  grossListCents: number      // sum of list prices redeemed
  discountCents: number       // total discount given
  netChargedCents: number     // excludes refunds
  currency: string
  firstRedeemedAt: string | null
  lastRedeemedAt: string | null
}
type CouponSummaryResponse = { rows: CouponSummaryRow[]; currency: string }
```

CSV columns: `redeemed_at,status,learner_name,learner_email,code,list_price_cents,discount_cents,charged_cents,currency`.
All new routes are gated by `ffCourseCoupons` and `course:{code}:item:create`, documented in
OpenAPI, and added to the route inventory.

## 10. UI / UX

- **Creator table (MKTC.4)** gains a compact performance cluster per row — "34 claimed · $180 off ·
  $1,020 net" — with the full breakdown in the redemptions drawer header. Numbers are formatted with
  the course currency and have text alternatives; a `—` state renders for coupons with no
  redemptions.
- **Redemptions drawer** gains an **Export CSV** button (labelled "Export redemptions for LAUNCH25")
  with a pending state and an error path; on rate limit it explains the hourly cap.
- **Learner cool-down copy** — `marketplace.coupon.cooldown`: "Too many attempts. Try again in
  15 minutes." rendered in the same `role="alert"` slot as other reasons (MKTC.5).
- **Admin platform settings** — the `ffCourseCoupons` toggle gains a description noting the GA
  default change, and a new "Maximum coupon discount" numeric field with help text.
- **Help centre** — two new articles, cross-linked, with screenshots from MKTC.4/MKTC.5.
- All UI follows the MKTC.4 accessibility rules (real table semantics, discernible names, live
  region for export completion).

## 11. AI / ML Considerations

Not AI-touching. Fraud detection is deterministic by design — no model, no training data, no
scoring — which is a stated constraint so that a decline is always explainable to a creator.

## 12. Integration Points

- **External** — Grafana / the existing observability stack (`docker-compose.observability.yml`),
  Redis (optional, for cross-replica limits), Sentry for limiter errors.
- **Internal** —
  `server/internal/service/billing/coupon_limits.go` (**new**),
  `server/internal/repos/billing/coupon_attempts.go` (**new**),
  `server/internal/repos/billing/coupon_summary.go` (**new**),
  `server/internal/httpserver/course_coupons_http.go` (summary + CSV routes),
  `server/internal/httpserver/marketplace_purchase_http.go` (cool-down on preview),
  `server/internal/repos/platformconfig/*` (`coupon_max_percent_off`, flag default flip),
  `server/internal/scheduler/` (attempt-log retention job),
  `server/internal/telemetry/default.go` (metrics),
  `clients/web/src/pages/lms/course-coupons-panel.tsx` +
  `course-coupon-redemptions-drawer.tsx` (performance + export),
  `clients/web/src/components/settings/platform-settings-panel.tsx` (ceiling field),
  `docs/help/`, `docs/runbooks/coupons.md`, `docs/monitoring/`.
- **Events** — audit events for export and for ceiling changes.

## 13. Dependencies & Sequencing

- **Must ship after** — every other MKTC story; the GA flip in particular must not precede MKTC.6,
  or mobile learners meet coupons the app cannot handle.
- **Must ship before** — nothing; this closes the epic.
- **Shared infra** — Redis (optional), Grafana, the scheduler, the help centre publishing pipeline.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Rate limits block legitimate learners at a live event (a room of 200 people typing one code) | M | H | Limits are per-user, not per-code; successful applies do not count toward the cool-down; velocity alerting is informational, never blocking |
| Enumeration finds a working code anyway | M | M | Entropy guidance + warning (FR-4), caps bound the loss, velocity alert catches exploitation, kill switch in the runbook |
| Attempt log becomes a PII liability | M | M | Hashed codes, IP prefixes only, 30-day retention job, listed in the data map |
| CSV export leaks learner emails | M | M | Same authorization as the roster, audit logged, 5/hour cap |
| Flag flip surprises tenants mid-term | M | M | Release notes + admin panel copy + explicit-setting preservation (AC-12); flip in its own revertable commit |
| Limiter outage causes oversell | L | H | Caps live in Postgres, not the limiter; AC-13 pins fail-open behaviour to guessing only |
| Dashboards go stale after the epic | M | L | Alert thresholds and panels are listed in the runbook with an owner |

## 15. Rollout Plan

- **Feature flag** — `ffCourseCoupons`. This story is where it becomes default ON.
- **GA bar** (all must hold before the flip commit merges):
  1. MKTC.1–MKTC.6 shipped and green in CI.
  2. Dogfood: ≥ 20 real redemptions across ≥ 3 internal courses, including one refund and one
     100 %-off grant, with the reconciliation query returning zero rows.
  3. Zero P1/P2 bugs open against the epic for 7 days.
  4. Accessibility audits (web axe + mobile) clean.
  5. Security sign-off on the enumeration threat model (FR-17, AC-14).
  6. Help pages published; support macro written.
  7. Dashboards and all four alerts live and verified by a synthetic trigger.
- **Staging** — enable for internal tenants → 10 % of tenants for 7 days → all tenants (flip the
  default) → remove the "beta" label from the admin panel copy after 30 days.
- **Comms** — release note to creators ("Coupon codes are here"), an in-product hint on the
  marketplace settings page for courses with a price and no coupons, and a support briefing.
- **Rollback** — set `ff_course_coupons=false` platform-wide (instant, no deploy). Redeemed
  entitlements remain valid — learners keep what they bought. Revert the default-flip commit if the
  rollback needs to be durable.

## 16. Test Plan

- **Unit** — limiter bucket arithmetic for all three tiers and the cool-down state machine
  (consecutive-failure counting, reset on success, expiry); code-hash helper (salt applied, stable,
  non-reversible); ceiling validation; summary aggregation math including refund exclusion; CSV row
  encoding and escaping (commas, quotes, non-ASCII names).
- **Integration** (DB) — attempt rows written with hashed codes and IP prefixes; retention job
  deletes past 30 days; summary endpoint reconciles against the earnings ledger for a mixed set of
  discounted, full-price and refunded sales; export authorization matrix; export rate limit.
- **End-to-end** — creator views performance figures after a scripted set of redemptions and
  exports the CSV; learner hits the cool-down and sees the right copy; flag flip verified by
  toggling the platform setting and reloading.
- **Security** — enumeration simulation (1,000 guesses) confirming limits, cool-down, alert firing
  and that no raw unknown code is persisted; verify limits cannot be bypassed by rotating course
  slug or by a second session; verify the ceiling cannot be raised by a non-admin.
- **Accessibility** — axe on the performance column, the export control and the settings field;
  screen-reader check that performance numbers read as sentences, not bare digits.
- **Performance / load** — summary endpoint with 100 coupons and 10,000 redemptions (single
  aggregate, p95 < 200 ms); CSV export of 10,000 rows streams without exceeding the memory budget;
  limiter overhead measured < 5 ms.
- **Manual exploratory** — trigger each alert synthetically and walk its runbook section end to
  end; simulate a Redis outage during a purchase burst; flip the flag off mid-checkout and confirm
  in-flight reservations still complete.

## 17. Documentation & Training

- **End-user docs** — "Using a coupon code" (learner), covering entry, links, every failure reason,
  mobile differences, and what happens on refund.
- **Admin / instructor docs** — "Create and share coupon codes" (creator), plus an admin note on
  the discount ceiling and the platform flag.
- **API reference** — summary and CSV endpoints in `openapi.json`; final entry in
  `docs/api-changelog-course-coupons.md` marking the epic complete.
- **Internal runbook** — `docs/runbooks/coupons.md` completed: reconciliation queries, stuck
  reservation, leaked code kill switch, "discount didn't apply" triage, alert-by-alert response,
  feature disable procedure.
- **Training** — 20-minute support walkthrough with the macro and the runbook; a short creator-facing
  changelog post.

## 18. Open Questions

1. Are the proposed limits (15/min, 60/h, 100/h, 200/h per IP, 10-failure cool-down) right for a
   live classroom where 200 people type the same code at once? They are per-user, so the answer
   should be yes — validate with one real event before the 10 % rollout.
2. Should the discount ceiling default to something below 100 % (e.g. 90 %) to make free-grant
   mistakes harder, with an explicit opt-in for comped cohorts?
3. Should coupon performance appear in the creator earnings page as well as the coupon table?
   (Proposed: link from earnings to the coupon table rather than duplicating figures.)
4. Should the attempt log feed a shared abuse signal used elsewhere (sign-in, affiliate codes), or
   stay coupon-local? (Proposed: coupon-local now; note the extension point.)
5. Do we announce coupons in-product to all creators at GA, or only to creators with a paid course?
   (Proposed: paid-course creators only, to avoid noise.)
6. Should refunded redemptions release the seat (MKTC.3 FR-11) *and* be excluded from performance
   figures (FR-8)? The two are consistent as proposed, but confirm with finance that the ledger
   presentation matches their reporting.

## 19. References

- Existing files: `server/internal/httpserver/billing_http.go:23-48` (limiter shape to generalize),
  `server/internal/redisclient/`, `server/internal/telemetry/default.go`,
  `server/internal/scheduler/`, `docker-compose.observability.yml`, `docs/monitoring/`,
  `docs/runbooks/`, `clients/web/src/components/settings/platform-settings-panel.tsx`,
  `server/internal/repos/platformconfig/features.go` (the default-flip site).
- Standards: OWASP ASVS V11 (business-logic abuse) and V4 (access control), NIST SP 800-63B rate
  limiting guidance, GDPR Art. 5(1)(c)/(e) (data minimisation and storage limitation) for the
  attempt log.
- Related plans: [MKTC.1](../../completed/marketplace/MKTC.1-coupon-data-model-and-discount-engine.md) …
  [MKTC.6](MKTC.6-mobile-coupon-redemption.md) (completed),
  [17.7 observability](../../completed/17-platform-performance-operability/),
  the [S-series standards folder](../standards/README.md) for retention and data-map obligations.
