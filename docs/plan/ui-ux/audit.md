# Lextures Web — UI / UX Gap Analysis

> Measured audit of `clients/web` against the evidence base in [research.md](research.md).
> Audit date **2026-08-05**, at commit `aa0d523a`.
> Findings are numbered `G-n` and referenced by the `UX.*` plans in this folder.

---

## 0. Method

- **Static measurement.** Every quantitative claim below was produced by grep/find
  over `clients/web/src`, excluding `__tests__` and `*.test.*`. The commands are
  reproducible; counts are stated so they can be re-run as a ratchet.
- **Structural reading.** The app shell, navigation, dashboard, top bar, settings,
  and the design-system primitives were read in full.
- **Not covered.** This audit is code- and structure-based. It does **not**
  substitute for moderated usability testing or a screen-reader audit — both are
  scoped as work inside `UX.4`, `UX.7` and `UX.18`.

### Scale of the surface under audit

| Measure | Value |
|---|---|
| TS/TSX files | 1,572 |
| TSX files (non-test) | **795** |
| Lines of TS/TSX (non-test) | ~283,000 |
| Registered routes | **200** |
| Page components | ~310 |
| Platform feature definitions (nav-shaping flags) | **114** |
| Locales shipped | 4 (`en`, `es`, `fr`, `ar` — `ar` is RTL) |
| Entry bundle (gzip, baseline) | **245,104 B** |

---

## 1. Executive summary

**Lextures has built the right primitives and then not adopted them.**

This is the single organising finding of the audit. The repository contains a
button component, an overlay surface, a skeleton set, an empty-state component, a
toast helper with undo, a focus-trap utility, a confirm-dialog hook, a motion
token system, an i18n pipeline with four synchronised locales, a command palette,
a contrast CI check, and a written visual design document. Almost none of it is
wired into the 795 components that make up the product.

| Primitive | Exists | Files using it | Files that should |
|---|---|---|---|
| `components/ui/button.tsx` | ✅ | **2** | ~600 |
| `components/ui/overlay-surface.tsx` | ✅ | **0** *(dead code)* | ~129 |
| `components/ui/lms-content-skeletons.tsx` | ✅ | **5** | ~200 |
| `components/ui/empty-state.tsx` | ✅ | **15** | ~150 |
| `lib/lms-toast.ts` (incl. `toastWithUndo`) | ✅ | **1** | ~300 |
| `lib/a11y/focus-trap.ts` | ✅ | **3** | **129** |
| `components/use-confirm.tsx` | ✅ | 41 | 41 ✅ |
| Semantic colour tokens | ❌ | — | all |

The consequence is that the product's *quality ceiling* is not set by the design
system — it is set by whatever each of 795 files happened to hand-roll. Two
engineers implementing the same button on the same day produce two different
buttons, and both ship.

**Second-order finding.** The one domain where tokens *were* enforced —
**motion**, via the completed AN.1–AN.7 plans — is markedly consistent: 10
duration tokens, 3 easing curves, a documented bubble spring, reduced-motion
handling throughout, and a CI check (`check-interface-polish.mjs`) that fails the
build on bare `transition` and layout-animating properties. This is proof the
organisation *can* do this. It has simply only done it once, for the least
load-bearing domain.

### Severity roll-up

