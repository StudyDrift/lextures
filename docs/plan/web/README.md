# Web Client — Market-Readiness Plans (W01–W06)

Implementation plans for the gaps that still stand between the **web client** (`clients/web/`)
and a market-ready release for **Self-Learners (SL)**, **K-12**, and **Higher-Ed (HE)**.

Every plan follows [`../_TEMPLATE.md`](../_TEMPLATE.md). This folder is the web-client analogue of
[`../cli/`](../cli/) (C01–C40) and [`../mobile/`](../mobile/) (M-series): those catalogue *feature
parity* for the CLI and mobile apps; this catalogues the *remaining polish and coverage* gaps in the
reference web client.

## How these gaps were found (2026-07-06)

The web client is **large and mature** — 199 page components, ~508 components, and 150 Playwright
specs. It is not missing whole feature areas; nearly every server capability in `docs/completed/`
(sections 01–21) has a corresponding web route in [`clients/web/src/app.tsx`](../../../clients/web/src/app.tsx).

The gaps below were surfaced by scanning the web source for concrete, verifiable signals rather than
by guessing at absent features:

- **Incompleteness markers** — `TODO`/`FIXME`, `placeholder`, `stub` (the client is clean: one live
  `TODO`, one shipped placeholder region).
- **Framework adoption ratios** — how much of the app actually consumes a "completed" framework
  (e.g. i18n `t()` calls, the shared toast/confirm components).
- **Native-primitive smells** — `window.alert`/`confirm`/`prompt`, raw UUID rendering to end users.
- **Cross-client parity** — features that shipped on mobile/CLI or in a completed plan but whose web
  surface is thin or bugged.

Two candidate gaps were investigated and **rejected** because verification showed the web client
already implements them: the age-appropriate / simplified UI mode (§13.11 — present via
`src/styles/ui-modes/{k2,elementary}.css` + `src/lib/reading-preferences.ts` + a before-paint script
in `index.html`) and offline resilience (workbox background-sync queues + IndexedDB in `src/sw.ts`).

## Severity legend

- **BLOCKER** — cannot sell to the listed market without it.
- **MAJOR** — RFP-losing gap / erodes trust with the listed market.
- **MINOR** — polish / consistency / parity.

## Plans

| ID | Plan | Severity | Markets | Effort | One-line gap |
|---|---|---|---|---|---|
| W01 | [App-wide internationalization & RTL coverage](../../completed/web/W01-i18n-application-coverage.md) | MAJOR | K12 · HE · SL | L | **Done** — first-wave namespaces (`dashboard`, `parent`, `billing`, `onboarding`), `ar` bundle + RTL, lint gate on externalized surfaces. |
| W02 | [K-12 parent/guardian portal completeness](W02-parent-guardian-portal-completeness.md) | MAJOR | K12 | M | One read-only page; grades render as raw item-ID prefixes; no attendance / behavior / report-card visibility. |
| W03 | [In-app dialogs & notifications (replace native alerts)](W03-in-app-dialogs-notifications.md) | MINOR | K12 · HE · SL | S | ~67 `window.alert/confirm/prompt` calls in the grading workbench and settings bypass the app's toast + confirm system. |
| W04 | [Report-card AI comment — attendance wiring](../../completed/web/W04-report-card-attendance-wiring.md) | MINOR (bug) | K12 | XS | **Done** — report-card list payload includes period-scoped absences; AI suggest omits attendance when unknown. |
| W05 | [Human-readable labels for entity IDs](../../completed/web/W05-human-readable-entity-labels.md) | MINOR | HE · K12 | S | **Done** — moderation, peer-review, and assignment staff surfaces use `<EntityLabel>` + enriched API labels. |
| W06 | [Feature-help onboarding walkthrough media](W06-feature-help-onboarding-media.md) | MINOR | SL · K12 · HE | S | The feature-help dock ships a visible "placeholder for a demo GIF" region to end users. |

## Sequencing at a glance

- **W01** shipped first-wave i18n coverage; continue namespace-by-namespace rollout for remaining pages.
- **W02** is the remaining concrete K-12 defect; it subsumes the parent half of **W05**.
- **W03** and **W06** are cross-cutting polish that any team can pick up independently.

## Related plan sets

- CLI parity — [`../cli/`](../cli/)
- Mobile parity — [`../mobile/`](../mobile/)
- Completed platform plans — [`../../completed/`](../../completed/)
