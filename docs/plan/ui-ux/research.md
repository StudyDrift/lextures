# UX / UI Research Dossier

> Evidence base for the `UX.*` plans in this folder. Compiled 2026-08-05.
> Every claim in [audit.md](audit.md) and every design decision in a `UX.*` plan
> should trace back to a numbered finding here (`R-n`).

---

## 0. How to read this document

- **§1–§4** — the psychology. *Why* an interface choice helps or hurts a learner.
- **§5–§8** — the practice. Patterns the industry has converged on, and the
  evidence for them.
- **§9** — the 2024–2026 trend line. What "current" looks like and which trends
  are substance vs. fashion.
- **§10** — competitive baseline for LMS specifically.
- **§11** — the legal floor (accessibility law is now a shipping gate, not a
  nice-to-have).
- **§12** — the measurement model we adopt.
- **§13** — sources.

A **research finding** is written `R-n`. A finding is only recorded here when it
either (a) has empirical backing, or (b) is a near-universal industry convention
whose violation reliably surprises users.

---

## 1. Cognitive load — the governing theory for a learning product

Cognitive Load Theory (Sweller) partitions the demand an activity places on
working memory into three types:

| Type | Source | Design implication |
|---|---|---|
| **Intrinsic** | Inherent difficulty of the material | Not ours to reduce — this *is* the course |
| **Extraneous** | How information is *presented* | **This is the entire surface area of UI design** |
| **Germane** | Effort spent building schemas | What we want to protect and maximise |

The critical asymmetry for an LMS: working-memory capacity is fixed and shared.
Every unit of extraneous load the interface imposes is subtracted directly from
the germane load available for learning. In a productivity tool a confusing menu
costs seconds; in a learning tool it costs *comprehension of the material the
user came for*.

> **R-1.** In educational software, UI complexity is not merely an efficiency
> cost — it is a **learning-outcome** cost. 2024–2025 studies of online
> experiment/virtual-lab teaching found that UI quality and communication quality
> both significantly predicted learning outcomes, and that **extraneous cognitive
> load partially mediated the relationship**. Improving the interface improved
> outcomes *through* the mechanism of reduced extraneous load. (Sources: §13.1,
> §13.2)

> **R-2.** Mobile/small-screen learning UIs elevate extraneous load specifically.
> A validated instrument now exists for measuring UI-induced extraneous load in
> mobile learning apps, and poor UI design is called out as a primary driver.
> Responsive quality is therefore a **pedagogical** requirement for us, not a
> convenience one. (Source: §13.3)

> **R-3.** The standard mitigation is *segmenting*: present and organise
> information in smaller pieces to avoid overload and increase retention. In
> interface terms this is progressive disclosure plus chunked navigation.
> (Source: §13.1)

### 1.1 Consequences we adopt as design law

1. **Default density is a learner-safety setting.** A screen that shows 18
   equally-weighted things is a screen that shows nothing.
2. **Decorative variety is extraneous load.** Five accent colours with no
   semantic mapping force the learner to re-derive meaning on every screen.
3. **Inconsistency is load.** Two different button shapes for the same action
   class force re-identification. Consistency is not aesthetics; it is a working-
   memory subsidy.

---

## 2. Motivation — Self-Determination Theory

Deci & Ryan's SDT holds that durable motivation requires three innate needs to be
met: **autonomy** (I chose this), **competence** (I am getting better),
**relatedness** (I am among people). NN/g has explicitly translated SDT into UX
practice.

> **R-4.** Satisfying autonomy / competence / relatedness produces higher-quality
> motivation and sustained engagement; supports for these needs enhance both
> intrinsic motivation *and* well-internalised extrinsic motivation across
> educational levels and cultures. (Sources: §13.4, §13.5)

> **R-5.** Gamification is not a uniform good. A meta-analysis of 35 independent
> interventions (~2,500 participants) found gamification produced a positive,
> significant effect on **autonomy and relatedness** but **minimal effect on
> competence**. Points/badges/leaderboards therefore do *not* substitute for
> genuine progress feedback. (Source: §13.6)

> **R-6.** Competitive metrics can actively backfire. Review work on behaviour-
> change technology found ranking systems were perceived as **surveillance
> rather than motivation** in some populations. Leaderboards must be opt-in,
> scoped, and never the primary progress signal. (Sources: §13.7, §13.8)

