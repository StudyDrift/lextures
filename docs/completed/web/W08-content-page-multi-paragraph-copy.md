# W08 — Content page multi-paragraph selection copy

> Bugfix. Source: content page reader clipboard handling in
> `clients/web/src/components/content-page/content-page-reader.tsx`.

## Implementation notes

- **Root cause:** After mouseup the reader clears the native selection, stores a `Range`, and
  re-renders the selection toolbar. That re-render remounted markdown DOM nodes. Chrome then
  boundary-adjusts the stored `Range`, so `range.toString()` returns only the first paragraph.
  ⌘/Ctrl+C preferred that live `Range` over the string captured at selection time.
- **Fix:** Capture plain text via `plainTextFromRange` into `pendingSelectionTextRef` while the
  Range is still live; copy / highlight / note use that captured string. Memoize
  `MarkdownArticleView` markdown components so toolbar state updates do not remount article DOM.
- **Tests:** Vitest `selection-plain-text.test.ts`; Playwright
  `e2e/tests/content-page-selection-copy.spec.ts`.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | W08 |
| **Section** | Web / Content page reader |
| **Severity** | MINOR — student copy/paste correctness |
| **Status** | DONE |
