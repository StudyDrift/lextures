# Focus-anchor highlight — accessibility conformance (CC.8)

Transient highlight applied when a user follows a checklist (or help) deep link with `?focus=`.

## WCAG 2.1 mapping

| Criterion | How we meet it |
|---|---|
| **2.4.3 Focus Order** | Programmatic focus moves once, on a user-initiated navigation (checklist click or paste). Tab continues in document order; no focus trap. |
| **2.4.7 Focus Visible** | Accent outline (2 px) + 4 px offset ring meets ≥ 3:1 contrast against adjacent colours in light and dark themes. |
| **1.4.1 Use of Colour** | Ring is paired with a readable “Here” text chip (not colour alone). |
| **2.3.1 Three Flashes** | No pulsing, flashing, or looping animation under any setting. |
| **4.1.3 Status Messages** | Single polite `aria-live` announcement: “{label} — this is the setting from your checklist.” |
| **2.2.2 Pause, Stop, Hide** | Highlight is finite (4 s) or clears on interaction; nothing auto-updates after arrival. |

## Reduced motion

When `prefers-reduced-motion: reduce` (or `html.reduced-motion`):

- `scrollIntoView` uses `behavior: 'auto'`
- Ring appears/disappears without CSS transition

## Screen readers

- Arrival announcement fires once via the shared polite announcer.
- Region anchors use `tabindex="-1"` so the wrapper can receive programmatic focus without entering the tab order permanently.

## Auditing

Run axe while a highlight is active (both themes, LTR and RTL). The ring must remain ≥ 3:1 against its background; the chip must remain readable.