### 2.1 SDT → interface mapping used by these plans

| Need | Anti-pattern | Pattern we adopt |
|---|---|---|
| Autonomy | Fixed dashboard the user cannot shape; forced modals | Reorderable/dismissible dashboard sections; density control; "not now" that persists |
| Competence | Grade % as the only feedback; hidden progress | Visible mastery/progress state, next-step clarity, error recovery that teaches |
| Relatedness | Silent, single-player LMS | Presence, instructor voice, cohort signals — scoped, never public shaming |

---

## 3. Decision cost — Hick's Law and choice overload

Hick's Law: time to choose among *n* equally likely options grows with
`log2(n+1)`.

> **R-7.** The relationship is logarithmic, not linear — which means the payoff
> is in *structure*, not deletion. Splitting 30 flat items into 5 groups of 6 is
> a far larger win than deleting 5 items from the flat 30. Grouping, chunking and
> progressive disclosure are the sanctioned mitigations for long menus.
> (Sources: §13.9, §13.10)

> **R-8.** Beyond a threshold, extra options stop being a feature: users
> experience choice overload, leading to frustration, task abandonment, or
> **decision paralysis** — not deciding at all. (Source: §13.10)

> **R-9.** Progressive disclosure has ~4 decades of supporting evidence: novices
> learn faster and err less, while experts pay at most one extra click. It is the
> highest-leverage single pattern for an application with a wide feature surface.
> (Sources: §13.11, §13.12)

---

## 4. Findability and information architecture

> **R-10.** NN/g's position on menu length is explicitly *not* "7±2." Four
> factors decide it, and the governing rule is: *the number of categories should
> be determined by what makes it easiest for people to discover and access
> information.* Complex products legitimately need more categories than simple
> ones. (Source: §13.13)

> **R-11.** **Too many categories harm findability primarily through semantic
> overlap** — users cannot choose because several labels plausibly contain what
> they want. The failure mode is ambiguity, not length. Two nav items with the
> same icon and near-identical names are the worst case. (Source: §13.13)

> **R-12.** NN/g's only hard numeric guideline in this area: consider
> **alphabetical ordering only for lists of ~20+ items**; below ~10 items
> alphabetisation actively hurts, because it destroys the frequency/task ordering
> that helps users. (Source: §13.13)

> **R-13.** Intranet IA trends show sustained movement *toward fewer* top-level
> categories: 11% of studied companies had >12 categories in 2007; **none did by
> 2014**. Wide flat navigation is a receding practice. (Source: §13.14)

> **R-14.** Vertical (sidebar) lists are the correct container when a product
> genuinely has many categories: easy to expand, easy to scan, familiar. Our
> sidebar choice is sound; the *contents* are the problem. (Source: §13.13)

> **R-15.** Low findability has four diagnosable causes, each with a matching
> test method (tree testing for structure, first-click for labels, etc.).
> IA changes must be **tested**, not argued. (Source: §13.15)

---

## 5. Interface state — loading, empty, error

> **R-16.** Skeleton screens outperform spinners because they communicate *what
> is coming*, set layout expectations, and prevent layout shift. Reported
> perceived-load-time reduction is on the order of **~30%**, and skeletons are
> the default at YouTube/Facebook/LinkedIn scale. (Sources: §13.16, §13.17)

> **R-17.** Skeletons are not universally better. They win when content shape is
> predictable and the wait is short-to-medium. For genuinely long or
> indeterminate operations, honest progress communication beats a fake layout.

> **R-18.** Empty states are **onboarding surface, not error surface**. A good
> empty state explains what belongs here, why it is empty, and the single next
> action. Treating "no data" as a dead end wastes the highest-intent moment a
> new user has. (Source: §13.18)

> **R-19.** Empty / loading / error are the three states most reliably omitted in
> fast-moving codebases, and their absence is felt hardest by new and low-
> confidence users — exactly the LMS student population. (Source: §13.18)

---

## 6. Destructive actions, confirmation, and undo

> **R-20.** **Friction should be proportional to reversibility and damage.** An
> action that can be un-done in one click deserves ~zero friction. NN/g's
> position: confirmation dialogs prevent errors *only if not overused* — an
> over-used confirm becomes a reflex click and stops protecting anything.
> (Sources: §13.19, §13.20)

