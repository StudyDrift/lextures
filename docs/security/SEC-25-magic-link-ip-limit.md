# SEC-25 — Magic-link rate limit is per-user only, no IP throttle

- **Severity:** Low
- **Status:** Confirmed present
- **Area:** Server / auth
- **File:** [server/internal/service/authservice/magic_link.go](../../server/internal/service/authservice/magic_link.go) (`magicLinkRateMax`)

## Problem

The magic-link request flow rate-limits at `3 / 5 min` **per known user**. There is no IP-keyed limit, and requests for unknown emails return early before the per-user counter is consulted, so enumeration attempts against non-existent emails are not counted at all.

## Risk

An attacker can spray magic-link requests across many email addresses from one IP without tripping any limit, enabling email enumeration and using the platform as an email-sending amplifier (deliverability/abuse risk). Lower severity because it doesn't directly grant access, but it complements SEC-04 and SEC-23.

## Fix

Add an IP-keyed bucket (e.g. 10 / 15 min) evaluated **before** the email lookup, so requests for unknown emails are throttled too. Keep the existing per-user limit as a second dimension. Read the real client IP from the trusted forwarded header (consistent with SEC-04).

## Verification

- 11 magic-link requests from one IP across different emails in 15 min return `429`.
- Requests for unknown emails count toward the IP bucket.
