# MC1 — AI Essentials (Course Content & Assessments)

> Implementation plan. Source: [docs/plan/marketplace-courses/README.md](../../plan/marketplace-courses/README.md). A first-party, **free** marketplace course provisioned via the [MC0](MC0-authoring-provisioning-foundation.md) harness.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | MC1 |
| **Section** | Marketplace Courses |
| **Severity** | MAJOR |
| **Markets** | SL (primary) · HE · K12 (grades 9+) |
| **Status (today)** | COMPLETED |
| **Estimated effort** | M (2–4w authoring + review) |
| **Owner (proposed)** | Content team + AI/ML SME reviewer |
| **Depends on** | MC0 |
| **Unblocks** | — |
| **Course code (proposed)** | `AI-ESSENTIALS` |
| **Catalog** | slug `ai-essentials` · category `Technology` · difficulty `beginner` · language `en` · **price `$0`** |
| **Length** | 7 modules · ~4–6 hours · self-paced |

---

## 1. Problem Statement

AI is now embedded in everyday tools, yet most learners lack a clear, non-hype mental model of what it is, how it works, and where it fails. Search results and social media are full of contradictory, often wrong explanations. The marketplace needs a flagship, credible, **free** introduction that gives any adult or older teen a working literacy in AI — enough to use AI tools well, evaluate claims, and reason about risks — grounded in authoritative sources (University of Helsinki, Google, Stanford HAI, NIST). This is also the strongest marketing hook for the marketplace launch.

## 2. Goals

- Ship a beginner course that builds **conceptual AI literacy** with zero coding required.
- Every substantive claim is backed by a cited, link-checked authoritative source.
- Learners leave able to **use AI tools effectively and critically** and to reason about limitations and ethics.
- Assessments verify understanding (auto-scored quizzes) and application (a prompt lab + a use-case analysis).
- Free, `marketplace_listed`, and provisioned reproducibly via MC0.

## 3. Non-Goals

- **No programming / no math derivations.** (That is a future "Machine Learning with Python" course; MC2 teaches Python separately.)
- Not a tool tutorial for any single vendor product; vendor-neutral.
- Not a deep-learning/research course (no backprop math, no training your own models).
- No certification/accredited credential in this story (see MC0 §18).
- Not current-events reporting; content is evergreen with periodic reference refresh.

## 4. Personas & User Stories

- **As a curious professional**, I want to understand what AI can and cannot do so that I can adopt tools wisely at work.
- **As a student (HE/K12 9+)**, I want a trustworthy primer so that I can talk about AI accurately and use it responsibly for study.
- **As a self-learner**, I want plain-language explanations with links to authoritative sources so that I can go deeper on my own.
- **As an instructor**, I want a vetted free AI-literacy course to recommend so that I don't have to build one.
- **As a skeptic**, I want an honest treatment of bias, hallucination, and risk so that I can trust the material.

## 5. Functional Requirements

- **FR-1.** The course MUST contain 7 modules, each with ≥3 content pages, a module summary, and one auto-scored knowledge-check quiz (5–8 questions).
- **FR-2.** The course MUST define measurable learning outcomes (§ below) surfaced on the syllabus and course landing page.
- **FR-3.** Every content page making a factual or definitional claim MUST cite at least one source from §19; each external link MUST pass the MC0 link-checker.
- **FR-4.** The course MUST include one applied assignment ("Prompt Lab", M5) and one capstone ("Responsible AI Use-Case Analysis", M7), both auto-graded (`completion_full` or `grader_agent`).
- **FR-5.** The course MUST be provisioned with `price_cents=0`, `published=true`, `marketplace_listed=true`, `difficulty_level='beginner'`, `catalog_category='Technology'`, `catalog_slug='ai-essentials'`.
- **FR-6.** Terminology MUST be introduced before use and collected in a glossary page; definitions MUST match the cited authoritative source.
- **FR-7.** The responsible-AI module MUST frame trustworthiness using the **NIST AI RMF** characteristics (valid & reliable, safe, secure & resilient, accountable & transparent, explainable & interpretable, privacy-enhanced, fair with harmful bias managed).
- **FR-8.** Content SHOULD be written at roughly a Grade 9–11 reading level; SHOULD avoid unexplained jargon; MAY include analogies and diagrams (with alt text).

## 6. Non-Functional Requirements

- **Accuracy** — SME sign-off checklist; claims traceable to §19; no vendor benchmarks stated as timeless fact (framed as "as of" with a source).
- **Accessibility** — WCAG 2.1 AA: alt text on every diagram, ordered headings, descriptive links, captions on any embedded video, no color-only meaning.
- **Privacy & Compliance** — no PII; assignments warn learners not to paste sensitive data into third-party AI tools; original prose (no copied text beyond attributed fair-use quotes).
- **Internationalization** — EN at GA; ES fast-follow (matches intro course); numbers/dates locale-safe.
- **Reliability/Observability** — provisioned + validated via MC0; link-check in CI.
- **Maintainability** — dated claims isolated to a "Further reading / state of AI" page so refreshes are localized.