> **R-21.** For reversible actions, **undo strictly dominates confirm**: the
> system acts immediately and offers a timed "Undo" affordance. This removes an
> interruption from the 99% of cases where the user meant it, while still
> protecting the 1%. Practitioner consensus puts confirm dialogs as the wrong
> choice in the large majority of instances. (Sources: §13.20, §13.21)

> **R-22.** Confirmation is still correct for the genuinely irreversible and
> high-blast-radius (delete a course with submissions, publish grades to SIS,
> revoke an org's access). Those dialogs must **name the object and the
> consequence** and use a verb-specific button ("Delete 3 submissions"), never
> "OK". (Source: §13.22)

> **R-23.** Optimistic UI — render the result immediately, reconcile/revert on
> failure — is the standard perceived-performance pattern for CRUD-ish
> interactions, and pairs naturally with undo. (Source: §13.18)

---

## 7. Design systems and tokens

> **R-24.** Design-token usage jumped from **56% to 84% of teams in a single
> year**; tokens are now the mainstream mechanism for scaling UI, not an
> advanced practice. (Source: §13.23)

> **R-25.** The canonical architecture is **three layers**:
> 1. **Global / primitive** — raw values (`indigo-600`, `#4f46e5`).
> 2. **Semantic** — meaning-based (`color.text.primary`, `color.bg.danger`).
> 3. **Component** — component-scoped overrides (`button.primary.bg`).
>
> Application code should reference **layer 2 only**. Referencing layer 1 from
> feature code is the defect that makes rebrands, dark mode, and high-contrast
> mode impossible. (Source: §13.23)

> **R-26.** The W3C Design Tokens Community Group shipped a **stable Design
> Tokens Format Module in October 2025**, with explicit support for modern colour
> spaces (Display P3, OKLCH), backed by 24+ organisations including Adobe,
> Google, Microsoft, Meta and Figma. Token format is now a standard, not a
> vendor choice. (Source: §13.23)

> **R-27.** **Coverage is the correct adoption metric** — the percentage of UI
> built from system components vs. total. It can be computed technically (import
> analysis), visually (rendered-pixel analysis), or by render counts. Preply's
> 2025 "Visual Coverage" work measures adoption **from the user's perspective**
> by analysing rendered pixels and weighting by UI impact, interactivity and
> accessibility. (Sources: §13.24, §13.25, §13.26)

> **R-28.** Counter-point worth honouring: adoption-as-a-number can be a red
> herring if it becomes the goal. The system exists to make good UI cheap; a
> component nobody wants should be fixed or deleted, not mandated. Coverage
> targets must be paired with a real contribution path. (Source: §13.27)

---

## 8. Keyboard, density, and the "power tool" shape

> **R-29.** Command palettes (Cmd/Ctrl-K) are now a **baseline expectation in any
> SaaS product with more than ~10 features** — Linear, Notion, Raycast, GitHub,
> Slack. The palette works because it lets users *say what they want* instead of
> remembering *where it lives*. It is the standard release valve for a wide
> feature surface. (Sources: §13.28, §13.29)

> **R-30.** A palette does **not** excuse bad IA. It serves users who already
> know the vocabulary. New users, students, and occasional users still navigate
> structurally. Palette + IA, never palette instead of IA.

> **R-31.** The 2025–2026 movement in daily-use professional tools is toward
> **quiet chrome and higher information density**, a return from chart-heavy
> dashboards to *excellent tables*, inline editing, and heavy keyboard support
> (Attio, Retool, Linear, Hex). Density is being re-legitimised after a decade of
> airy marketing-site aesthetics leaking into product UI. (Source: §13.30)

> **R-32.** Progressive disclosure in dashboards is now expressed as a **three-
> layer contract**: (1) high-level KPIs at a glance, (2) detail on click,
> (3) configuration on deliberate intent. This is the reference model for our
> dashboard work. (Sources: §13.31, §13.32)

---

## 9. Visual trend line, 2024 → 2026

Filtering fashion from substance:

| Trend | Verdict | Why |
|---|---|---|
| **Semantic token layers** | **Adopt** | Baseline expectation for 2026; unblocks theming, dark mode, high contrast, white-label (§13.23) |
| **OKLCH / perceptual colour** | **Adopt** | Perceptually uniform lightness makes accessible ramps *derivable* instead of hand-tuned; now standard-supported (§13.23) |
| **Dark-mode surface elevation** | **Adopt** | Best 2026 implementations use multiple near-blacks where closer surfaces are lighter, rather than flat `#000`; produces real depth (§13.23) |
| **Higher density / excellent tables** | **Adopt** | Matches the actual job of gradebooks, enrollments, reports (§13.30) |
| **Command palette / keyboard-first** | **Already have — extend** | Baseline for >10-feature products (§13.28) |
| **AI affordances as designed components** | **Adopt selectively** | Summaries and suggested actions as first-class UI, not bolted-on chat (§13.30) |
| **Design-system consistency across product + site + docs** | **Adopt** | Named explicitly as a 2026 differentiator (§13.30) |
| **Oversized/expressive typography** | **Marketing only** | Appropriate on `www/`; hostile to a dense workspace (§13.23) |
| **3D / WebGPU / spatial** | **Defer** | Only justified where 3D conveys information 2D cannot; no such case in an LMS shell (§13.23) |

---

## 10. Competitive baseline (LMS)

| Product | Shape | What they get right | Where they are weak |
|---|---|---|---|
| **Canvas (Instructure)** | Course-centric, left nav per course | **Instructure UI (InstUI)** — a real, open-source, versioned React design system used across all products; a11y is a first-class contract | Long nav lists; feature sprawl; dated density |
| **Google Classroom** | 4 tabs: Stream / Classwork / People / Grades | **Ruthless IA reduction.** Every major job is ≤2 clicks. Task creation is trivially discoverable | Shallow — gives up power-user depth and reporting |
| **Moodle** | Highly configurable, block-based | Flexibility, plugin ecosystem | Navigational disorientation is a *recurring named finding* in usability literature (§13.33) |
| **Blackboard** | Enterprise, tab-heavy | Institutional depth | Repeatedly rated worst-in-class on usability comparisons |
| **Linear / Notion (out-of-category)** | Keyboard-first, dense, tokenised | Palette, motion discipline, consistency, speed | Not education; no pedagogy |

> **R-33.** The strategic gap in the market: Canvas has the design system but not
> the IA discipline; Google Classroom has the IA discipline but not the depth.
> **A product with Classroom's clarity at Canvas's depth is an unoccupied
> position** — and it is reachable primarily through IA and design-system work,
> not new features.

> **R-34.** Navigational disorientation and cognitive overload are the
> *most-cited* usability defects in the LMS literature specifically, and are
> called out as under-studied relative to their impact. (Source: §13.33)

---

## 11. The legal floor

Accessibility is now a shipping gate in our target markets.

> **R-35.** **WCAG 2.2 (Oct 2023)** added nine criteria. Five are **A/AA** and
> therefore in scope for any AA claim:
> - **2.4.11 Focus Not Obscured (Minimum)** (AA) — sticky headers/widgets must
>   not fully hide the focused element.
> - **2.5.7 Dragging Movements** (AA) — any drag operation needs a single-pointer
>   alternative (e.g. click-source-then-click-target).
> - **2.5.8 Target Size (Minimum)** (AA) — pointer targets ≥ **24×24 CSS px**
>   (with defined exceptions).
> - **3.2.6 Consistent Help** (A) — help affordances in consistent relative order
>   across pages.
> - **3.3.7 Redundant Entry** (A) — don't make users re-enter information already
>   provided in the same process.
> - **3.3.8 Accessible Authentication (Minimum)** (AA) — no cognitive function
>   test without an alternative.
> (Sources: §13.34, §13.35)

> **R-36.** **European Accessibility Act**: the deadline was **28 June 2025** and
> enforcement is live. Products placed on the market before that date have a
> transitional window to 2030; **service contracts concluded before 28 June 2025
> must comply by 28 June 2027**. The presumed-conformance standard is **EN 301
> 549**, which incorporates **WCAG 2.1 AA** plus additional non-web checkpoints.
> Enforcement is not theoretical — formal legal notices were issued within days
> of the deadline. (Sources: §13.36, §13.37)

> **R-37.** Practical consequence for Lextures: our existing VPAT
> (`docs/vpat/VPAT_2.5_INT_Lextures_2026-05.md`) and 12.x accessibility work
> established a **WCAG 2.1 AA** baseline. WCAG 2.2 AA is the current bar for new
> work, and EU sales exposure makes EN 301 549 the contractual standard. Any
> `role` we declare without its full keyboard contract is a defensible-claim
> problem, not just a quality problem.

---

## 12. Measurement model adopted by these plans

Because "the UI feels better" is unfalsifiable, every `UX.*` plan carries at
least one metric from this set.

**Structural / static (CI-enforceable, zero user cost)**
- `design-system-coverage` — % of interactive elements rendered by system
  components (R-27).
- `token-purity` — count of raw palette literals in feature code (target: 0).
- `aria-contract-coverage` — % of declared ARIA roles with their full APG
  keyboard contract implemented.
- `i18n-coverage` — % of user-visible strings externalised.
- `responsive-coverage` — % of route-level components with a verified ≤390px
  layout.

**Runtime / experiential**
- **Core Web Vitals** — LCP, INP, CLS at p75, per route class.
- Entry bundle gzip (current baseline: 245,104 B).
- Task success + time-on-task from moderated tests for the 8 critical journeys.

**Attitudinal**
- **SUS** (System Usability Scale) per persona, baselined before UX.1 ships.
- **SEQ** (Single Ease Question) after each critical task.
- Extraneous-load self-report on learner surfaces, using the mobile-learning
  instrument from §13.3 as a model (R-2).

**Sampling rule.** Baseline everything **before UX.1**. Without a pre-measurement
the whole programme is unaccountable.

---

## 13. Sources

**Cognitive load**
1. Educational Technology — *Cognitive Load Theory: Principles, Learning Processes, and Implications for Instructional Design* — https://educationaltechnology.net/cognitive-load-theory-principles-learning-processes-and-implications-for-instructional-design/
2. *The impact of system interaction quality on learning outcomes in online virtual experiment teaching: the mediating role of extraneous cognitive load* — https://www.ncbi.nlm.nih.gov/pmc/articles/PMC12868159/
3. *User interface design in mobile learning applications: Developing and evaluating a questionnaire for measuring learners' extraneous cognitive load* — https://www.ncbi.nlm.nih.gov/pmc/articles/PMC11422584/

**Motivation / SDT**
4. NN/g — *Autonomy, Relatedness, and Competence in UX Design* — https://www.nngroup.com/articles/autonomy-relatedness-competence/
5. Yu-kai Chou — *Self-Determination Theory: Deci & Ryan's 6 Mini-Theories* — https://yukaichou.com/gamification-analysis/self-determination-theory-guide-to-ryan-and-decis-motivation-framework/
6. Springer (ETR&D) — *Gamification enhances student intrinsic motivation, perceptions of autonomy and relatedness, but minimal impact on competency: a meta-analysis and systematic review* — https://link.springer.com/article/10.1007/s11423-023-10337-7
7. *Designing for Sustained Motivation: A Review of Self-Determination Theory in Behaviour Change Technologies* (Interacting with Computers, OUP) — https://academic.oup.com/iwc/advance-article/doi/10.1093/iwc/iwae040/7760010
8. *The role of gamified learning strategies in student's motivation in high school and higher education: A systematic review* — https://pmc.ncbi.nlm.nih.gov/articles/PMC10448467/

**Decision cost**
9. Laws of UX — *Hick's Law* — https://lawsofux.com/hicks-law/
10. Laws of UX — *Cognitive Load* — https://lawsofux.com/cognitive-load/
11. UX Tigers (Nielsen) — *Progressive Disclosure: From Training Wheels to Week-Long AI Agents* — https://www.uxtigers.com/post/progressive-disclosure
12. Lollypop Design — *Progressive Disclosure UX: Guide + Examples* — https://lollypop.design/blog/2025/may/progressive-disclosure/

**Information architecture**
13. NN/g — *Top 3 IA Questions about Navigation Menus* — https://www.nngroup.com/articles/ia-questions-navigation-menus/
14. NN/g — *Intranet Information Architecture (IA) Trends* — https://www.nngroup.com/articles/intranet-information-architecture-ia/
15. NN/g — *Low Findability and Discoverability: Four Testing Methods to Identify the Causes* — https://www.nngroup.com/articles/navigation-ia-tests/

**Interface state**
16. NN/g — *Skeleton Screens 101* — https://www.nngroup.com/articles/skeleton-screens/
17. LogRocket — *Skeleton loading screen design — How to improve perceived performance* — https://blog.logrocket.com/ux-design/skeleton-loading-screen-design/
18. *Empty States, Loading States, Error States — the three UI states AI almost never builds correctly* — https://blog.vibecoder.me/empty-states-loading-states-error-states

**Destructive actions**
19. NN/g — *Confirmation Dialogs Can Prevent User Errors (If Not Overused)* — https://www.nngroup.com/articles/confirmation-dialog/
20. Josh Wayne — *Confirm or undo? Which is the better option?* — https://joshwayne.com/posts/confirm-or-undo/
21. SaaSUI — *SaaS Destructive Actions & Confirmation UX Patterns (2026)* — https://www.saasui.design/blog/saas-destructive-actions-confirmation-ux-patterns
22. Kinneret Yifrah (UX Collective) — *Are you sure you want to do this? Microcopy for confirmation dialogues* — https://uxdesign.cc/are-you-sure-you-want-to-do-this-microcopy-for-confirmation-dialogues-1d94a0f73ac6

**Design systems / tokens**
23. Timothy Graf — *Design Token Architecture 2026: The Strategic Blueprint for Scalable Design Systems* — https://timgraf.com/ui/design-token-architecture-2026-the-strategic-blueprint-for-scalable-design-systems/
24. Into Design Systems — *How to Measure Design System Impact with Visual Coverage* — https://www.intodesignsystems.com/blog/measure-design-systems-impact
25. Mews Developers — *Building a design system adoption metric from production data* — https://developers.mews.com/design-system-adoption-metric-building/
26. Design Systems Collective — *Measuring Design System Adoption: Building a Visual Coverage Analyzer* — https://www.designsystemscollective.com/measuring-design-system-adoption-building-a-visual-coverage-analyzer-b5d9ae410d42
27. Luis Ouriach — *Design System "Adoption" is a Red Herring* — https://medium.com/@disco_lu/design-system-adoption-is-a-red-herring-6c6b5a504f43

**Keyboard / density / dashboards**
28. SaaSUI — *7 SaaS UI Design Trends for 2026, Shown With Real Screens* — https://www.saasui.design/blog/7-saas-ui-design-trends-2026
29. UX Patterns for Developers — *Command Palette Pattern* — https://uxpatterns.dev/patterns/advanced/command-palette
30. SaaSUI — *What is Command Palette?* — https://www.saasui.design/glossary/command-palette
31. FlowmazeUX — *SaaS Dashboard Design Best Practices: 2026 UX Frameworks* — https://flowmazeux.com/saas-dashboard-design-best-practices/
32. Pixxen — *Progressive Disclosure in SaaS Dashboards: The Playbook* — https://pixxen.com/progressive-disclosure-saas/

**LMS competitive / usability**
33. ACM — *Usability Study of a Learning Management System (LMS)* — https://dl.acm.org/doi/fullHtml/10.1145/3585059.3611415
34. Instructure UI (InstUI) — https://github.com/instructure/instructure-ui

**Accessibility standards & law**
35. W3C — *Web Content Accessibility Guidelines (WCAG) 2.2* — https://www.w3.org/TR/WCAG22/
36. TetraLogical — *What's new in WCAG 2.2* — https://tetralogical.com/blog/2023/10/05/whats-new-wcag-2.2/
37. Covington (Inside Global Tech) — *European Accessibility Act: June 2025 deadline has arrived* — https://www.insideglobaltech.com/2025/06/10/european-accessibility-act-june-2025-deadline-has-arrived/
38. Deque — *EN 301 549 | European standard for digital accessibility* — https://www.deque.com/en-301-549-compliance/

**Internal**
- `docs/design.md` — current (stale) visual direction document
- `docs/design-tokens.md` — current contrast contract
- `docs/vpat/VPAT_2.5_INT_Lextures_2026-05.md` — accessibility conformance claim
- `docs/completed/12-accessibility/` — shipped WCAG 2.1 AA work
- `docs/completed/animations/` — AN.1–AN.7 motion language (the one fully-realised
  token domain in the product)
- `docs/plan/tech_debt/TD.14-decompose-god-components.md` — structural prerequisite
  for several surface rewrites here
