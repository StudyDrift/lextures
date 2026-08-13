# Translating a help article

Translations are separate articles that share a translation group. English stays at `/docs/{category}/{slug}`; other locales publish at `/{locale}/docs/{category}/{slug}` and may use a localized slug. Publishing English never publishes a translation.

## Before you start

1. `ff_marketing_content` is on.
2. Editorial settings have **locales enabled** (`PATCH /api/v1/admin/marketing/settings` with `{ "localesEnabled": true }`).
3. The target locale row exists on `marketing.content_locales` and is **enabled** (Spanish, French, and Arabic are seeded disabled).
4. You have `global:app:marketing-content:author` (and publish permission when you are ready to ship).

## Workflow

1. Open the English source in Marketing Content.
2. Use **Translations → Add {language} translation**. The new row starts as a draft with the source structure copied and an empty body.
3. Write the translation in the editor. The read-only source pane shows the English revision you last synced to.
4. Localize the slug if it helps local search. The metadata panel previews the locale-prefixed URL.
5. Submit, review, and publish the translation on its own lifecycle.
6. After the source changes, the translation shows **Stale** in the list, editor, and editorial health view. Update the translation, then **Mark synced**.

## Staleness

A translation is stale when the source `content_updated_at` is newer than `source_synced_at`. Stale help is worse than English fallback. If the product change is large and the translation cannot be updated promptly, unpublish it rather than leave it live.

## In-app help

The help widget requests articles in the viewer's app locale (`en`, `es`, `fr`, `ar`) and falls back to English, labelled as such, when no translation exists.
