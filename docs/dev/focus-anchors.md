# Focus anchors (CC.8)

Deep-link and highlight targeting so checklist items (and help-centre links) land on the **exact control**.

## Two-registry rule

| Surface | Registry | Example IDs |
|---|---|---|
| Assignment / quiz **editor controls** | PS.1 `clients/web/src/lib/settings-registry.ts` | `assignment.scheduling.due-date` |
| Everything else | CC.8 `clients/web/src/lib/focus-anchors.ts` (+ `focus-anchors-registry.ts` data) | `course.general.dates`, `modules.item` |
| Editor **sections** (accordion open) | CC.8 focus-anchors | `assignment.scheduling`, `quiz.outcomes` |

Never mint a parallel ID for a control PS.1 already names. Full three-segment PS.1 IDs resolve through `resolveSettingId` automatically.

## Target grammar

```
{ route, anchor?, entityKey? }
```

- `route` — path template (`/courses/{courseCode}/settings/general`)
- `anchor` — focus-anchor ID matching `^[a-z][a-z0-9]*(\.[a-z0-9-]+){1,3}$`
- `entityKey` — for entity-kind anchors (and `{itemId}` substitution)

## URL contract

```
/courses/BIO101/settings/general?focus=course.general.dates
/courses/BIO101/modules?focus=modules.item&focusEntity=<uuid>
```

On arrival the runtime validates the id, opens any declared container, scrolls, focuses, highlights, announces, then **strips** `focus` / `focusEntity` via `history.replace` so refresh does not re-fire.

Unknown anchors → plain navigation (dev-only warning). Malicious values never reach an unescaped DOM selector.

## Adding an anchor

1. Append a `FocusAnchor` to `FOCUS_ANCHORS` in `focus-anchors-registry.ts` (`id`, `route`, `labelKey`, `label`, `kind`, optional `container`).
2. Attach it in the page with either:
   - `data-focus-anchor="your.id"` on the control/region, or
   - `<Anchor id="your.id">…</Anchor>` / `useAnchorRef('your.id')`.
3. Entity rows also set `data-focus-entity={id}`.
4. For virtualised lists, `registerEntityRevealer(routeKey, reveal)` so the row is scrolled into the DOM before focus.
5. If a checklist rule points at it, ensure the server `NavTarget.Anchor` matches (and re-generate the fixture — below).

## Integrity fixture

Server catalog targets are exported to:

`server/internal/service/coursechecklist/testdata/catalog_targets.json`

Regenerate:

```bash
cd server && go test ./internal/service/coursechecklist/ -run TestCatalogTargetsFixture -update
```

The web test `focus-anchors.test.ts` asserts every non-empty catalog `anchor` resolves via `resolveFocusAnchor` (registry, alias, or PS.1).

## Native table (CC.9)

`clients/packages/checklist-targets.json` maps each anchor id to an iOS / Android destination or `"web-only"`.
Bundled copies ship in:

- iOS: `clients/ios/Lextures/Resources/checklist-targets.json`
- Android: `clients/android/app/src/main/assets/checklist-targets.json`

Mobile pure logic (`CourseChecklistLogic` on both platforms) resolves targets against this table and falls
back to the in-app browser (`LinkOpener`) for `web-only` / unmapped anchors. **Do not hand-edit the
bundled copies** — update the package source and re-copy when the table changes.

## Highlight behaviour

- Outline + offset ring for 4 s (or until keypress / click / scroll > 200 px)
- “Here” chip (not colour-alone)
- No pulsing; reduced-motion → instant scroll, no fade transition
- Polite live-region announcement once: “{label} — this is the setting from your checklist.”

## Contributor checklist

- If you delete or rename a control, check `focus-anchors.ts` (and PS.1 if it is an editor setting).
- Prefer aliases over deleting IDs (`FOCUS_ANCHOR_ALIASES` / `RETIRED_FOCUS_ANCHOR_IDS`).
