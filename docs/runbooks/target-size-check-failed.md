# Runbook: Target-size check failed

**Script:** `cd clients/web && npm run a11y:target-size`  
**Metric:** `target_size_violations` (WCAG 2.2 SC 2.5.8 / UX.5 FR-3)

## What failed

The static ratchet found more interactive hosts with fixed sizes under 24×24 CSS
px than the baseline allows (without a `min-h-6` / `min-w-6` or larger companion).

## Fix options (prefer in order)

1. **Use the design system** — `Button`, `IconButton` with `size="sm"|"md"|"lg"`.
   Every size already meets 24×24.
2. **Enlarge** — set `min-h-6 min-w-6` (or larger) on the control.
3. **Spacing exception** — keep a compact visual hit area only if neighbouring
   targets are ≥24px apart *and* document the file in
   `clients/web/target-size-exceptions.json` with `exception: "spacing"`.
4. **Inline exception** — for text links inside a sentence (`exception: "inline"`).

Do **not** raise the baseline to “make CI green” without accessibility-lead
sign-off.

## After a deliberate cleanup batch

```bash
cd clients/web
npm run a11y:target-size -- --write-baseline   # only if count decreased
```

## Related

- `docs/guides/accessibility-patterns.md` (Target size section)
- `docs/completed/ui-ux/UX.5-wcag-2.2-aa-conformance-uplift.md`
