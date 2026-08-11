# MKTC.1 — Coupon Data Model, Redemption Ledger & Discount Engine

> Implementation plan. Source: [docs/plan/marketplace/README.md](../../plan/marketplace/README.md). Part of the MKTC Course Coupon Codes epic.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | MKTC.1 |
| **Section** | Marketplace |
| **Severity** | MAJOR |
| **Markets** | HS (primary) · HE · K12 |
| **Status (today)** | COMPLETE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Commerce / Growth squad (backend) |
| **Depends on** | MKT1 (marketplace foundation), MKT4 (purchase & entitlement flow), 15.3 (Stripe billing) |
| **Unblocks** | MKTC.2, MKTC.3, MKTC.4, MKTC.5, MKTC.6, MKTC.7 |

---

## 1. Problem Statement

A course creator today can set exactly one number — `course.courses.price_cents` — and every learner
pays it. There is no way to run a launch promotion, give a cohort of 30 homeschool families a
discount, or hand a conference audience a code that expires Friday. The only discount mechanism in
the codebase is Stripe's own `AllowPromotionCodes: true` flag in
`server/internal/service/paymentprovider/stripe.go:42`, which requires the creator to have Stripe
dashboard access they do not have (the platform owns the Stripe account) and cannot be scoped to one
course. The result is lost conversions on paid marketplace courses and a recurring support request.
This story lays the foundation: the tables, the redemption ledger that makes "first 50 people"
actually mean 50, and a pure, unit-testable discount engine.

## 2. Goals

- Persist **many coupons per course**, each with a code, a percent-or-fixed discount, an optional
  active window, an optional total redemption cap, and a per-learner cap.
- Provide a **redemption ledger** that makes the cap correct under concurrency — no oversell when 60
  people click Buy in the same second.
- Provide a **pure discount function** (`price × coupon → discount, charged`) with correct
  minor-unit rounding for zero-decimal currencies (JPY) and a defined clamp at the payment
  provider's minimum charge.
- Provide a **pure eligibility function** returning a typed reason (`ok`, `not_found`, `inactive`,
  `not_started`, `expired`, `exhausted`, `already_used`, `currency_mismatch`, `course_free`,
  `owned`) so every caller renders the same vocabulary.
- Add the `coupon` acquisition source to entitlements so a 100 %-off grant is distinguishable from
  a genuinely free course in reporting.

## 3. Non-Goals

- Any HTTP route (MKTC.2 for creator CRUD, MKTC.3 for learner apply/checkout).
- Any UI (MKTC.4/MKTC.5/MKTC.6).
- Mirroring coupons into Stripe Coupon / Promotion Code objects — explicitly rejected in the
  [epic README](../../plan/marketplace/README.md).
- Platform-wide or org-wide coupons, subscription-plan coupons, gift cards, bundle/learning-path
  pricing, automatic scheduled price changes.
- Stacking two coupons on one purchase.

## 4. Personas & User Stories

- **As a course creator**, I want a code I create to stop working the moment my window closes or my
  seat count runs out, so that a screenshot of my code on social media cannot cost me money forever.
- **As a homeschool parent**, I want the code from my co-op to take the advertised amount off, so
  that I trust the checkout total.
- **As a platform admin**, I want every discount to leave a ledger row, so that revenue reporting
  reconciles against Stripe.
- **As a backend engineer**, I want cap enforcement to live in one repo function under a row lock,
  so that no future caller can oversell by forgetting a check.
- **As a finance analyst**, I want to distinguish "free course" from "paid course claimed with a
  100 % code", so that conversion and discount cost are reportable.

## 5. Functional Requirements

- **FR-1.** The system MUST store coupons in a new table `billing.course_coupons`, scoped by
  `course_id`, with columns: `code`, `discount_type`, `percent_off`, `amount_off_cents`, `currency`,
  `starts_at`, `ends_at`, `max_redemptions`, `max_redemptions_per_user`, `redeemed_count`,
  `status`, `note`, `created_by`, `created_at`, `updated_at`.
- **FR-2.** `code` MUST be stored **upper-case** and match `^[A-Z0-9][A-Z0-9_-]{3,31}$` (4–32 chars),
  enforced by a `CHECK` constraint, not only in Go. Normalization (trim, upper-case, collapse
  internal whitespace to nothing) MUST be a single exported helper `coupons.NormalizeCode`.
