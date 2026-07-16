# Contributing to the web client

## Route code splitting

All LMS pages are lazy-loaded via `React.lazy` in `src/lazy-pages.ts` and rendered inside a top-level `Suspense` boundary in `src/app.tsx`. When adding a new route:

1. Add the page component under `src/pages/`.
2. Export a lazy wrapper in `src/lazy-pages.ts`.
3. Reference the lazy export from `src/app.tsx` through the `Pages` namespace.

Do not import page modules synchronously in `app.tsx`; that pulls every route into the initial bundle.

## Bundle budgets (LH.2)

Production builds run `npm run bundle:check` after `vite build`. Budgets:

| Check | Limit |
| --- | --- |
| Initial entry chunk (`index-*.js` gzip) | ≤ 257 KiB + 2 KiB CI slack |
| Dashboard lazy chunk regression | ≤ +10 KB vs `scripts/bundle-baseline.json` |

To refresh the dashboard baseline after an intentional size increase:

```bash
BUNDLE_UPDATE_BASELINE=1 npm run bundle:check
```

Commit the updated `scripts/bundle-baseline.json` with the PR that explains the change.

## Lighthouse

See root `AGENTS.md` for the dashboard dark-mode Lighthouse harness (`npm run lighthouse:dashboard:dark`).
