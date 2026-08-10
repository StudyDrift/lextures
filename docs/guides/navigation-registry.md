# Navigation registry (UX.7)

Single source of truth for shell destinations. Sidebars, the command palette
synonym index, personalisation prefs, and the CI collision check all consume
`clients/web/src/lib/nav/`.

## Adding a destination

1. **Append a `NavDestination`** to the correct registry file:
   - Global shell → `registry-global.ts`
   - In-course → `registry-course.ts`
2. Required fields: `id`, `route`, `labelKey`, `label`, `icon`, `section`,
   `sectionV2`, `priority`, `audience`.
3. Optional: `permission`, `featureFlag`, `courseFeature`, `synonyms`,
   `primaryV2`, `essential`, `end`, `utility`.
4. **Add the Lucide icon name** to `icons.tsx` (import + `ICONS` map).
5. **Run** `npm run nav:check` in `clients/web/` — must pass (no duplicate icons
   or near-duplicate labels within a scope).
6. **i18n** — add `labelKey` under `common` for `en` / `es` / `fr` / `ar`
   (parity required by `i18n:check`).
7. Do **not** hand-edit `side-nav-*-links.tsx` lists for global/course scopes;
   they render from the registry.

### Priority ranks

- Lower number = higher in the section.
- **Never sort alphabetically** while a section has fewer than 20 items (R-12).
- Gradebook is intentionally priority `1` in `grades-insights` so it is first
  for instructors (AC-1).

### Feature flags

- A platform feature may gate a destination via `featureFlag` / `courseFeature`.
- Turning a flag off **removes** the destination (no disabled placeholders, FR-10).
- Adding a feature does **not** auto-add a nav link — you must declare section +
  priority in the registry (FR-20). `npm run nav:check` enforces structure.

## V2 taxonomy (`ffNavigationV2`)

Default **off**. When on:

- Task-based sections (Teach / Engage / Assess & analyse / Manage, etc.).
- Up to **7 primary** destinations ungrouped.
- Section **More** disclosure after the visible budget.
- Recent destinations (client-local).

Registry refactor (rendering current IA from data) ships unflagged.

## Personalisation

| Action | API |
|---|---|
| Load | `GET /api/v1/nav/preferences?scope=…` |
| Save | `PUT /api/v1/nav/preferences` body `{ scope, pinned, hidden, collapsed }` |
| Reset | `DELETE /api/v1/nav/preferences?scope=…` |

Scopes: `global` | `settings` | `admin` | `course:<code>` | `course-settings:<code>`.

Unknown destination ids are dropped on write. Missing row = defaults.

## Command palette synonyms

Declare `synonyms` on destinations (e.g. `marks` → Gradebook). Indexed via
`buildNavSynonymIndex` / `matchNavSynonyms`. Hidden items remain findable.

## Collision check

```bash
cd clients/web && npm run nav:check
```

Fails on duplicate icons, near-duplicate labels (normalised Levenshtein ≤2),
duplicate ids, missing section/priority, or icons missing from `icons.tsx`.

## Native clients

The web registry is the taxonomy authority. A shared JSON artefact for iOS/Android
is residual work; until then native should not invent parallel IA.

## Support: “A destination is missing for a user”

1. Confirm feature flag + permission for that destination.
2. Inspect prefs: `GET /api/v1/nav/preferences?scope=course:CODE` (or `global`).
3. If the id is in `hidden`, user can unhide via **Customise** or support can
   `DELETE` the scope to reset.
4. If `ffNavigationV2` is on, check **More** disclosure and pinned order.