- **FR-3.** A partial unique index MUST make `(course_id, code)` unique among non-archived rows, so
  a retired code can be re-issued later without a hard delete.
- **FR-4.** `discount_type` MUST be exactly one of `percent` or `fixed`, with a `CHECK` that
  `percent` rows carry `percent_off ∈ (0, 100]` and null `amount_off_cents`, and `fixed` rows carry
  `amount_off_cents > 0`, a non-null `currency`, and null `percent_off`.
- **FR-5.** `status` MUST be one of `active`, `disabled`, `archived`. Only `active` coupons are
  redeemable; `disabled` is a reversible creator action; `archived` is the soft delete.
- **FR-6.** The system MUST store redemptions in `billing.coupon_redemptions` with `coupon_id`,
  `course_id`, `user_id`, `entitlement_id`, `status` (`reserved` | `redeemed` | `released`),
  `checkout_session_id`, `provider_event_id`, `list_price_cents`, `discount_cents`,
  `charged_cents`, `currency`, `reserved_at`, `redeemed_at`, `released_at`, `expires_at`.
- **FR-7.** The repo MUST expose `Reserve(ctx, tx, couponID, userID, quote) (*Redemption, error)`
  that takes `SELECT ... FOR UPDATE` on the coupon row, re-checks every eligibility rule against
  live counts, and inserts a `reserved` row with `expires_at = now() + COUPON_RESERVATION_TTL`.
  Concurrency correctness MUST come from the lock, never from an application-level read-then-write.
- **FR-8.** The repo MUST expose `Redeem(ctx, pool, in RedeemInput)` (idempotent on
  `provider_event_id`) promoting a reservation to `redeemed`, and `Release(ctx, pool, ...)` moving
  it to `released`. Both MUST keep `course_coupons.redeemed_count` in step in the same transaction.
- **FR-9.** Consumed seats MUST be counted as rows in (`reserved`, `redeemed`) whose `expires_at` is
  null or in the future. `released` rows and expired reservations MUST NOT consume a seat.
- **FR-10.** A sweeper MUST release reservations past `expires_at` (`ReleaseExpiredReservations`),
  runnable from the existing scheduler; correctness MUST NOT depend on the sweeper having run —
  the count query itself excludes expired rows.
- **FR-11.** The system MUST expose a pure function
  `coupons.ApplyDiscount(listCents int, currency string, c Coupon) Quote` returning
  `{ListCents, DiscountCents, ChargedCents, Currency}`. Percent discounts round **half up** to the
  currency's minor unit via `internal/currency`. Fixed discounts are clamped to `listCents`.
  `ChargedCents` is never negative.
- **FR-12.** When `0 < ChargedCents < currency.MinimumChargeCents(currency)` (the provider floor
  already encoded in `currency.ValidateCatalogPrice`), `ApplyDiscount` MUST clamp `ChargedCents` to
  `0` and raise `DiscountCents` to the full list price, and MUST set `Quote.ClampedToFree = true`
  so callers can log/telemeter it. Rationale: the alternative is a Stripe session the provider
  rejects at redirect time. See §18 Q1.
- **FR-13.** The system MUST expose a pure function
  `coupons.Evaluate(now time.Time, c *Coupon, ctx EvalContext) Eligibility` returning a typed
  `Reason` from the fixed vocabulary in §2, where `EvalContext` carries course price/currency,
  consumed count, this-user count, and whether the user already owns the course. `Evaluate` MUST
  take no database handle.
- **FR-14.** `billing.user_entitlements.acquisition_source` MUST accept a new value `coupon`
  (migration alters the existing CHECK added by `368_course_marketplace.sql`), used when a
  100 %-off code grants access without a payment provider.
- **FR-15.** A coupon MUST be redeemable only against the course it belongs to; there is no
  cross-course or global coupon in this story.
- **FR-16.** Deleting a course MUST cascade-delete its coupons and redemptions
  (`ON DELETE CASCADE`); deleting a user MUST cascade-delete their redemptions but MUST NOT delete
  the coupon (`created_by` is `ON DELETE SET NULL`).
