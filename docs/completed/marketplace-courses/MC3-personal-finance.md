# MC3 — Personal Finance (Course Content & Assessments)

> Implementation plan. Source: [docs/plan/marketplace-courses/README.md](../../plan/marketplace-courses/README.md). A first-party, **free** marketplace course provisioned via the [MC0](MC0-authoring-provisioning-foundation.md) harness.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | MC3 |
| **Section** | Marketplace Courses |
| **Severity** | MAJOR |
| **Markets** | SL (primary) · HE · K12 (grades 9+) |
| **Status (today)** | COMPLETED |
| **Estimated effort** | M (3–4w authoring + review) |
| **Owner (proposed)** | Content team + personal-finance SME reviewer |
| **Depends on** | MC0 |
| **Unblocks** | — |
| **Course code (proposed)** | `PERSONAL-FINANCE` |
| **Catalog** | slug `personal-finance` · category `Life Skills` · difficulty `beginner` · language `en` · **price `$0`** |
| **Length** | 7 modules · ~5–7 hours · self-paced |
| **Scope note** | **US-focused**; educational only — **not** personalized financial, tax, or legal advice (§6). |

---

## 1. Problem Statement

Most people never receive formal instruction in managing money, and the internet's financial content is a minefield of conflicting advice, sales pitches, and outright scams. The marketplace needs a trustworthy, **free**, vendor-neutral personal-finance course that teaches the durable fundamentals — budgeting, saving, credit, debt, investing basics, and fraud protection — grounded exclusively in **government and nonprofit** sources (CFPB, SEC/Investor.gov, FDIC, MyMoney.gov, Khan Academy). It is broadly appealing across every market and completes a well-rounded free launch trio alongside AI Essentials and Introduction to Python.

## 2. Goals

- Give any adult or older teen the **core competencies** to budget, save, manage credit/debt, understand investing basics, and avoid fraud.
- Ground **every** claim in official/nonprofit sources; take **no** product or vendor position.
- Be **practical**: learners produce their own budget, a debt-payoff plan, and a simple financial plan.
- Be explicit that it is **educational, US-focused, and not individualized advice.**
- Free, `marketplace_listed`, provisioned reproducibly via MC0.

## 3. Non-Goals

- **No personalized advice**, product recommendations, stock/crypto picks, or affiliate links.
- Not tax preparation or legal guidance (points to IRS/professionals instead).
- Not advanced investing (options, real estate, small-business finance) — fundamentals only.
- Not exhaustively multi-jurisdiction; US-centric at GA, with a note to check local rules (localization/other markets = §18).
- No certification/credential in this story.

## 4. Personas & User Stories

- **As a young adult / new earner**, I want to learn budgeting and credit so that I can manage my first paycheck and avoid debt traps.
- **As a household budgeter**, I want a clear method and a template so that I can control cash flow and build an emergency fund.
- **As a nervous beginner investor**, I want plain-language fundamentals and fraud red flags so that I can start safely.
- **As a student (HE/K12 9+)**, I want a trustworthy money-basics course so that I'm prepared for financial independence.
- **As an instructor / counselor**, I want a vetted, vendor-neutral free course to recommend so that I can trust the material.

## 5. Functional Requirements

- **FR-1.** The course MUST contain 7 modules, each with ≥3 content pages, a summary, and one auto-scored knowledge check (5–8 questions).
- **FR-2.** Every content page MUST cite at least one **government or nonprofit** source from §19; each external link MUST pass the MC0 link-checker. Commercial/vendor sources are prohibited as authorities.
- **FR-3.** The course MUST include applied assignments: a **monthly budget** (M2), a **savings/compound-interest** exercise using the Investor.gov calculator (M3/M5), a **debt-payoff plan** (M4), and a **capstone financial plan** (M7). Assignments are auto-graded (`completion_full`/`grader_agent`).
- **FR-3a.** Assignments MUST NOT require learners to enter real account numbers, balances, SSNs, or other sensitive PII; templates use learner-chosen or illustrative figures, with an explicit warning.
- **FR-4.** The course MUST display a persistent **disclaimer** (educational only, US-focused, not individualized financial/tax/legal advice) on the syllabus and the first page of every module that discusses money decisions.
- **FR-5.** The course MUST be provisioned `price_cents=0`, `published=true`, `marketplace_listed=true`, `difficulty_level='beginner'`, `catalog_category='Life Skills'`, `catalog_slug='personal-finance'`.
- **FR-6.** Investing content MUST emphasize diversification, low costs, long-time-horizon compounding, and **fraud red flags**, and MUST NOT recommend specific securities, asset classes as "best", or timing strategies.
- **FR-7.** Where figures/limits change annually (e.g., contribution limits, tax brackets), content MUST link to the official source of current numbers rather than hard-coding them, or clearly label them "as of {year} — verify current figures at {source}."
- **FR-8.** Content SHOULD be Grade 9–11 reading level, jargon defined on first use and collected in a glossary; SHOULD use worked examples.