| # | Finding | Severity |
|---|---|---|
| G-1 | No semantic token layer; 33,331 raw colour literals in 698/795 files | **BLOCKER** |
| G-2 | Component library exists but is unadopted (Button: 2 files; 2,016 raw `<button>`) | **BLOCKER** |
| G-3 | 32 of 37 ARIA menus and **22 of 22** tablists lack keyboard contracts | **BLOCKER** (legal) |
| G-4 | 126 modals declare `aria-modal` without focus trapping | **BLOCKER** (legal) |
| G-5 | Navigation sprawl: up to 40 in-course links, alphabetically ordered | **MAJOR** |
| G-6 | Dashboard is an unprioritised vertical banner farm (~18 sections) | **MAJOR** |
| G-7 | Body type is 14px/12px; `text-base` used 168× vs `text-xs` 2,546× | **MAJOR** |
| G-8 | 929 inputs, 10 `aria-invalid` — errors not programmatically associated | **MAJOR** (legal) |
| G-9 | Three competing loading idioms; 4 error boundaries for 200 routes | **MAJOR** |
| G-10 | i18n at 34% of components despite shipping an RTL locale | **MAJOR** |
| G-11 | Responsive coverage 32%; tables and touch targets unverified | **MAJOR** |
| G-12 | `/settings/*` vs `/admin/*` IA split-brain across 33 destinations | **MAJOR** |
| G-13 | Feedback fragmented: 364 hand-rolled error banners, 1 toast import | **MAJOR** |
| G-14 | 5 competing radii, 62 arbitrary hex, 594 arbitrary px/rem | **MINOR** |
| G-15 | God components block surface work (3,403-line page; 100 `useState`) | **MAJOR** |
| G-16 | `docs/design.md` is stale prose, contradicted by the code | **MINOR** |
| G-17 | Entry bundle 245 KB gzip; no CWV budget per route class | **MAJOR** |
| G-18 | No motivational/progress model; gamification present but unframed | **MINOR** |

---

## 2. Design system and visual language

### G-1 — There is no semantic colour token layer *(BLOCKER)*

**Measured.**

| Metric | Count |
|---|---|
| Raw Tailwind palette literals in `.tsx` | **33,331** |
| Files containing at least one | **698 / 795 (88%)** |
| Distinct arbitrary hex values (`bg-[#...]`) | 28 (62 occurrences) |
| Arbitrary sizes (`[14px]`, `[10rem]`, …) | 594 |

`src/index.css` defines **motion tokens only** — 10 durations, 3 easings,
stagger, enter-translate, enter-scale, press-scale — plus `--shadow-card` and a
handful of reading-preference variables. There are **zero** semantic colour,
spacing, or typography tokens.

**Two competing neutral ramps.** The codebase uses `slate-*` for light mode and
`neutral-*` for dark mode, per-element, by hand:

```
slate-200  2160     neutral-400  2009
slate-500  1949     neutral-800  1610
slate-600  1448     neutral-100  1602
slate-900  1408     neutral-700  1474
```

Every surface therefore carries a hand-written light/dark pair. A representative
line from `pages/lms/dashboard.tsx:706`:

```
border-slate-200 bg-white text-slate-800 shadow-sm hover:border-slate-300
hover:bg-slate-50 dark:border-neutral-700 dark:bg-neutral-900
dark:text-neutral-100 dark:hover:border-neutral-600 dark:hover:bg-neutral-800
```

That is ten colour decisions, repeated verbatim four times in the same file, for
one visual concept ("secondary button"). There is no single place to change it.

**Colour carries no meaning.** The dashboard assigns accent colours by section
with no semantic rule: Continuing Education = teal, Achievements = violet,
Credentials = emerald, Review practice = amber, What's Next = violet gradient,
errors = rose, warnings = amber. A learner cannot form a colour→meaning mapping
because none exists. Per **R-1/R-3**, this is pure extraneous cognitive load.

**Consequences.** Rebranding is impossible. Org white-labelling (`org-branding`
page exists) cannot reach 33,331 literals. High-contrast mode
(`styles/high-contrast.css`) must fight every one of them. The contrast CI check
validates a **hand-maintained allowlist of pairs**, not the pairs actually
rendered — it cannot see 33,331 ad-hoc combinations.

**Against R-25/R-26**: the industry standard is a three-layer token architecture
where feature code references semantic tokens only. Lextures has layer 1 in
feature code and no layer 2.

---

### G-2 — The component library is unadopted *(BLOCKER)*

**Measured.**

| Metric | Count |
|---|---|
| Files in `components/ui/` | 19 |
| Raw `<button>` elements | **2,016** |
| Files importing `ui/button` | **2** |
| Occurrences of `bg-indigo-600` (hand-rolled primary) | 379 |
| Files hand-rolling `role="dialog"` | **129** |
| Files hand-rolling `fixed inset-0` scrims | 123 |
| Files importing `ui/overlay-surface` | **0** |
| Duplicated click-outside listeners | 34 |

