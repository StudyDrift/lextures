# MB — Mobile Experience

Plans for the iOS and Android clients that are not scoped to another feature section. Every plan follows
[_TEMPLATE.md](../_TEMPLATE.md).

Mobile work that belongs to a feature section lives with that section instead — e.g. the mobile Content
Tools stories are `CT.M1`–`CT.M9` in [`../../completed/content_tools/`](../../completed/content_tools/).

## Plans

| ID | Plan | Status | Effort |
|---|---|---|---|
| MB.1 | [In-app browser: keep mobile links inside Lextures](../../completed/mobile/MB.1-in-app-browser.md) | **DONE** | M (2–4w) |

## MB.1 in one line

Today ~90 link call sites across the two apps eject the user to Safari or Chrome. MB.1 introduces one
`LinkOpener` with a single pure routing policy (native · in-app · system · external app · auth · blocked)
and a full-screen in-app browser — the X/Instagram pattern — with one-tap copy, drag-to-dismiss, explicit
carve-outs for SSO, checkout and native-app links, and a fix for the bearer-token prefix check in the
current `WebItemView` / `WebItemScreen`.