## 6. Non-Functional Requirements

- **Accuracy & neutrality** — SME (personal-finance) sign-off; all authorities are .gov/nonprofit; no vendor bias; dated figures sourced per FR-7.
- **Compliance/liability** — persistent educational-only disclaimer (FR-4); no advice-giving language ("you should buy…"); PII-minimizing assignments (FR-3a). Legal review of the disclaimer and §19 list.
- **Accessibility** — WCAG 2.1 AA: alt text on charts/diagrams; data tables have headers; descriptive links; ordered headings; no color-only meaning.
- **Privacy** — no PII collected; assignment templates warn against entering sensitive data into third-party tools.
- **Internationalization** — EN + US at GA; prose ES fast-follow; other-jurisdiction adaptation noted (§18). Currency/number formatting locale-safe in prose.
- **Reliability/Observability** — provisioned + validated via MC0; link-check in CI.
- **Maintainability** — annually-changing numbers isolated and sourced, so refreshes are localized.

## 7. Acceptance Criteria

- **AC-1.** *Given* MC0 is deployed, *When* `provision-marketplace-courses --only personal-finance` runs, *Then* the course appears in the storefront as **Free**, `beginner`, category **Life Skills**, with 7 modules, a syllabus of outcomes, and the disclaimer visible.
- **AC-2.** *Given* any money-decision module, *When* a learner opens it, *Then* the educational-only disclaimer is present and external links resolve.
- **AC-3.** *Given* the M2 budget assignment, *When* a learner submits a completed budget template (illustrative figures), *Then* it is auto-graded (`completion_full`) and recorded, with **no** sensitive PII required.
- **AC-4.** *Given* the M5 compounding exercise, *When* a learner reports results from the Investor.gov compound-interest calculator, *Then* the submission is accepted and explained.
- **AC-5.** *Given* CI, *When* the validator + link-checker run, *Then* all quiz answers resolve and every §19 link returns 2xx/3xx.

## 8. Learning Outcomes

By the end of Personal Finance, a learner can:

1. **Set goals** — write SMART financial goals and distinguish needs from wants aligned to personal values.
2. **Budget** — build and maintain a monthly budget (e.g., 50/30/20 or zero-based), track cash flow, and size an emergency fund.
3. **Bank & save** — choose appropriate accounts; explain interest, APY, **compound growth**, and **FDIC deposit insurance**.
4. **Manage credit & debt** — read a credit report and score, explain APR, and build a debt-payoff plan (avalanche vs. snowball).
5. **Understand investing basics** — explain compound interest, risk/return, diversification, index funds, and **tax-advantaged retirement accounts**, and recognize **investment-fraud red flags**.
6. **Plan for big items** — describe the basics of taxes, insurance, and major expenses (housing, vehicles, student aid) and how to prepare.
7. **Protect yourself** — spot scams and identity theft, respond appropriately, and find **trustworthy** sources of help.

## 9. Syllabus (authored `syllabus.json` sections)