## 7. Acceptance Criteria

- **AC-1.** *Given* MC0 is deployed, *When* `provision-marketplace-courses --only ai-essentials` runs, *Then* the course appears in the storefront as **Free**, `beginner`, category **Technology**, with 7 modules and a syllabus listing the outcomes.
- **AC-2.** *Given* a learner claims the course, *When* they open Module 1, *Then* pages render with working external links and a knowledge check that auto-scores on submit.
- **AC-3.** *Given* the M5 Prompt Lab, *When* a learner submits a reflection meeting the rubric, *Then* it is auto-graded full credit (`grader_agent`) with optional AI feedback.
- **AC-4.** *Given* the validator/link-checker, *When* CI runs, *Then* all quiz answers resolve and all §19 links return 2xx/3xx.
- **AC-5.** *Given* the capstone, *When* a learner submits a use-case analysis naming one limitation and one mitigation, *Then* it is accepted and recorded in the gradebook.

## 8. Learning Outcomes

By the end of AI Essentials, a learner can:

1. **Define and distinguish** artificial intelligence, machine learning, deep learning, and generative AI, and give an everyday example of each.
2. **Explain in plain language** how a machine-learning model "learns" from data (training vs. inference) and how a large language model produces text (next-token prediction).
3. **Judge fit** — decide when AI is and is not appropriate for a task, and name at least three limitations (hallucination, bias, no ground-truth understanding, data/privacy constraints).
4. **Prompt and evaluate** — write a clear, iterated prompt and critically verify the output instead of trusting it blindly.
5. **Reason about responsibility** — describe the NIST AI RMF trustworthiness characteristics and apply a short responsible-use checklist, including human oversight and privacy.
6. **Recognize impact** — identify where AI appears in common tools and how it is changing everyday work.

## 9. Syllabus (authored `syllabus.json` sections)

- **Course overview** — What AI Essentials is, who it's for (no coding, ~4–6h, self-paced), how it's graded.
- **What you'll learn** — the six outcomes above.
- **Module guide** — the table in §10.
- **Grading & completion** — knowledge checks auto-scored; Prompt Lab + Capstone auto-graded; complete = finish the final knowledge check + capstone.
- **How to use AI tools safely in this course** — don't paste personal/confidential data; verify claims; cite sources.
- **Getting help & going deeper** — the §19 resource list.

## 10. Module Outline

| # | Module | Key pages | Knowledge check focus | Primary sources |
|---|---|---|---|---|
| 1 | **What Is AI?** | AI vs. ML vs. DL vs. GenAI; narrow vs. general; a short honest history & "the AI effect" | Distinguish the four terms; narrow vs general | Elements of AI; Google MLCC; AI for Everyone |
| 2 | **How Machines Learn** | Data & features; training vs. inference; supervised / unsupervised / reinforcement; "a model is a pattern learned from data"; overfitting intuition | Match learning types to examples; training vs inference | Google MLCC; Elements of AI |
| 3 | **Neural Networks & Deep Learning (gently)** | Neuron→layer→network analogy; why "deep"; what made it work (data + compute); what it's good/bad at | Why deep learning took off; strengths/limits | Google MLCC; Elements of AI |
| 4 | **Generative AI & LLMs** | Tokens & next-word prediction; what training on text does; images/audio; "understanding" vs prediction | How an LLM generates text; what it does *not* do | Generative AI for Everyone; "Attention Is All You Need"; Google MLCC (LLM unit) |
| 5 | **Prompting & Working with AI Tools** | Anatomy of a good prompt; iteration; verifying outputs; hallucination; when not to use AI | Identify a strong vs weak prompt; spot a hallucination | Generative AI for Everyone; AI for Everyone |
| 6 | **Limitations, Bias & Risks** | Where bias comes from; hallucination; privacy & data; a plain-language look at prompt injection/misuse; environmental & cost notes | Sources of bias; privacy do/don'ts | NIST AI RMF; Stanford HAI AI Index |
| 7 | **Responsible & Practical AI** | NIST trustworthiness characteristics; human oversight; AI in careers; how to keep learning; **Capstone** | Map a scenario to trustworthiness characteristics | NIST AI RMF; Stanford HAI AI Index; AI for Everyone |

## 11. Assessments (detail)

**Per-module knowledge checks** — 5–8 auto-scored questions (`multiple_choice`, `true_false`), instant feedback, unlimited attempts, low stakes. Example items:

