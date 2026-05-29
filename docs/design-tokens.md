# Design Tokens — Color Contrast Contract

> Part of plan 12.3 — Color-Contrast Compliance (WCAG 2.1 AA).
> This document is the single source of approved foreground/background token pairs.
> The machine-readable version lives in `clients/web/contrast-config.json` and is
> validated by CI (`npm run contrast:check`).

## Governing standards

| Standard | Requirement |
|---|---|
| WCAG 2.1 SC 1.4.3 | Normal text ≥ 4.5:1, large text (≥ 18pt or ≥ 14pt bold) ≥ 3:1 |
| WCAG 2.1 SC 1.4.11 | Non-text UI components ≥ 3:1 against adjacent background |
| WCAG 2.1 SC 1.4.1 | Color must not be the sole differentiator for status states |
| Section 508 §1194.21(j) | Operability without color perception |

---

## Tailwind token hex values

These are the canonical hex values for every token used in Lextures UI. Updated when Tailwind's palette changes.

| Token | Hex | Swatch |
|---|---|---|
| `white` | `#ffffff` | ████ |
| `slate-50` | `#f8fafc` | ████ |
| `slate-100` | `#f1f5f9` | ████ |
| `slate-200` | `#e2e8f0` | ████ |
| `slate-400` | `#94a3b8` | ████ |
| `slate-500` | `#64748b` | ████ |
| `slate-600` | `#475569` | ████ |
| `slate-700` | `#334155` | ████ |
| `slate-800` | `#1e293b` | ████ |
| `slate-900` | `#0f172a` | ████ |
| `neutral-100` | `#f5f5f5` | ████ |
| `neutral-200` | `#e5e5e5` | ████ |
| `neutral-300` | `#d4d4d4` | ████ |
| `neutral-400` | `#a3a3a3` | ████ |
| `neutral-500` | `#737373` | ████ |
| `neutral-700` | `#404040` | ████ |
| `neutral-800` | `#262626` | ████ |
| `neutral-900` | `#171717` | ████ |
| `neutral-950` | `#0a0a0a` | ████ |
| `indigo-400` | `#818cf8` | ████ |
| `indigo-500` | `#6366f1` | ████ |
| `indigo-600` | `#4f46e5` | ████ |
| `indigo-700` | `#4338ca` | ████ |
| `red-600` | `#dc2626` | ████ |
| `amber-500` | `#f59e0b` | ████ |
| `green-600` | `#16a34a` | ████ (⚠ only 3.30:1 on white — fails AA for text) |
| `green-700` | `#15803d` | ████ |
| `emerald-500` | `#10b981` | ████ |

---

## Approved light-mode pairs

All pairs achieve ≥ 4.5:1 (WCAG AA normal text).

| Foreground | Background | Ratio | Usage |
|---|---|---|---|
| `slate-900` | `white` | 17.85:1 | Body text on card surfaces |
| `slate-800` | `white` | 11.92:1 | Headings on card surfaces |
| `slate-700` | `white` | 10.33:1 | Subheadings on card surfaces |
| `slate-600` | `white` | 7.25:1 | Muted / secondary text on card surfaces |
| `slate-500` | `white` | 4.76:1 | Placeholder and tertiary text; TipTap editor placeholder |
| `slate-900` | `slate-50` | 17.05:1 | Body text on page background |
| `slate-600` | `slate-50` | 7.25:1 | Muted text on page background |
| `slate-500` | `slate-50` | 4.55:1 | Placeholder text on page background |
| `white` | `indigo-600` | 6.28:1 | Primary button / CTA label |
| `white` | `indigo-700` | 8.59:1 | Primary button hover / active |
| `red-600` | `white` | 4.83:1 | Error / destructive text |
| `red-600` | `slate-50` | 4.64:1 | Error text on page background |
| `green-700` | `white` | 5.02:1 | Success text (use green-700, not green-600 — see known exceptions) |
| `slate-900` | `slate-100` | 15.33:1 | Text on table row alternates and hover states |

---

## Approved dark-mode pairs

All pairs achieve ≥ 4.5:1 (WCAG AA normal text).

| Foreground | Background | Ratio | Usage |
|---|---|---|---|
| `neutral-100` | `neutral-950` | 18.16:1 | Body text on page background |
| `neutral-200` | `neutral-900` | 14.23:1 | Headings on card surfaces |
| `neutral-300` | `neutral-900` | 12.09:1 | Subheadings on card surfaces |
| `neutral-400` | `neutral-900` | 7.10:1 | Muted / secondary text on card surfaces |
| `neutral-100` | `neutral-800` | 13.86:1 | Text on elevated surfaces |
| `neutral-400` | `neutral-800` | 5.99:1 | Muted text on elevated surfaces |
| `neutral-400` | `neutral-950` | 7.85:1 | Placeholder text; TipTap editor placeholder |
| `indigo-400` | `neutral-950` | 6.64:1 | Link / accent color on page background |
| `indigo-400` | `neutral-900` | 6.25:1 | Link / accent color on card surfaces |

---

## Status colors — non-color differentiators (SC 1.4.1)

Color alone must never be used to convey state. Every status indicator must include
at least one non-color signal:

| Status | Color token | Required companion |
|---|---|---|
| Error / destructive | `red-600` | Error icon (XCircle) + text label |
| Warning | `amber-500` | Warning icon (AlertTriangle) + text label |
| Success | `green-600` | Success icon (CheckCircle) + text label |
| Info | `indigo-500` | Info icon (Info) + text label |

---

## Adding new token pairs

1. Verify the contrast ratio using `scripts/check-contrast.mjs` locally:
   ```sh
   cd clients/web
   npm run contrast:check
   ```
2. Add the pair to `clients/web/contrast-config.json` under `pairs.light` or `pairs.dark`.
3. Use the token in components. Do **not** introduce raw hex values in component code.
4. CI will reject PRs that introduce any pair below threshold.

---

## Known exceptions

| Pair | Ratio | Reason |
|---|---|---|
| `slate-200` border on `white` | 1.23:1 | Decorative section divider — not required to identify a UI component per SC 1.4.11. Must be accompanied by other visual separators (spacing, background color). |
| `neutral-700` border on `neutral-900` | 1.73:1 | Same as above for dark-mode decorative dividers. |
| `green-600` text on `white` | 3.30:1 | **Do not use** `green-600` for body/label text on white — it fails AA. Use `green-700` (#15803d, 5.02:1) instead. |

---

## References

- WCAG 2.1 SC 1.4.3: <https://www.w3.org/TR/WCAG21/#contrast-minimum>
- WCAG 2.1 SC 1.4.11: <https://www.w3.org/TR/WCAG21/#non-text-contrast>
- WCAG 2.1 SC 1.4.1: <https://www.w3.org/TR/WCAG21/#use-of-color>
- axe-core color-contrast rule: <https://dequeuniversity.com/rules/axe/4.7/color-contrast>
- Colour Contrast Analyser (Paciello Group): <https://www.tpgi.com/color-contrast-checker/>