- **FR-17.** The repo MUST expose read helpers used by later stories:
  `ListByCourse`, `GetByID`, `GetByCourseAndCode` (normalized), `CountsForCoupons` (batch consumed
  counts for the manager table), and `ListRedemptions(couponID, cursor, limit)`.

## 6. Non-Functional Requirements

- **Performance** — `GetByCourseAndCode` p95 < 10 ms (unique index hit). `Reserve` p95 < 25 ms
  including the row lock. `CountsForCoupons` MUST be a single grouped query for the whole page of
  coupons, not N+1.
- **Security** — Discount amounts are derived from DB rows only; no input from the request body
  participates in the arithmetic beyond the code string. Codes are not secrets but MUST be compared
  after normalization to avoid case/whitespace bypass of the uniqueness constraint. `created_by` is
  recorded for audit.
- **Privacy & Compliance** — `coupon_redemptions` links a learner to a purchase and is therefore a
  financial record under the 15.13 retention policy; it inherits the same retention window as
  `user_entitlements`. No new PII fields. FERPA: redemption rows are not education records but are
  covered by the existing DSAR export for billing data (S-series), so add the two tables to the
  DSAR/erasure inventory.
- **Accessibility** — No UI in this story.
- **Scalability** — Row lock is per-coupon, so contention is bounded by concurrent buyers of one
  coupon. Expected worst case is a launch-day code with a few hundred concurrent reservations;
  Postgres handles this serially at < 25 ms each. Indexes: `(course_id, status, created_at DESC)`,
  `(coupon_id, status)`, `(user_id, coupon_id, status)`.
- **Reliability** — Reservation TTL guarantees seats return without operator action. `Redeem` is
  idempotent on `provider_event_id` (partial unique index), so Stripe webhook re-delivery cannot
  double-count.
- **Observability** — Repo-level counters emitted from `internal/telemetry`:
  `coupon_reserve_total{result}`, `coupon_redeem_total{result}`, `coupon_release_total{reason}`,
  `coupon_reservation_expired_total`. Structured logs carry `coupon_id`, `course_id`, `user_id`,
  never the discount arithmetic inputs of another tenant.
- **Maintainability** — New package `server/internal/service/coupons` holds the pure engine (no
  `pgx` import, so it is trivially testable); SQL lives in `server/internal/repos/billing/coupons.go`
  and `coupon_redemptions.go`, both under the 600-LOC file budget and inside the existing
  `repos/billing` package (currently 5 files, well under the 40-file package budget).
- **Internationalization** — `Reason` values are stable machine tokens; all human copy is produced
  by clients from i18n keys. Currency handling delegates to `internal/currency`; timestamps are
  `TIMESTAMPTZ` and compared in UTC.
- **Backward compatibility** — Purely additive: two new tables, one widened CHECK constraint. No
  existing column changes type or meaning. A down migration drops the two tables and restores the
  original `acquisition_source` CHECK (after asserting no `coupon` rows exist).

## 7. Acceptance Criteria

- **AC-1.** *Given* a coupon `LAUNCH25` (25 % off) on a $40 (4000¢ USD) course, *When*
  `ApplyDiscount` runs, *Then* it returns `discount=1000`, `charged=3000`.
- **AC-2.** *Given* a percent coupon producing a fractional minor unit (33 % of 999¢),
  *When* `ApplyDiscount` runs, *Then* the discount rounds half up to 330 and `charged=669`, and for
  a JPY course the same call operates on whole yen with no fractional unit.
- **AC-3.** *Given* a fixed coupon of 5000¢ on a 3000¢ course, *When* `ApplyDiscount` runs, *Then*
  `discount=3000`, `charged=0`, `clampedToFree=false` (exact zero, not a clamp).
- **AC-4.** *Given* a 99 % coupon on a 4000¢ USD course (charged would be 40¢, below the 50¢ Stripe
  floor), *When* `ApplyDiscount` runs, *Then* `charged=0`, `discount=4000`, `clampedToFree=true`.
- **AC-5.** *Given* a coupon with `max_redemptions=1`, *When* two goroutines call `Reserve`
  concurrently, *Then* exactly one succeeds and the other returns `Reason=exhausted` — asserted by
  a DB integration test using two connections.
- **AC-6.** *Given* a reservation older than the TTL, *When* a new learner calls `Reserve`, *Then*
  the seat is available even though the sweeper has not run.