`components/ui/button.tsx` is a good component — four variants, loading state
with width preservation, `aria-busy`, haptics, reduced-motion awareness, press
scale. It is used in `run-agent-popover.tsx` and `dry-run-dock.tsx`. Nowhere else.

**The system does not even compose with itself.** `components/ui/empty-state.tsx`
declares its own local `ActionButton` and re-implements the primary/secondary
button classes inline rather than importing `Button`. If the design system's own
components do not use the design system, feature code will not either.

**`overlay-surface.tsx` is dead code** — zero importers — while 129 files
hand-roll dialogs. This is the clearest possible signal that primitives are being
authored without a migration path.

Per **R-27**, coverage is the correct adoption metric. Lextures' technical
coverage for its most-used interactive element is **2/795 files ≈ 0.25%**.

---

### G-7 — Typography is too small and has no scale *(MAJOR)*

**Measured.**

| Class | Rendered size | Occurrences |
|---|---|---|
| `text-sm` | 14px | **5,489** |
| `text-xs` | 12px | **2,546** |
| `text-[10px]` / `text-[11px]` | 10–11px | 229 |
| `text-base` | 16px | **168** |

The product's body text is **14px**, with **2,775 instances of 12px or smaller**.
`text-base` — the browser default and the conventional floor for body copy — is
used 168 times across 795 files.

