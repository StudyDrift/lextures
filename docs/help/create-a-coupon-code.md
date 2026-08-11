# Create a coupon code

> Instructor help for the course **Coupon codes** panel (plan MKTC.4).

## When to use this

Run a launch promotion, give a co-op a fixed discount, or cap the first N
purchases of a paid marketplace course without changing the list price.

## Prerequisites

- The course is **published** and has a **non-zero** marketplace price (save the
  fee in **Marketplace** settings first).
- Platform flag **Course coupon codes** (`ffCourseCoupons`) is on
  (Settings → Global platform). The course marketplace flag must also be on.
- You have edit rights on the course (owner, co-teacher, or designer).

## Steps

1. Open **Course settings → Marketplace**.
2. Scroll to **Coupon codes** (below the fee form). If you only see the fee
   form, the coupons flag is off or you lack edit permission.
3. Click **New coupon**.
4. Enter a code (letters, digits, `_` / `-`; 4–32 characters) or click
   **Generate**. The field upper-cases as you type.
5. Choose **Percent** or **Fixed amount**, then set the value. Watch the live
   preview: “Learners pay … (was …)”. If the result is free, confirm before
   creating.
6. Optionally set start/end (your local timezone), a total redemption cap, a
   per-learner limit (default 1), and an internal note.
7. Click **Create coupon**. The new row appears at the top.

## Share the code

Each row has a **Copy share link** control (not only in the ⋯ menu). It copies
the in-app marketplace URL with `?coupon=CODE` already attached. For public
courses, open the split menu and choose **Copy public site link** for the
marketing-site URL with the same parameter.

Do **not** hand-build the URL — always use the button so the path and parameter
match the server.

If the browser blocks clipboard access, a dialog shows the URL in a selectable
field so you can copy it manually.

## Pause vs archive

| Action | Effect |
|---|---|
| **Pause** | Status becomes *Paused* (`disabled`). New redemptions stop immediately; history stays. Resume later from the same menu. |
| **Archive** | Soft-delete. Hidden from the default table (use **Show archived**). Same code can be re-created later as a new row. Past redemptions are kept. |

You cannot change a code’s discount after creation. Archive and create a new
code if the amount was wrong.

## Performance

Each row shows a compact **Performance** summary: claims, total discount given, and
net charged (refunds excluded). Open **View redemptions** for the full breakdown and
**Export CSV** (learner name, email, status, amounts, ISO dates) for co-op
reconciliation. Exports are rate-limited (5 per hour).

## Mobile share links

Your share links work on phones. On Android, learners complete the discounted purchase
in the app. On iOS, **partial** discounts finish on the web (App Store fixed price points);
100% off codes still grant access in the app. Set expectations in your own marketing copy.

## Who can manage coupons

Only roles with course `item:create` (owner / teacher / designer). Teaching
assistants and students cannot list or create codes.

## What a coupon costs you

See [what-a-coupon-costs-you.md](what-a-coupon-costs-you.md) for how discounts affect
revenue share.

## Related

- Runbook: [docs/runbooks/coupons.md](../runbooks/coupons.md)
- API changelog: [docs/api-changelog-course-coupons.md](../api-changelog-course-coupons.md)
- Learner help: [using-a-coupon-code.md](using-a-coupon-code.md)