- **Course overview** — what it covers, who it's for, ~5–7h, self-paced; **educational-only / US-focused disclaimer**.
- **What you'll learn** — the seven outcomes above.
- **Module guide** — the table in §10.
- **Grading & completion** — knowledge checks auto-scored; budget + debt-plan + compounding + capstone auto-graded; complete = final knowledge check + capstone.
- **A note on trust** — why we cite only government/nonprofit sources and take no product position; how to spot financial misinformation.
- **Getting help & going deeper** — the §19 resource list.

## 10. Module Outline

| # | Module | Core content | Assessment focus | Primary sources |
|---|---|---|---|---|
| 1 | **Money Mindset & Goals** | Values & money; needs vs. wants; SMART goals; the MyMoney Five framework | Classify needs/wants; write a SMART goal | MyMoney.gov (MyMoney Five); Khan Academy |
| 2 | **Budgeting & Cash Flow** | Income vs. expenses; 50/30/20 & zero-based; tracking; emergency fund; **budget assignment** | Build a budget; identify overspending | CFPB (budgeting); FDIC Money Smart; Khan Academy |
| 3 | **Banking & Saving** | Checking vs. savings; interest & APY; high-yield savings; **compound interest**; **FDIC insurance** | APY vs. rate; why compounding matters | Investor.gov (compound interest); FDIC; CFPB |
| 4 | **Credit & Debt** | Credit scores & reports (free annual report); APR; good vs. bad debt; snowball vs. avalanche; credit cards; **debt-payoff assignment** | Read a score; pick a payoff strategy | CFPB (credit/debt); AnnualCreditReport.com; FDIC Money Smart |
| 5 | **Investing Basics** | Risk/return; diversification; stocks/bonds/funds; **index funds**; retirement accounts (401(k)/IRA); fees; **fraud red flags**; **compounding assignment** | Diversification; spot a fraud red flag | Investor.gov (SEC); Bogleheads (getting started); MyMoney.gov |
| 6 | **Big Expenses, Taxes & Insurance** | Tax basics & where to file; withholding; insurance types; housing/vehicles; **student aid** | Match insurance to risk; tax basics | IRS.gov; CFPB; Khan Academy |
| 7 | **Protecting Your Money & Next Steps** | Scams & identity theft; safe habits; who to trust; making a simple plan; **capstone** | Spot a scam; assemble a plan | CFPB (fraud); Investor.gov (avoiding fraud); FDIC |

## 11. Assessments (detail)

**Per-module knowledge checks** — 5–8 auto-scored questions (`multiple_choice`, `true_false`), instant feedback, low stakes. Examples:

- *M2 (multiple_choice):* "In the 50/30/20 guideline, the 20% is intended for:" → correct: "Savings and debt repayment." Distractors: "Rent"; "Entertainment"; "Groceries."
- *M3 (true_false):* "FDIC insurance protects deposits at an insured bank up to the standard limit if the bank fails." → **True.**
- *M4 (multiple_choice):* "The debt-**avalanche** method pays extra toward the debt with the:" → correct: "Highest interest rate first." Distractors: "Smallest balance first"; "Longest term"; "Newest account."
- *M5 (multiple_choice):* "Which is a classic **investment-fraud red flag**?" → correct: "Guaranteed high returns with little or no risk." Distractors: "A diversified low-cost index fund"; "A written prospectus"; "An SEC-registered adviser."

**Applied assignments (`completion_full`/`grader_agent`), PII-minimizing:**