- **AC-7.** *Given* the same `provider_event_id` is passed to `Redeem` twice, *When* both complete,
  *Then* there is one `redeemed` row and `redeemed_count` incremented once.
- **AC-8.** *Given* a coupon whose `ends_at` is in the past, *When* `Evaluate` runs, *Then*
  `Reason=expired`; *and given* `starts_at` in the future, *Then* `Reason=not_started`.
- **AC-9.** *Given* a learner who already redeemed a coupon with `max_redemptions_per_user=1`,
  *When* `Evaluate` runs for them, *Then* `Reason=already_used`, while another learner gets `ok`.
- **AC-10.** *Given* a fixed-amount coupon in `usd` on a course priced in `eur`, *When* `Evaluate`
  runs, *Then* `Reason=currency_mismatch`.
- **AC-11.** *Given* a lower-case code `launch25` with surrounding spaces, *When*
  `NormalizeCode` runs, *Then* it returns `LAUNCH25`, and an insert of `Launch25` for the same
  course violates the unique index.
- **AC-12.** *Given* the migration is applied and rolled back on a database containing entitlements,
  *When* `make test` runs the migration suite, *Then* both directions succeed and no existing row
  is modified.

## 8. Data Model

New migration pair `server/migrations/472_course_coupons.sql` / `472_course_coupons.down.sql`
(next free number after `471_revert_user_nav_preferences`; re-check at implementation time).

```sql
-- MKTC.1 — course-scoped coupon codes and redemption ledger.

CREATE TABLE IF NOT EXISTS billing.course_coupons (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id                UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    code                     TEXT NOT NULL,
    discount_type            TEXT NOT NULL,
    percent_off              NUMERIC(5,2),
    amount_off_cents         INT,
    currency                 TEXT,
    starts_at                TIMESTAMPTZ,
    ends_at                  TIMESTAMPTZ,
    max_redemptions          INT,
    max_redemptions_per_user INT NOT NULL DEFAULT 1,
    redeemed_count           INT NOT NULL DEFAULT 0,
    status                   TEXT NOT NULL DEFAULT 'active',
    note                     TEXT,
    created_by               UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT billing_course_coupons_code_shape_check
        CHECK (code ~ '^[A-Z0-9][A-Z0-9_-]{3,31}$'),
    CONSTRAINT billing_course_coupons_status_check
        CHECK (status IN ('active', 'disabled', 'archived')),
    CONSTRAINT billing_course_coupons_kind_check CHECK (
        (discount_type = 'percent'
             AND percent_off IS NOT NULL AND percent_off > 0 AND percent_off <= 100
             AND amount_off_cents IS NULL)
        OR (discount_type = 'fixed'
             AND amount_off_cents IS NOT NULL AND amount_off_cents > 0
             AND currency IS NOT NULL AND percent_off IS NULL)
    ),
    CONSTRAINT billing_course_coupons_window_check
        CHECK (ends_at IS NULL OR starts_at IS NULL OR ends_at > starts_at),
    CONSTRAINT billing_course_coupons_max_check
        CHECK (max_redemptions IS NULL OR max_redemptions > 0),
    CONSTRAINT billing_course_coupons_per_user_check
        CHECK (max_redemptions_per_user > 0 AND max_redemptions_per_user <= 100)
);

COMMENT ON TABLE billing.course_coupons IS
    'Creator-managed discount codes scoped to one marketplace course (plan MKTC.1).';

CREATE UNIQUE INDEX IF NOT EXISTS uq_course_coupons_code
    ON billing.course_coupons (course_id, code)
    WHERE status <> 'archived';

CREATE INDEX IF NOT EXISTS idx_course_coupons_course
    ON billing.course_coupons (course_id, status, created_at DESC);

CREATE TABLE IF NOT EXISTS billing.coupon_redemptions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    coupon_id           UUID NOT NULL REFERENCES billing.course_coupons (id) ON DELETE CASCADE,
    course_id           UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    user_id             UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    entitlement_id      UUID REFERENCES billing.user_entitlements (id) ON DELETE SET NULL,
    status              TEXT NOT NULL DEFAULT 'reserved',
    checkout_session_id TEXT,
    provider_event_id   TEXT,
    list_price_cents    INT NOT NULL,
    discount_cents      INT NOT NULL,
    charged_cents       INT NOT NULL,
    currency            TEXT NOT NULL,
    reserved_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at          TIMESTAMPTZ,
    redeemed_at         TIMESTAMPTZ,
    released_at         TIMESTAMPTZ,
    CONSTRAINT billing_coupon_redemptions_status_check
        CHECK (status IN ('reserved', 'redeemed', 'released')),
    CONSTRAINT billing_coupon_redemptions_amounts_check
        CHECK (discount_cents >= 0 AND charged_cents >= 0 AND list_price_cents >= 0)
);

COMMENT ON TABLE billing.coupon_redemptions IS
    'Per-learner coupon reservations and redemptions; the authority for redemption caps (plan MKTC.1).';

CREATE UNIQUE INDEX IF NOT EXISTS uq_coupon_redemption_session
    ON billing.coupon_redemptions (checkout_session_id)
    WHERE checkout_session_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_coupon_redemption_event
    ON billing.coupon_redemptions (provider_event_id)
    WHERE provider_event_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_coupon_redemptions_coupon
    ON billing.coupon_redemptions (coupon_id, status);

CREATE INDEX IF NOT EXISTS idx_coupon_redemptions_user
    ON billing.coupon_redemptions (user_id, coupon_id, status);

-- Distinguish a 100%-off grant from a genuinely free course (plan MKTC.1 FR-14).
ALTER TABLE billing.user_entitlements
    DROP CONSTRAINT IF EXISTS billing_user_entitlements_acquisition_source_check;

ALTER TABLE billing.user_entitlements
    ADD CONSTRAINT billing_user_entitlements_acquisition_source_check
        CHECK (acquisition_source IN ('stripe', 'free', 'comp', 'coupon'));
```