This compounds badly with G-1: the most common secondary-text colour is
`text-slate-500` (#64748b), which sits at ~4.76:1 on white. That passes AA at
normal size, but the combination of *near-floor contrast* at *12px* across
thousands of elements is the wrong default for a product whose users include K-12
students, learners with dyslexia, and learners with low vision.

There is no type scale, no line-length constraint on body content, and no
semantic heading/body/caption roles. `docs/design.md` specifies "Plus Jakarta
Sans or Inter"; the app actually ships a custom `Lextures` typeface in four
weights. The document has never been reconciled.

Positive: `text-wrap: balance` on headings and `text-wrap: pretty` on body are
applied in `index.css` — good, current practice.

---

### G-14 — Geometry and elevation have no rules *(MINOR)*

| Radius | Uses | | Shadow | Uses |
|---|---|---|---|---|
| `rounded-lg` (8px) | 1,393 | | `shadow-sm` | 635 |
| `rounded-xl` (12px) | 1,177 | | `shadow-xl` | 128 |
| `rounded-md` (6px) | 829 | | `shadow-lg` | 84 |
| `rounded-full` | 393 | | `shadow-md` | 28 |
| `rounded-2xl` (16px) | 332 | | `shadow-card` *(the token)* | **5** |

Five radii in heavy use with no rule mapping surface type → radius.
`docs/design.md` specifies "12–16px corner radius on cards, inputs, and primary
controls" — i.e. `rounded-xl`/`rounded-2xl`. The most-used radius in the codebase
is `rounded-lg` (8px), which the document does not sanction. The one custom
elevation token (`--shadow-card`) is used 5 times against 635 uses of raw
`shadow-sm`.

---

### G-16 — The design document is stale and unenforced *(MINOR)*

`docs/design.md` (78 lines) is a prose style guide with no enforcement mechanism.
It is contradicted by the code on font stack, radius, and colour semantics.
`docs/design-tokens.md` documents **raw Tailwind hex values**, not semantic
tokens — it is a contrast contract, not a token specification.

Neither document can be validated against a running interface. Per **R-24**, prose
guidance has been superseded by tokens as the mechanism for scaling UI.

---

## 3. Accessibility

> Context: the product publishes a **WCAG 2.1 AA** VPAT
> (`docs/vpat/VPAT_2.5_INT_Lextures_2026-05.md`) and shipped a 12.x accessibility
> programme. The findings below are gaps against that existing claim, and against
> the WCAG 2.2 AA / EN 301 549 bar established in **R-35/R-36**.

### G-3 — Declared ARIA roles without their keyboard contracts *(BLOCKER — legal)*

**Measured.**

| Widget | Files declaring the role | Files implementing arrow-key navigation |
|---|---|---|
| `role="menu"` | **37** | **4** |
| `role="tablist"` | **22** | **0** |

**All 22 tab sets in the product are keyboard-inoperable** as ARIA tabs. Every
one declares `role="tablist"` / `role="tab"`, which tells assistive technology
"this is a tab widget — use arrow keys." None implements `ArrowLeft`/`ArrowRight`,
`Home`/`End`, or roving `tabindex`. **33 of 37 menus** have the same defect.

This is worse than using plain links and buttons. Declaring the role sets an
expectation the implementation does not honour, so a screen-reader user is
actively misled. It fails **WCAG 2.1.1 Keyboard** and **4.1.2 Name, Role, Value**,
and it directly undermines the published VPAT.

Example — `components/layout/top-bar.tsx:124–156`: a `role="menu"` with two
`role="menuitem"` children, `aria-haspopup`, `aria-expanded`, Escape handling and
click-outside — but no arrow keys and no focus management into the menu.

### G-4 — Modals declare `aria-modal` without trapping focus *(BLOCKER — legal)*

| Metric | Count |
|---|---|
| Files with `role="dialog"` | 129 |
| Files with `aria-modal` | 126 |
| Files importing `lib/a11y/focus-trap.ts` | **3** |
| Files handling `Escape` | 111 |

`aria-modal="true"` tells assistive technology that content outside the dialog is
inert. In ~123 dialogs that promise is false: keyboard and screen-reader users can
Tab straight out of the dialog into background content that is visually obscured
by a scrim. There is no `inert`/`aria-hidden` management of the background either.

A correct `focus-trap.ts` **already exists in the repository** and is used three
times.

Also unverified across these 129 dialogs: focus moves *into* the dialog on open,
focus returns to the trigger on close, and the dialog has an accessible name.

### G-5a — Interactive affordances without accessible tooltips

289 uses of the native `title=` attribute as a tooltip. `title` does not appear on
touch, is unreliable for screen readers, cannot be styled, and has an
uncontrollable delay. The repository contains `side-nav-tooltip.tsx`,
`icon-action-tooltip.tsx` and `action-error-tooltip.tsx` — again, primitives
exist; adoption does not.

### G-8 — Form errors are not programmatically associated *(MAJOR — legal)*

| Metric | Count |
|---|---|
| `<input>` elements | **929** |
| `<label>` elements | 1,159 |
| `aria-describedby` | 56 |
| `aria-invalid` | **10** |
| `placeholder=` | 396 |
| Files with a `useState<string \| null>` error | 364 |

Ten `aria-invalid` attributes for 929 inputs. Validation errors are rendered as
free-floating text near a field with no programmatic relationship to it. A screen-
reader user tabbing to a field in an error state is told nothing about the error.

Fails **WCAG 3.3.1 Error Identification** and **3.3.3 Error Suggestion**. With
396 placeholders and only 1,159 labels for 929 inputs, placeholder-as-label
(**WCAG 3.3.2 Labels or Instructions**) is likely present and needs a sweep.

No form library and no shared field component: 4 files use `zod`, the other
validation is bespoke per form.

### G-5b — WCAG 2.2 criteria not yet addressed

Against **R-35**, the following are unaudited:

- **2.5.8 Target Size (Minimum), AA** — 177 uses of `h-6`/`h-7`/`h-8` (24–32px)
  vs 140 uses of a ≥44px touch class. Icon-only controls at `h-8 w-8` with
  `p-1` are likely below 24×24 effective target in several toolbars.
- **2.5.7 Dragging Movements, AA** — `@dnd-kit` powers module reordering,
  gradebook layout, kanban and board surfaces. No keyboard/single-pointer
  alternative was found for these flows.
- **2.4.11 Focus Not Obscured, AA** — the app has a sticky `h-14` top bar, a
  sticky quiz focus bar, a reading focus bar and toast stacks. Focus scroll-into-
  view offsetting was not found.
- **3.2.6 Consistent Help, A** — `help-widget.tsx` is in the top bar, but a
  separate `feature-help` component set and `checklist-help-popover` also exist;
  consistent relative order is unverified.
- **3.3.7 Redundant Entry, A** — unaudited across onboarding and enrollment.
- **3.3.8 Accessible Authentication, AA** — WebAuthn is supported
  (`@simplewebauthn/browser`), which helps; MFA and magic-link flows need review.

### G-5c — Focus indication is inconsistent

Only **75 of 795** files use `focus-visible:`. Most of the 2,016 hand-rolled
buttons inherit whatever the browser default is over a custom background, which
is not a designed, verified-contrast focus indicator. Fails **WCAG 2.4.7 Focus
Visible** in an unknown but large number of places.

---

## 4. Information architecture and navigation

### G-5 — Navigation sprawl and alphabetical ordering *(MAJOR)*

**Global sidebar** (`side-nav-main-links.tsx`) renders up to **28 links** across
6 section labels: *(unlabelled)*, Learning, Notes & portfolio, Records, Family,
Administration, Account. Visibility is driven by ~20 independent feature flags.

**In-course sidebar** (`side-nav-course-links.tsx`) renders up to **40 links**
across 8 sections: Content, Collaboration, Your learning, Assessment, Grades &
insights, People, Manage. The *Grades & insights* section alone contains **15
links**.

Those 15 links are ordered **alphabetically**:

> At-risk · Behavior · Evaluation results · Event log · Final grades ·
> **Gradebook** · Mastery heatmap · Outcomes report · Reading dashboard ·
> Report cards · Reports · Standards coverage · Standards gradebook ·
> What's working · Content Tools

Per **R-12**, NN/g's guidance is to consider alphabetisation only at ~20+ items;
below ~10 it actively harms by destroying frequency ordering. **Gradebook — the
single most-used instructor destination in any LMS — sits sixth**, below "Event
log" and "Evaluation results."

**Duplicate icons and near-duplicate labels** (the semantic-overlap failure mode
of **R-11**):

| Icon | Used for |
|---|---|
| `BookMarked` | Global notebook · My Notebooks · Standards coverage · Standards gradebook |
| `Award` | Credential wallet · My achievements · My credentials · My grades |
| `Activity` | Behavior · Event log |
| `ClipboardList` | Attendance · Gradebook |
| `Video` | Live Sessions · Screen share |
| `BarChart3` | Reports (global) · Reports (course) |
| `Layers` | Modules · Content Tools |

Three global nav items — **Credential wallet**, **My credentials**, **My
achievements** — share one icon and are semantically indistinguishable to a user
who has not been trained on the domain model. Similarly *Reports* exists at both
global and course level, and *Standards coverage* vs *Standards gradebook* are
one word apart with an identical icon.

**No overflow, prioritisation, or personalisation.** Every one of the 114
platform features that owns a nav slot adds a permanent link. There is no "More",
no user-pinned section (course pinning exists via `side-nav-pinned-courses.tsx`,
but not for destinations), no frequency-based ordering, and no way to hide what an
individual never uses. Per **R-13**, the industry has moved decisively toward
fewer top-level categories.

**Nav labels are not internationalised.** `Dashboard`, `Courses`, `Calendar`,
`Todos`, `Modules`, `Syllabus`, `Back`, `Gradebook` and ~60 others are hardcoded
English string literals in both nav files, in a product that ships an RTL Arabic
locale.

### G-12 — `/settings/*` vs `/admin/*` split-brain *(MAJOR)*

The Settings navigation exposes 33 destinations, of which five route into a
*different* console:

```
/admin/accessibility        (labelled under Settings)
/admin/bookstore            (labelled under Settings)
/admin/consortium           (labelled under Settings)
/admin/consent-studies      (labelled under Settings)
```

Meanwhile `/admin/*` has its own layout (`AdminLayout.tsx`), its own nav
(`side-nav-admin-links.tsx`, 15.4 KB) and its own overview page. There are 48
files in `components/settings/` and 46 pages under `pages/admin/`.

An administrator cannot predict where a given setting lives. There is no search
within settings. This is the classic two-hierarchies-for-one-concept IA defect and
is exactly the semantic-overlap problem in **R-11**.

### G-5d — No persistent global search field

`top-bar.tsx` contains breadcrumbs, then eight right-aligned controls
(Reading preferences `Aa`, AI tutor, Canvas import, Feedback, Help,
Notifications, View-as, User menu). **There is no search input.**

Search exists only through the command palette, whose desktop trigger lives in
the **sidebar** (a well-designed full-width capsule with a `⌘K` hint) and whose
top-bar trigger is `md:hidden` — mobile only.

The palette itself is a genuine strength and matches **R-29**. But per **R-30**,
a palette serves users who already know the vocabulary. For 200 routes, students,
and occasional users, the absence of a persistent search affordance in the primary
chrome is a findability gap.

---

## 5. Core surfaces

### G-6 — The dashboard is an unprioritised banner farm *(MAJOR)*

`pages/lms/dashboard.tsx` is 1,487 lines and renders a **single vertical stack**
of ~18 independent, full-width, equally-weighted sections:

> Quick links → Intro welcome banner → Intro course card → Start-here card →
> What's next → Study stats → Daily goal → Study buddy prompts → Gamification →
> Recent certificates → Learning paths → Degree progress → Research consent →
> Continuing education → Achievements → Credentials → Notebook tasks →
> Review practice → Self-paced → per-course sections → Grading backlog

Each is gated by a feature flag, so enabling a feature appends another banner.
There is:

- **no grid** — everything is full-width, so nothing is subordinate to anything;
- **no priority** — "Research consent" has the same visual weight as "What's next";
- **no personalisation** — the user cannot reorder, collapse or dismiss;
- **no density control**;
- **no role differentiation** — instructors scroll past student-motivation cards
  to reach the grading backlog, which is rendered *last*.

Each banner also carries its own accent colour (teal / violet / emerald / amber /
indigo) with no semantic system — see G-1.

Against **R-32**, the reference model is three layers: KPIs at a glance, detail on
click, configuration on intent. The dashboard implements one layer, eighteen times.
Against **R-8**, eighteen equally-weighted calls to action is the textbook
condition for choice overload.

### G-11 — Data tables are unsystematised

99 files contain `<table>`; **81** wrap it in `overflow-x-auto`, leaving ~18
tables that will force horizontal page scroll on narrow viewports. There is no
shared table component, so column sizing, sticky headers, sort affordances, row
selection, keyboard navigation, empty rows and pagination are re-solved per table.

`gradebook-grid.tsx` is 2,030 lines with a separate 1,000-line transposed variant.
Per **R-31**, "excellent tables" are the centre of gravity for professional tools
in 2026; this is a strategically important surface with no system behind it.

### G-15 — God components block surface work *(MAJOR)*

| File | Lines | `useState` |
|---|---|---|
| `pages/lms/course-module-quiz-page.tsx` | 3,403 | — |
| `pages/lms/course-modules.tsx` | 3,284 | **100** |
| `components/quiz/quiz-student-take-panel.tsx` | 2,196 | — |
| `pages/lms/gradebook/gradebook-grid.tsx` | 2,030 | — |
| `pages/lms/course-enrollments.tsx` | 1,913 | — |
| `pages/lms/course-settings.tsx` | 1,884 | — |

One hundred `useState` hooks in a single component is not a surface that can be
redesigned; it can only be rewritten. This is already tracked as
[`TD.14-decompose-god-components`](../tech_debt/TD.14-decompose-god-components.md)
and **UX.10/UX.11 depend on it**.

---

## 6. Interaction, feedback and state

### G-9 — Three competing loading idioms; almost no error boundaries *(MAJOR)*

| Idiom | Files |
|---|---|
| Literal `Loading…` text | 106 |
| `animate-spin` spinner | 90 |
| `animate-pulse` skeleton | 44 |
| `lms-content-skeletons` (the system component) | **5** |
| `aria-busy` | 35 |

Three different loading treatments coexist, and the *system* skeleton set is the
least used. Per **R-16**, skeletons reduce perceived load time by ~30% and prevent
layout shift; the product mostly ships spinners and raw text.

**Four `ErrorBoundary`/`componentDidCatch` implementations for 200 routes.** A
render error in any of ~310 page components propagates to the nearest boundary —
in most cases, the app root. There is no per-route recovery and no
"something went wrong, retry" surface with the error preserved.

**Offline is essentially unhandled in the UI**: 8 files reference
`navigator.onLine`/offline, despite a full Workbox PWA setup, background sync,
and Dexie/IndexedDB local storage. The offline *infrastructure* exists; the
offline *experience* does not.

### G-13 — Feedback is fragmented *(MAJOR)*

| Pattern | Files |
|---|---|
| Hand-rolled `useState<string \| null>` error banner | **364** |
| `useConfirm` (the good, consistent path) | 41 |
| Importing `lib/lms-toast.ts` | **1** |

`lib/lms-toast.ts` exports `toast`, `toastSaveOk`, `toastMutationError` and
**`toastWithUndo`**, and `<LmsToaster />` is mounted in `main.tsx`. One file
imports it.

`toastWithUndo` is precisely the pattern **R-21** identifies as strictly dominant
for reversible destructive actions — the system acts, then offers timed undo. It
is implemented and unused, while 364 components hand-roll inline error banners
with per-file styling.

**Credit where due:** `components/use-confirm.tsx` is used consistently at 41
sites with `variant: 'danger'` and i18n'd titles. This is the one feedback pattern
that works. Its problem is the opposite of the others — per **R-20/R-22**, some of
those 41 confirmations are on *reversible* actions and should be undo instead.

### G-10 — i18n coverage is 34% *(MAJOR)*

| Metric | Value |
|---|---|
| Locales | 4 (`en`, `es`, `fr`, `ar`) |
| Keys per locale | 3,746 — **perfectly in sync across all four** |
| Namespaces | 10 |
| Files using `useTranslation` | **273 / 795 (34%)** |

The pipeline is excellent: four locales at exact key parity, ICU message
formatting, RTL locale list, missing-key handling, a lint plugin, and a CI check.
Coverage is the problem — roughly two-thirds of the product's user-visible text is
hardcoded English, including both navigation files.

Because `ar` is shipped, this is not a future concern: an Arabic user today
navigates an English sidebar. Logical properties (`ms-`/`ps-`/`end-`) are used
correctly in the chrome, and a `convert-physical-tailwind.mjs` script exists — so
the RTL *layout* work was done. Only the strings were not.

### G-11 — Responsive coverage is 32% *(MAJOR)*

| Breakpoint prefix | Occurrences |
|---|---|
| `sm:` | 659 |
| `md:` | 161 |
| `lg:` | 118 |
| `xl:` | 19 |
| `2xl:` | 1 |

**257 of 795 files (32%)** contain any responsive prefix at all. `docs/plan/mobile/`
and `docs/accessibility/mobile-audit-checklist.md` exist, and the native
iOS/Android clients absorb much of the mobile load — but the web app is used on
tablets and small laptops, and 68% of components have a single fixed layout.

Combined with 18 non-scrolling tables and unverified 24px targets (G-5b), narrow
viewports are the least-tested dimension of the product.

---

## 7. Performance

### G-17 — No per-route web-vitals budget *(MAJOR)*

Baseline (`scripts/bundle-baseline.json`): entry **245,104 B gzip**, dashboard
chunk 12,291 B gzip.

A 245 KB gzipped entry bundle (~1 MB parsed) is heavy for an application whose
first meaningful screen is a dashboard. Route-level code splitting is in place
(`lazy-pages.ts`, 16.4 KB of lazy imports) and bundle-size CI checks exist —
good — but:

- there is **no Core Web Vitals budget** (LCP / INP / CLS) per route class;
- the only Lighthouse artefact is a single dark-mode dashboard run
  (`docs/lighthouse/global-dashboard-darkmode.json`);
- the dashboard fires **N+1 requests** — `mapPool` fans out per-course structure,
  grades, gradebook grid, feed channels and feed messages for every enrolled
  course, which is what makes an 18-section dashboard slow as well as noisy.

INP is the criterion most at risk given the god components in G-15.

---

## 8. Motivation and pedagogy

### G-18 — Progress and motivation are features, not a model *(MINOR)*

The product has gamification (`components/gamification/`), badges, streaks
(`Flame` icon on the review card), a leaderboard widget
(`pages/lms/LeaderboardWidget.tsx`), study stats, daily goals, credentials and
learning paths. These are shipped independently and surfaced as sibling banners.

Against **R-4/R-5/R-6**:

- **Competence** is the weakest-supported need, and gamification is
  meta-analytically *ineffective* at supporting it. Points and badges are present;
  a coherent, visible "here is what you now know and what's next" mastery signal
  is not the primary dashboard element — `What's next` is one of eighteen
  co-equal cards.
- **Autonomy** is unsupported at the interface level: the user cannot shape the
  dashboard, ordering, or density.
- **Relatedness** signals (presence, feed, discussions) exist but are scattered.
- A **leaderboard widget** exists. Per **R-6**, ranking must be opt-in and scoped
  or it reads as surveillance. Its default state and scope need an explicit
  policy decision.

---

## 9. What is already good

It matters that this list is real, because it establishes that the team can build
to this standard when the mechanism exists.

| Strength | Evidence |
|---|---|
| **Motion system** | 10 duration tokens, 3 easings, documented bubble spring, per-UI-mode timing (k2 / elementary), reduced-motion handling everywhere, CI enforcement via `check-interface-polish.mjs`. **The proof that tokens+CI works here.** |
| **Command palette** | Well-designed full-width capsule trigger with `⌘K` hint, sensible `aria-label`, onboarding hook, motion presence. Matches R-29. |
| **i18n infrastructure** | 4 locales at exact 3,746-key parity, ICU formatting, RTL locale registry, missing-key telemetry, lint plugin, CI check. |
| **RTL layout work** | Logical properties (`ms-`, `ps-`, `end-`) used correctly in chrome; conversion script exists. |
| **`useConfirm`** | Consistently adopted (41 sites), i18n'd, danger variant. |
| **Reading accessibility** | `ReadingPreferencesPanel`, reading ruler, read-aloud with non-colour-only sentence highlight (explicitly noted as WCAG 1.4.1), caption styling variables, high-contrast stylesheet, k2/elementary UI modes. |
| **Contrast CI** | `check-contrast.mjs` + `contrast-config.json` + `docs/design-tokens.md` — the right idea, applied to the wrong layer (allowlist rather than tokens). |
| **PWA / offline infra** | Workbox precaching, background sync, Dexie. |
| **Structural guardrails** | Bundle-size checks, entity-label checks, platform-feature-toggle checks, i18n checks, interface-polish checks. A real ratchet culture already exists to build on. |

---

## 10. Traceability — findings to plans

| Finding | Addressed by |
|---|---|
| G-1 Tokens | [UX.1](UX.1-semantic-design-token-system.md) |
| G-2 Component adoption | [UX.2](UX.2-core-component-library-and-adoption-ratchet.md) |
| G-7 Typography | [UX.3](UX.3-typography-and-reading-system.md) |
| G-3, G-4, G-5a, G-5c ARIA | [UX.4](UX.4-aria-widget-and-focus-management-remediation.md) |
| G-5b WCAG 2.2 | [UX.5](UX.5-wcag-2.2-aa-conformance-uplift.md) |
| G-8 Forms | [UX.6](UX.6-form-and-validation-system.md) |
| G-5, G-5d Navigation | [UX.7](UX.7-navigation-information-architecture.md) |
| G-12 Settings IA | [UX.8](UX.8-settings-and-admin-ia-unification.md) |
| G-6 Dashboard | [UX.9](UX.9-role-aware-dashboard.md) |
| G-15 Course surfaces | [UX.10](UX.10-course-home-and-learning-flow.md) |
| G-11 Tables | [UX.11](UX.11-data-table-and-gradebook-system.md) |
| G-9 States | [UX.12](UX.12-loading-empty-error-offline-states.md) |
| G-13 Feedback | [UX.13](UX.13-feedback-undo-and-destructive-actions.md) |
| G-11 Responsive | [UX.14](UX.14-responsive-and-small-viewport-experience.md) |
| G-10 i18n | [UX.15](UX.15-i18n-coverage-and-rtl-completion.md) |
| G-18 Motivation | [UX.16](UX.16-progress-motivation-and-learner-agency.md) |
| G-17 Performance | [UX.17](UX.17-perceived-performance-and-web-vitals-budget.md) |
| G-2, G-16 Governance | [UX.18](UX.18-design-system-governance-and-measurement.md) |
