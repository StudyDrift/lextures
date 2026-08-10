# UX.5 — WCAG 2.2 AA conformance gap report

**Date:** 2026-08-09  
**Scope:** Web client (`clients/web`) + related auth/shell behaviour  
**Claim target:** WCAG 2.2 Level AA (superset of 2.1 AA)

## Criteria inventory (new or changed in 2.2)

| SC | Level | Title | Status after UX.5 | Evidence |
|---|---|---|---|---|
| 2.4.11 | AA | Focus Not Obscured (Minimum) | **Addressed** | `--lx-sticky-offset`, `useStickyOffset`, scroll-margin on focusables; toast offset under chrome |
| 2.5.7 | AA | Dragging Movements | **Addressed (modules); inventory for rest** | `MoveToPositionMenu`, `moveModuleToIndex` / `moveChildToIndex`, KeyboardSensor; `drag-surfaces-inventory.json` + `a11y:drag-alt` |
| 2.5.8 | AA | Target Size (Minimum) | **Addressed by construction + ratchet** | UX.2 `min-h-6` tokens; iconGhost ≥24px; `a11y:target-size` |
| 3.2.6 | A | Consistent Help | **Addressed** | `HelpWidgetMenu` in top bar on authenticated routes (`data-lx-help-entry`) |
| 3.3.7 | A | Redundant Entry | **Addressed** | Multi-step inventory; magic-link email prefill; onboarding state retention |
| 3.3.8 | AA | Accessible Authentication (Minimum) | **Addressed** | autocomplete attrs; paste allowed; passkey primary on MFA; magic link; OTP visible |
| 2.4.12 | AAA | Focus Not Obscured (Enhanced) | Out of scope | Aspirational |
| 2.4.13 | AAA | Focus Appearance | Out of scope | Aspirational |
| 3.3.9 | AAA | Accessible Authentication (Enhanced) | Out of scope | Aspirational |

## Remaining residual risk

1. **Non-module drag surfaces** still rely on keyboard + inventory allowlist for
   click-to-move; adopt `MoveToPositionMenu` / `useClickToMove` on catalog kanban,
   boards, gradebook layout, portfolio, live-quiz kit as traffic prioritises.
2. **Static target-size harness** is a source ratchet, not a full Playwright
   measurement of the top 40 routes — commission e2e visual measurement before
   external audit (AC-1 full).
3. **Third-party audit (AC-12)** is not executed in this PR; required before
   contractual WCAG 2.2 claims in EU RFPs.
4. **Passwordless passkey on the bare login page** still routes through MFA
   pending token (existing security model); passkey is first-class on MFA.

## CI observability

| Metric | Command |
|---|---|
| `target_size_violations` | `npm run a11y:target-size` |
| `drag_surfaces_without_alternative` | `npm run a11y:drag-alt` |
| Combined | `npm run a11y:wcag22` |

## Sign-off checklist (engineering)

- [x] Sticky offset token live in shell
- [x] Reorderable primitives + modules Move to…
- [x] Auth autocomplete / paste / passkey primacy
- [x] Help entry consistent in top bar
- [x] Multi-step inventory documented
- [x] VPAT notes for 2.2 criteria
- [ ] External WCAG 2.2 AA audit commissioned
