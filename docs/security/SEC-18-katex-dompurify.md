# SEC-18 — KaTeX HTML rendered via `dangerouslySetInnerHTML` without sanitization

- **Severity:** Medium
- **Status:** Confirmed present
- **Area:** Web client
- **Files:** [clients/web/src/components/math/katex-expression.tsx:70](../../clients/web/src/components/math/katex-expression.tsx), [clients/web/src/components/editor/equation-editor-dialog.tsx:237](../../clients/web/src/components/editor/equation-editor-dialog.tsx)

## Problem

KaTeX-rendered HTML is injected into the DOM via `dangerouslySetInnerHTML` with no sanitization layer:

```tsx
<span dangerouslySetInnerHTML={{ __html: html }} />
```

KaTeX is currently invoked with `trust: false, strict: 'ignore'`, which is safe *today*. But student- and teacher-authored LaTeX is rendered this way, so the safety depends entirely on KaTeX's own escaping. A future KaTeX CVE, or a one-line config change to `trust: true`, becomes stored XSS via authored math content with no second line of defense.

## Risk

Stored XSS reachable by any content author (LaTeX in questions, answers, discussion posts). Given SEC-02 (tokens in `localStorage`) and SEC-03 (no CSP), an XSS here is a token-theft vector. This is a defense-in-depth gap rather than a live bug.

## Fix

1. Add `dompurify` and sanitize KaTeX output before it reaches `dangerouslySetInnerHTML`:
   ```ts
   DOMPurify.sanitize(html, { USE_PROFILES: { mathMl: true, svg: true, html: true } })
   ```
   Centralize in the `KatexExpression` component so every call site is covered.
2. Add a regression test that renders LaTeX containing an injected `<img src=x onerror=alert(1)>` and asserts no event handler survives.
3. A strict CSP (SEC-03) provides the backstop if sanitization is ever bypassed.

## Verification

- Authored math with an injected event handler renders inert (handler stripped).
- The regression test fails if sanitization is removed.
