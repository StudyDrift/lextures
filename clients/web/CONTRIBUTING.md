# Web client (`clients/web`)

## Mobile-first standards (plan 7.2)

- **Breakpoints:** Tailwind defaults — `sm` 640px, `md` 768px, `lg` 1024px. Prefer mobile-first utilities (`flex-col` then `md:flex-row`) over desktop-first overrides.
- **Touch targets:** Interactive controls (buttons, icon hits, quiz options) should be at least **44×44 CSS pixels** on narrow viewports (`min-h-11 min-w-11` or equivalent padding). Desktop may use denser `sm:` / `md:` sizes.
- **Horizontal overflow:** Wide grids and tables must live in a **`min-w-0 max-w-full overflow-x-auto`** (or `overflow-x-scroll`) container so the page shell does not gain horizontal scroll on phones.
- **Drag-and-drop:** Where `@dnd-kit` drag handles are hard to use on touch, provide **explicit reorder controls** at `max-md` (e.g. move up / move down) while keeping drag for pointer-first desktop layouts.