**Backfill** — none. Every existing course has zero coupons; every existing entitlement keeps its
current `acquisition_source`.

**Down migration** — `DROP TABLE billing.coupon_redemptions; DROP TABLE billing.course_coupons;`
then restore the three-value CHECK guarded by
`UPDATE billing.user_entitlements SET acquisition_source='free' WHERE acquisition_source='coupon';`
so the constraint can be re-added on a database that already granted coupon entitlements.

## 9. API Surface

None. This story ships packages, not routes. The Go surface consumed by MKTC.2/MKTC.3:

```go
// server/internal/service/coupons (pure — no pgx import)
type Kind string // "percent" | "fixed"

type Coupon struct {
    ID, CourseID          uuid.UUID
    Code                  string
    Kind                  Kind
    PercentOff            float64 // 0 when fixed
    AmountOffCents        int     // 0 when percent
    Currency              string  // set when fixed
    StartsAt, EndsAt      *time.Time
    MaxRedemptions        *int
    MaxRedemptionsPerUser int
    Status                string
}

type Quote struct {
    ListCents, DiscountCents, ChargedCents int
    Currency                               string
    ClampedToFree                          bool
}

type Reason string // ok | not_found | inactive | not_started | expired |
                   // exhausted | already_used | currency_mismatch | course_free | owned

type EvalContext struct {
    Now            time.Time
    CoursePrice    int
    CourseCurrency string
    ConsumedSeats  int
    UserSeats      int
    AlreadyOwned   bool
}

func NormalizeCode(raw string) string
func ValidateCode(code string) error
func ApplyDiscount(listCents int, currency string, c Coupon) Quote
func Evaluate(c *Coupon, ec EvalContext) (Reason, Quote)

// server/internal/repos/billing (SQL)
func ListCouponsByCourse(ctx, pool, courseID, includeArchived bool) ([]Coupon, error)
func GetCouponByID(ctx, pool, id) (*Coupon, error)
func GetCouponByCourseAndCode(ctx, pool, courseID, normalizedCode) (*Coupon, error)
func CouponSeatCounts(ctx, pool, couponIDs []uuid.UUID) (map[uuid.UUID]SeatCount, error)
func UserSeatCount(ctx, pool, couponID, userID) (int, error)
func CreateCoupon(ctx, pool, in CreateCouponInput) (*Coupon, error)
func UpdateCoupon(ctx, pool, in UpdateCouponInput) (*Coupon, error)
func SetCouponStatus(ctx, pool, id, status string) error
func ReserveCoupon(ctx, pool, in ReserveInput) (*Redemption, coupons.Reason, error)
func RedeemCoupon(ctx, pool, in RedeemInput) (*Redemption, bool, error) // bool = created
func ReleaseCouponReservation(ctx, pool, id uuid.UUID, reason string) error
func ReleaseExpiredCouponReservations(ctx, pool, now time.Time) (int, error)
func ListCouponRedemptions(ctx, pool, couponID, cursor, limit) ([]Redemption, string, error)
```

