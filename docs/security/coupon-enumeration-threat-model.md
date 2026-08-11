# Coupon enumeration & discount abuse — threat model (MKTC.7)

**Status:** Ready for security owner sign-off before `ffCourseCoupons` default ON.  
**Scope:** First-party course coupon apply (preview), creation ceiling, attempt audit trail.  
**Related:** [runbook](../runbooks/coupons.md), [plan MKTC.7](../plan/marketplace/MKTC.7-abuse-prevention-analytics-and-rollout.md).

## Assets

| Asset | Sensitivity |
|---|---|
| Valid coupon codes (short secrets) | High — money-moving |
| `billing.coupon_attempts` (hashed codes, IP prefix, user) | Medium — abuse signal, limited PII |
| Platform discount ceiling | High — liability bound |
| Redemptions CSV (learner name/email) | Medium — roster-equivalent bulk egress |

## Adversaries & goals

1. **Code sprayer** — brute-force short codes on a popular course to discover a working discount.
2. **Leaked unlimited code** — share a high-cap or uncapped code publicly; high velocity redemptions.
3. **Creator over-discount** — 100% off mistakes or intentional giveaways that blow revenue share.
4. **Insider export abuse** — staff CSV-export learner emails for spam (already have roster access).

## Attack surface

- `POST /api/v1/marketplace/courses/{slug}/coupon/preview` (authenticated session required).
- Creator create: `POST /api/v1/courses/{course_code}/coupons`.
- CSV: `GET .../coupons/{coupon_id}/redemptions.csv`.

Unauthenticated callers cannot apply codes (session required). Caps (max redemptions) live in Postgres and are not bypassed by rate-limit fail-open.

## Mitigations (mapped)

| Threat | Control | FR |
|---|---|---|
| Online guessing | Layered limits: 15/min & 60/h per (user,course); 100/h per user; 200/h per IP | FR-1 |
| Sustained failures | 10 consecutive fails → 15 min cool-down (`reason: cooldown`) | FR-2 |
| Mining attempt logs | Salted hash of code for `not_found`; raw never stored; IP /24 or /48 only; 30-day retention | FR-3 |
| Low-entropy codes | Server warn `low_entropy` for &lt;6 chars or dictionary; client generate ≥8 from 32-symbol alphabet | FR-4 |
| Unbounded % off | Platform `couponMaxPercentOff` (default 100); create above ceiling → 422 | FR-5 |
| Leaked code velocity | Metric + alert on redemptions/10m; kill switch = archive/disable | FR-6 |
| CSV bulk egress | Same auth as list; 5 exports/h/user; admin audit event | FR-9 |
| Limiter outage | Fail **open** on in-memory/Redis blip; seat caps still enforced in DB | AC-13 |

## Residual risk

- Short dictionary codes (e.g. `FREE`) remain guessable if a creator ignores the warning and leaves high caps; operators should set caps and the ceiling.
- Per-user limits do not stop a botnet of many accounts; velocity alerts + seat caps bound loss.
- Cool-down is in-process memory today (same shape as checkout limiter); multi-replica deployments should prefer Redis-backed buckets when `REDIS_URL` is set (extension point — current implementation is process-local fail-open).

## Acceptance for flag flip

Security owner reviews this note (AC-14) before GA. The default for `ffCourseCoupons` is **ON** in
`repos/platformconfig/features.go` as of MKTC.7 (explicit DB `false` still wins). Rollback: set the
platform setting to false (instant) or revert the default-flip commit.

## Sign-off

| Role | Name | Date | Result |
|---|---|---|---|
| Security owner | _pending_ (review before production GA announcement) | | |
| Commerce / growth | implemented with MKTC.7 | 2026-08-10 | Mitigations shipped |