- **M2 — Monthly Budget (10 pts):** complete a provided budget template with learner-chosen/illustrative figures across income and 50/30/20 categories; submit + a 2–3 sentence reflection on one change to make. *(No real account data required.)*
- **M4 — Debt-Payoff Plan (10 pts):** given a sample set of debts (or the learner's illustrative ones), choose avalanche or snowball, order the payoffs, and justify the choice.
- **M5 — Compounding Exercise (10 pts):** use the **Investor.gov Compound Interest Calculator** to compare an early-start vs. late-start scenario and report the difference + one takeaway.
- **M7 — Capstone: My Simple Financial Plan (15 pts, `grader_agent`):** assemble a one-page plan — a SMART goal, a budget snapshot, an emergency-fund target, a debt or savings step, and one fraud-protection habit. Demonstrates outcomes 1–7. Illustrative figures only.

## 12. Data Model

No new schema — MC0 harness + existing structure/quiz/syllabus tables. Course row: `catalog_slug='personal-finance'`, `catalog_category='Life Skills'`, `difficulty_level='beginner'`, `catalog_language='en'`, `price_cents=0`, `is_public=true` (recommended), `marketplace_listed=true`.

## 13. API Surface

None new. Served by existing course/storefront/claim/quiz endpoints.

## 14. UI / UX

Existing storefront card, syllabus (with disclaimer), module reader, and quiz player. Card copy: *"Personal Finance — Budget, save, manage credit, and invest with confidence. Vendor-neutral, from trusted public sources. Free."* Budget/plan templates render as Markdown tables (accessible headers). Hero image: embedded neutral finance-motif banner (`assets/personal-finance-banner.jpg`), provisioned into course-files and wired to `hero_image_url` (same pattern as other official marketplace courses). The educational-only disclaimer appears on the landing page and each money-decision module's first page (FR-4).

## 15. AI / ML Considerations

`grader_agent` grades the capstone/reflections for good-faith completeness against the rubric — it MUST NOT give individualized financial advice in feedback (system-prompt constraint); it comments on completeness/clarity only. Reuses the intro-course grading path.

## 16. Integration Points

MC0 harness; `coursemodulequiz`; `grader_agent` (with an advice-avoidance guardrail); CI link-check over `content/en/personal-finance/**`.

## 17. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Reads as financial advice / liability | M | H | Persistent educational-only disclaimer (FR-4); advice-free language; legal review; grader guardrail |
| Annually-changing numbers go stale | H | M | Don't hard-code limits/brackets; link official current-figure sources (FR-7); scheduled review |
| US-centric content misleads other markets | M | M | Label scope; teach transferable principles; note "check local rules"; localization backlog (§18) |
| Perceived vendor bias / looks like a sales funnel | M | H | Only .gov/nonprofit authorities; no products/affiliates; neutral examples |
| Learners enter sensitive PII in assignments | M | H | PII-minimizing templates + explicit warnings (FR-3a) |
| Link rot | M | M | MC0 CI link-checker; stable .gov/nonprofit domains |

## 18. Rollout Plan

Provision to staging → SME + **legal** review (disclaimer, neutrality, no-advice language) → a11y (axe) pass → GA (list; set `is_public`). Rollback = unlist. **Backlog:** non-US localization/adaptation (different tax/benefit systems) and an ES translation of prose; annual figure-refresh job.

## 19. References (verified, link-checked)

Authoritative government/nonprofit sources backing the content. All URLs confirmed reachable during planning.

- **Consumer Financial Protection Bureau (CFPB)** — consumer guides on budgeting, credit, debt, and fraud — https://www.consumerfinance.gov/
- **Investor.gov (U.S. SEC)** — investing basics, avoiding fraud, and the **Compound Interest Calculator** — https://www.investor.gov/ · calculator https://www.investor.gov/financial-tools-calculators/calculators/compound-interest-calculator
- **MyMoney.gov (FLEC, U.S. Treasury)** — federal financial-literacy hub and the **MyMoney Five** framework — https://www.mymoney.gov/ · https://www.mymoney.gov/mymoney-five-tools
- **FDIC Money Smart for Adults** — free, research-backed financial-education curriculum — https://www.fdic.gov/consumer-resource-center/money-smart-adults
- **Khan Academy — Personal Finance** — free lessons on budgeting, saving, credit, and investing — https://www.khanacademy.org/college-careers-more/personal-finance
- **AnnualCreditReport.com** — the federally authorized site for free credit reports (Module 4) — https://www.annualcreditreport.com/
- **IRS.gov** — official source for tax basics, filing, and current figures (Module 6; per FR-7) — https://www.irs.gov/
- **Bogleheads Wiki — Getting Started** — respected nonprofit community primer on low-cost, diversified investing — https://www.bogleheads.org/wiki/Getting_started
- Internal: [MC0](MC0-authoring-provisioning-foundation.md) (harness), `../../plan/marketplace/README.md` (commerce), `server/internal/service/introcourse` (pattern).