No rate-limit or OpenAPI surface changes in this story (MKTC.2/MKTC.3 own those).

## 10. UI / UX

None. The only user-visible artifact is the vocabulary of `Reason` tokens, which MKTC.4/MKTC.5 map
to i18n keys (`marketplace.coupon.reason.expired`, `…exhausted`, …). Fixing the vocabulary here
prevents each client inventing its own copy.

## 11. AI / ML Considerations

Not AI-touching.

## 12. Integration Points

- **External** — none directly; the amounts produced here are consumed by
  `service/paymentprovider` in MKTC.3.
- **Internal** —
  `server/migrations/472_course_coupons.sql` (new),
  `server/internal/service/coupons/` (new package: `coupon.go`, `discount.go`, `evaluate.go`, tests),
  `server/internal/repos/billing/coupons.go` + `coupon_redemptions.go` (new files in the existing
  package), `server/internal/currency/exponent.go` (extend with `MinimumChargeCents`),
  `server/internal/telemetry/default.go` (new counters),
  `server/internal/scheduler/` (register the reservation sweeper job).
- **Events** — none emitted in this story; MKTC.3 emits purchase/notification events.

## 13. Dependencies & Sequencing

- **Must ship after** — MKT1 (`billing.user_entitlements` generalization, `acquisition_source`),
  MKT4 (purchase flow this plugs into), 15.3 (`billing` schema).
- **Must ship before** — MKTC.2, MKTC.3 (and transitively everything else in the epic).
- **Shared infra** — Postgres only. The reservation sweeper uses the existing scheduler/job queue
  (`APP_ENV=local` runs it in-process; production runs it on the API worker).

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Oversell of a capped coupon under concurrency | M | H | `SELECT … FOR UPDATE` on the coupon row inside `Reserve`; concurrency test with two connections (AC-5) |
| Abandoned checkouts permanently burn seats | H | M | TTL on reservations + count query excludes expired rows + sweeper (FR-9, FR-10, AC-6) |
| Rounding disputes ("your 30 % took 29.99 %") | M | M | Single half-up rounding helper, currency-aware, unit-tested per currency; the quote surfaces exact cents to the client |
| Discounted amount below provider minimum → checkout fails at redirect | M | M | Clamp to free (FR-12) plus a creator-side warning in MKTC.4; telemetry counter on clamps |
| Down migration blocked by `coupon` entitlement rows | M | L | Down script rewrites `coupon` → `free` before restoring the CHECK |
| `redeemed_count` drifts from the ledger | L | M | Counter only ever written in the same tx as a ledger transition; a reconciliation query ships in MKTC.7's runbook |
| Repos/billing package growth | L | L | Two new files, package stays far under the 40-file budget; pure logic lives in `service/coupons` |

## 15. Rollout Plan

- **Feature flag** — `ffCourseCoupons` (`settings.platform_app_settings.ff_course_coupons`,
  default **OFF**) is introduced in MKTC.2 where the first route appears. This story's migration
  ships unflagged because it is inert without routes.
- **Sequencing** — schema → repo + engine → unit/integration tests green → merge. No backfill, no
  code path reads the tables yet.
- **Dogfood** — seed a coupon by SQL on an internal course and exercise `Reserve`/`Redeem` from a
  DB test; verify counts.
- **GA criteria** — concurrency test green in CI; migration up/down verified on a copy of a staging
  database; `make lint-structure` and `golangci-lint run ./...` clean.
- **Rollback** — drop the two tables via the down migration; nothing else references them.

## 16. Test Plan