- *M1 (multiple_choice):* "Which statement best distinguishes machine learning from traditional programming?" → correct: "The system learns rules from examples/data instead of being given the rules explicitly." Distractors: "It always uses a neural network"; "It never makes mistakes"; "It requires no data."
- *M4 (true_false):* "A large language model chooses each next word by predicting likely continuations from patterns in its training data, not by looking up verified facts." → **True.**
- *M6 (multiple_choice):* "A hiring model rates candidates from one neighborhood lower. The most likely root cause is:" → correct: "Bias present in the historical training data." Distractors include "the model is broken"; "randomness"; "too little compute."

**M5 — Prompt Lab (assignment, `grader_agent`, 10 pts).** Learner picks a real task, writes an initial prompt, iterates at least once, and submits: the two prompts, what changed, and a 3–5 sentence evaluation of the output's accuracy and one thing they verified. Rubric rewards iteration + a verification step. Full credit for a good-faith, on-rubric submission; optional AI feedback when enabled.

**M7 — Capstone: Responsible AI Use-Case Analysis (assignment, `grader_agent`, 15 pts).** Learner chooses a realistic scenario (e.g., "use AI to draft customer emails"), and writes: (a) whether AI is appropriate and why, (b) one NIST trustworthiness characteristic at risk, (c) one concrete mitigation and the human-oversight step. Demonstrates outcomes 3 & 5.

## 12. Data Model

No new schema — uses MC0's harness and the existing structure/quiz/syllabus tables. Course row: `catalog_slug='ai-essentials'`, `catalog_category='Technology'`, `difficulty_level='beginner'`, `catalog_language='en'`, `price_cents=0`, `is_public=true` (recommended), `marketplace_listed=true`.

## 13. API Surface

None new. Served by existing course, storefront (MKT3), claim (MKT4), and quiz endpoints.

## 14. UI / UX

Renders through existing storefront card, course landing, syllabus, module reader, and quiz player. Card copy: *"AI Essentials — Understand AI, use it well, and reason about its limits. No coding. Free."* Hero image: embedded abstract network banner (`assets/ai-essentials-banner.jpg`), provisioned into course-files and wired to `hero_image_url` (same pattern as the intro course). "Official" badge if MC0 §18 resolves.

## 15. AI / ML Considerations

`grader_agent` grades M5/M7 reflections (reuses intro-course capstone path). Content pages that show AI output MUST label it as illustrative and note it may be wrong. No live model calls in content.

## 16. Integration Points

MC0 harness; `coursemodulequiz` for quizzes; `grader_agent` grading; CI link-check over `content/en/ai-essentials/**`.

## 17. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| AI facts/benchmarks age quickly | H | M | Isolate dated claims to one "state of AI" page citing the annually-updated Stanford HAI AI Index; scheduled review |
| Oversimplification misleads | M | M | SME review; link to authoritative deep-dives; hedge appropriately |
| Vendor bias / looks like an ad | M | M | Vendor-neutral examples; no product endorsements |
| Link rot | M | M | MC0 CI link-checker; prefer stable domains (helsinki, google, nist, stanford) |

## 18. Rollout Plan

Provision to staging → internal review + a11y (axe) pass → SME accuracy sign-off → GA (list in storefront; set `is_public` for SEO). Rollback = unlist (`marketplace_listed=false`). First payload to validate the MC0 pipeline end-to-end.

## 19. References (verified, link-checked)

Authoritative, free (or free-to-audit) sources backing the content. All URLs confirmed reachable during planning.

- **Elements of AI** — University of Helsinki & MinnaLearn, free intro AI course — https://www.elementsofai.com/ (core reference for Modules 1–3)
- **Google — Machine Learning Crash Course** — free, covers ML fundamentals through LLMs & fairness — https://developers.google.com/machine-learning/crash-course
- **Andrew Ng — "AI for Everyone"** (DeepLearning.AI / Coursera, free to audit; non-technical) — https://www.coursera.org/learn/ai-for-everyone · course page https://www.deeplearning.ai/courses/ai-for-everyone
- **Andrew Ng — "Generative AI for Everyone"** (DeepLearning.AI / Coursera, free to audit) — https://www.coursera.org/learn/generative-ai-for-everyone
- **NIST AI Risk Management Framework (AI RMF 1.0)** — trustworthiness characteristics used in Modules 6–7 — https://www.nist.gov/itl/ai-risk-management-framework
- **Stanford HAI — AI Index Report** — annually updated, citable statistics on the state of AI — https://hai.stanford.edu/ai-index
- **Vaswani et al., "Attention Is All You Need" (2017)** — the transformer paper underpinning modern LLMs (referenced, not required reading) — https://arxiv.org/abs/1706.03762
- Internal: [MC0](MC0-authoring-provisioning-foundation.md) (harness), `../marketplace/` (commerce), `server/internal/service/introcourse` (pattern).
