# W09 — Section heading Enter focuses content

> UX polish. Source: content / syllabus block editor section title field in
> `clients/web/src/components/syllabus/syllabus-block-editor.tsx`.

## Implementation notes

- **Problem:** In the content editor, pressing Enter while editing a section title did nothing
  useful; authors expect the caret to move into the section body so they can keep writing.
- **Fix:** On Enter (without Shift, and not during IME composition) in the canvas section heading
  input, prevent default and focus the TipTap body editor for that section (`focus('end')`).
- **Tests:** Vitest `section-heading-enter.test.ts` +
  `syllabus-block-editor-heading-enter.test.tsx`; Playwright
  `e2e/tests/section-heading-enter-focus.spec.ts`.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | W09 |
| **Section** | Web / Content editor |
| **Severity** | MINOR — authoring keyboard flow |
| **Status** | DONE |