- **Unit** (`service/coupons`, no DB) — percent/fixed math across `usd`/`eur`/`jpy`; half-up
  rounding boundaries (`0.5¢` cases); fixed discount larger than price; zero and negative guards;
  provider-floor clamp; `NormalizeCode` (case, spaces, unicode look-alikes rejected);
  `ValidateCode` shape errors; `Evaluate` for each of the ten `Reason` values; window boundaries
  evaluated exactly at `starts_at`/`ends_at` (inclusive start, exclusive end).
- **Integration** (DB, `*_db_test.go` in `repos/billing`) — create/list/update/status transitions;
  unique index rejects a duplicate normalized code and permits re-use after archive; `Reserve`
  under two concurrent transactions (AC-5); TTL expiry frees a seat without the sweeper (AC-6);
  `Redeem` idempotency on `provider_event_id` (AC-7); `Release` returns the seat and decrements
  `redeemed_count`; `CouponSeatCounts` is one query for N coupons (assert via query counter);
  cascade behaviour on course delete and user delete.
- **End-to-end** — none in this story (no route or UI).
- **Security** — verify no repo function accepts a caller-supplied discount or charged amount;
  fuzz `NormalizeCode` with control characters, RTL marks, and 10 KB input.
- **Accessibility** — n/a.
- **Performance / load** — `pgbench`-style loop: 200 concurrent `Reserve` calls on one coupon with
  `max_redemptions=50` → exactly 50 succeed, p95 < 25 ms.
- **Manual exploratory** — apply the migration to a database with existing entitlements and
  purchases; confirm `\d billing.user_entitlements` shows the widened CHECK and existing rows are
  untouched.

## 17. Documentation & Training

- **End-user docs** — none yet (MKTC.7 owns the help-centre article once the UI exists).
- **Admin / instructor docs** — none yet.
- **API reference** — none yet.
- **Internal runbook** — add `docs/runbooks/coupons.md` stub with: how to inspect a coupon's seat
  usage, how to manually release a stuck reservation, and the reconciliation query comparing
  `redeemed_count` to the ledger.
- **Engineering docs** — a short "how discounts are computed" note in
  `docs/marketplace-courses-authoring.md` pointing at `service/coupons`.

## 18. Open Questions

1. **Sub-minimum charge policy.** FR-12 clamps to free. The alternative — reject the code with
   "cannot be applied to this price" — protects revenue but produces a confusing dead end. Product
   to confirm the clamp before MKTC.3 hardens the checkout path.
2. **Reservation TTL default.** 30 minutes is proposed (Stripe Checkout sessions expire at 24 h,
   but real abandonment is minutes). Should this be a platform setting rather than a constant?
3. **Per-user cap default.** Proposed `1`. Do we allow creators to raise it above 1 at all, and if
   so is the ceiling 100 (as the CHECK asserts) the right bound?
4. **Window boundary semantics.** Proposed inclusive `starts_at`, exclusive `ends_at`, both in UTC.
   Creators think in local dates — MKTC.4 must convert; confirm the storage contract now.
5. **Retention.** Do redemption rows follow the entitlement retention window (proposed) or the
   shorter analytics window? Needs the compliance owner from the S-series.
6. **Refund behaviour on the ledger.** Proposed: a refunded purchase `release`s the redemption so
   the seat returns to the pool. Product to confirm — the alternative (seat stays burned) punishes
   the creator for a refund they may not control. Implemented in MKTC.3, decided here.

## 19. References

- Existing files: `server/migrations/278_billing_stripe.sql`,
  `server/migrations/285_revenue_share.sql` (the `affiliate_codes` precedent),
  `server/migrations/368_course_marketplace.sql` (entitlement generalization + partial unique index
  precedent), `server/internal/repos/billing/entitlements.go`,
  `server/internal/currency/exponent.go`, `server/internal/telemetry/default.go`.
- Conventions: [docs/ARCHITECTURE_CONVENTIONS.md](../../ARCHITECTURE_CONVENTIONS.md) §1–§4
  (layering, file/package budgets), `server/migrations/README.md`.
- Related plans: [MKTC.2](MKTC.2-creator-coupon-management-api.md),
  [MKTC.3](MKTC.3-coupon-aware-checkout-and-redemption.md),
  [MKT4](MKT4-course-purchase-entitlement-flow.md),
  [15.13 tax compliance](../15-self-learner-specific/15.13-tax-compliance.md).
