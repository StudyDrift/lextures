# WCAG 2.2 SC 3.3.7 — Multi-step flow inventory (UX.5 FR-14)

Information already supplied in a multi-step process must be auto-populated or
selectable, not retyped (except password confirmation / security re-verification).

| Flow | Location | Steps | Redundant-entry status | Notes |
|---|---|---|---|---|
| Password + magic link sign-in | `pages/login.tsx`, `components/auth/magic-link-request-form.tsx` | email → optional magic link | **OK** | Magic link email prefills from password form (`defaultEmail`) |
| MFA challenge / setup | `pages/mfa-login.tsx` | password (prior) → passkey / TOTP / backup | **OK** | No re-entry of email; MFA pending token carries session |
| Signup | `pages/signup.tsx` | single page | **OK** | Not multi-step |
| Onboarding | `pages/onboarding/onboarding-page.tsx` | 7 steps, client state | **OK** | Choices held in React state; back navigation restores values |
| Course enrollment | LMS enrollment dialogs / self-enroll | varies | **OK** | Uses viewer profile; does not re-ask identity fields |
| Parent activation | `activate-parent` routes | token + optional profile | **OK** | Token carries identity |
| Conference booking | conference pages | slot → confirm | **OK** | Prior selections retained in wizard state |
| Accessibility intake | settings / accommodations | multi-field form | **OK** | Single form or draft state; no forced retype across steps |
| Checkout / billing | billing flows | plan → payment | **Partial** | Payment details may re-request for PCI; identity from account |

## Essential re-entry (allowed)

- Password confirmation on password change
- MFA OTP / backup codes (security re-verification)
- Payment card PAN (PCI), when not using a stored payment method

## Follow-ups

- If enrollment ever splits into server-side multi-step drafts without GETting
  prior values, add draft prefill on step endpoints (plan open question Q3).
