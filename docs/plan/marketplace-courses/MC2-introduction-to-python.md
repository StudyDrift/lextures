# MC2 — Introduction to Python (Course Content & Assessments)

> Implementation plan. Source: [docs/plan/marketplace-courses/README.md](README.md). A first-party, **free** marketplace course provisioned via the [MC0](../../completed/marketplace-courses/MC0-authoring-provisioning-foundation.md) harness.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | MC2 |
| **Section** | Marketplace Courses |
| **Severity** | MAJOR |
| **Markets** | SL (primary) · HE · K12 (grades 8+) |
| **Status (today)** | MISSING |
| **Estimated effort** | M (3–4w authoring + review) |
| **Owner (proposed)** | Content team + software-engineering SME reviewer |
| **Depends on** | MC0 |
| **Unblocks** | Future "Python for Data" / "Automate with Python" courses |
| **Course code (proposed)** | `INTRO-PYTHON` |
| **Catalog** | slug `introduction-to-python` · category `Programming` · difficulty `beginner` · language `en` · **price `$0`** |
| **Length** | 8 modules · ~8–12 hours · self-paced, hands-on |

---

## 1. Problem Statement

Python is the most common first programming language and a gateway skill for data, automation, and AI, but quality free introductions are scattered across many sites and vary wildly in accuracy and pacing. The marketplace needs a coherent, **free**, hands-on "learn to code from zero" course that takes an absolute beginner from "what is a program?" to writing a small working program of their own, grounded in the official Python documentation and respected free curricula (Python for Everybody, CS50P, Automate the Boring Stuff). It also seeds a future Python track.

## 2. Goals

- Take a **complete beginner** to writing and running real Python programs (variables → control flow → data structures → functions → files).
- Be genuinely **hands-on**: every module has code the learner runs; assessments include predict-the-output and write-the-code tasks.
- Ground syntax and semantics in the **official Python 3 documentation**; teach **PEP 8** basics.
- Provide a clear, low-friction way to run code (online interpreter or local install) so no setup blocks progress.
- Free, `marketplace_listed`, provisioned reproducibly via MC0.

## 3. Non-Goals

- Not object-oriented, decorators, async, web frameworks, data science, or ML — those are follow-on courses (§13 references a future track).
- Not language-agnostic CS theory (Big-O, data-structure proofs) beyond intuition.
- Not an IDE/tooling deep dive; we recommend one simple setup and move on.
- No autograded code *execution* on the platform in this story unless MC0's optional runner lands (see §18) — assessments use predict-the-output quizzes + submitted-code assignments run by the learner in a linked free environment.
- No certification/credential in this story.

## 4. Personas & User Stories

- **As an absolute beginner**, I want a from-zero path with runnable examples so that I can write my first program without prior experience.
- **As a career switcher**, I want a credible free Python foundation so that I can move on to data/automation courses.
- **As a student**, I want practice with instant-feedback quizzes and small projects so that I can build confidence.
- **As a self-learner**, I want links to the official docs and respected free courses so that I can go deeper.
- **As an instructor**, I want a vetted free intro to recommend so that students arrive with basics.

## 5. Functional Requirements

- **FR-1.** The course MUST contain 8 modules, each with runnable code examples, ≥3 content pages, a summary, and one auto-scored knowledge check (5–8 questions) including at least one **predict-the-output** item.
- **FR-2.** Module 1 MUST give learners a working way to run Python **before any syntax** — a recommended free online interpreter *and* local-install instructions from python.org.
- **FR-3.** The course MUST include ≥4 **write-the-code** assignments (one small program each) plus a capstone; assignments are submitted as code (text/file) and graded `completion_full` or `grader_agent`, with a self-check checklist and expected output.
- **FR-4.** All syntax/semantics claims MUST match the official Python 3 docs (docs.python.org); style guidance MUST follow **PEP 8**.
- **FR-5.** Code MUST target **Python 3.x**, run without third-party packages (standard library only), and be copy-paste runnable.
- **FR-6.** The course MUST be provisioned `price_cents=0`, `published=true`, `marketplace_listed=true`, `difficulty_level='beginner'`, `catalog_category='Programming'`, `catalog_slug='introduction-to-python'`.
- **FR-7.** Every code block MUST show expected output (or note "no output"); assignments MUST state expected behavior so learners can self-verify.
- **FR-8.** Content SHOULD build strictly incrementally (no forward references) and SHOULD flag common beginner errors (indentation, `=` vs `==`, off-by-one, type mismatches) with fixes.

## 6. Non-Functional Requirements

- **Accuracy** — SME review that every example runs on a current Python 3 and produces the stated output; claims cite docs.python.org.
- **Accessibility** — WCAG 2.1 AA: code blocks in real text (not images) so screen readers/copy work; alt text on diagrams; descriptive links; ordered headings.
- **Privacy & Compliance** — no PII; if recommending an online interpreter, warn not to paste secrets; original code/prose (no copied text beyond attributed fair-use).
- **Internationalization** — EN at GA; code/keywords stay English (language syntax); prose ES fast-follow.
- **Reliability/Observability** — provisioned + validated via MC0; link-check in CI.
- **Maintainability** — pin narrative to stable docs; a single "setup" page isolates install details that change across OS versions.

## 7. Acceptance Criteria

- **AC-1.** *Given* MC0 is deployed, *When* `provision-marketplace-courses --only introduction-to-python` runs, *Then* the course appears in the storefront as **Free**, `beginner`, category **Programming**, with 8 modules and a syllabus of outcomes.
- **AC-2.** *Given* Module 1, *When* a learner follows it, *Then* they can run a `print()` program via the linked online interpreter **and** know how to run a `.py` file locally.
- **AC-3.** *Given* a predict-the-output quiz item, *When* the learner submits, *Then* it auto-scores with an explanation.
- **AC-4.** *Given* a write-the-code assignment, *When* the learner submits code matching the expected behavior and self-check, *Then* it is graded full credit (`grader_agent`/`completion_full`).
- **AC-5.** *Given* CI, *When* the validator + link-checker run, *Then* every quiz answer resolves, every external link is reachable, and (if the optional runner exists) sample solutions execute to the stated output.

## 8. Learning Outcomes

By the end of Introduction to Python, a learner can:

1. **Run Python** via an interactive interpreter (REPL) and by executing a `.py` file, using an online tool or a local install.
2. **Use core types & I/O** — variables; `int`, `float`, `str`, `bool`; type conversion; `input()`; and formatted output with f-strings.
3. **Control flow** — implement logic with `if/elif/else`, boolean/comparison operators, and `while`/`for` loops (with `range`, `break`, `continue`).
4. **Collections** — store and manipulate data with lists, tuples, dictionaries, and sets, including indexing, slicing, and iteration.
5. **Functions & modules** — write reusable functions with parameters/return values and import from the standard library (`math`, `random`, `statistics`).
6. **Data & files** — use string methods, read/write text files, and handle errors with `try`/`except`.
7. **Style & debugging** — apply PEP 8 basics, read a small program, and diagnose common beginner errors.
8. **Build** — design and write a small, complete program from a specification (capstone).

## 9. Syllabus (authored `syllabus.json` sections)

- **Course overview** — from-zero, hands-on, ~8–12h, self-paced; how to run code; how it's graded.
- **What you'll learn** — the eight outcomes above.
- **Before you start** — pick your environment (online interpreter *or* local Python from python.org); nothing else to buy.
- **Module guide** — the table in §10.
- **Grading & completion** — knowledge checks auto-scored; 4+ coding assignments + capstone; complete = pass the final knowledge check + submit the capstone.
- **Getting help & going deeper** — official docs + the free courses in §19.

## 10. Module Outline

| # | Module | Core content | Assessment focus | Primary sources |
|---|---|---|---|---|
| 1 | **Getting Started** | What a program is; run code online; install Python; the REPL; `print()`; running a `.py` file; comments | Run first program; identify REPL vs script | python.org; docs.python.org tutorial §1–2 |
| 2 | **Variables, Types & Input** | Variables & naming; `int/float/str/bool`; conversion; `input()`; f-strings | Predict output; fix a type error | docs.python.org tutorial §3; PY4E |
| 3 | **Operators & Expressions** | Arithmetic, comparison, boolean; precedence; string concatenation/repetition; `len()` | Evaluate expressions; precedence | docs.python.org; PY4E |
| 4 | **Making Decisions & Looping** | `if/elif/else`; truthiness; `while`; `for` + `range`; `break`/`continue`; nesting | Trace a loop; write a conditional; **assignment** | docs.python.org tutorial §4; CS50P |
| 5 | **Collections: Lists, Tuples, Dicts, Sets** | Create/index/slice; iterate; common methods; when to use which; mutability | Choose the right structure; **assignment** (list/dict) | docs.python.org tutorial §5; PY4E |
| 6 | **Functions & Modules** | `def`, params, `return`, defaults, scope; docstrings; `import`; `math`/`random`/`statistics` | Write a function; import & use a module; **assignment** | docs.python.org tutorial §4.7–6; PY4E |
| 7 | **Strings, Files & Errors** | String methods; reading/writing text files; `with`; `try/except`; common exceptions | Parse a file; handle an error; **assignment** | docs.python.org tutorial §7,§8; Automate the Boring Stuff |
| 8 | **Putting It Together** | Program design from a spec; PEP 8 basics; debugging; **capstone**; where to go next | Build a complete program | PEP 8; CS50P; PY4E |

## 11. Assessments (detail)

**Per-module knowledge checks** — 5–8 auto-scored questions mixing concept MC/true-false with **predict-the-output** and **spot-the-bug**. Examples:

- *M2 (predict-the-output, multiple_choice):* "What does `print(3 + 4 * 2)` display?" → **11** (precedence). Distractors: `14`, `10`, `24`.
- *M2 (spot-the-bug, multiple_choice):* "`age = input('Age: ')` then `if age > 18:` raises `TypeError`. The fix is:" → correct: "Convert with `int(age)` before comparing." Distractors: "use `==`"; "remove the colon"; "rename `age`."
- *M4 (true_false):* "In Python, `for i in range(3):` iterates with `i` taking the values 0, 1, 2." → **True.**
- *M5 (multiple_choice):* "You need to look up a value by a unique key. The best built-in structure is a:" → **dictionary.** Distractors: list, tuple, set.

**Coding assignments (`grader_agent`/`completion_full`, ~10 pts each), 4 minimum:**

- **M4 — FizzBuzz-style:** print 1–20, but "even"/"odd" (or classic FizzBuzz) — exercises loops + conditionals. Expected output provided.
- **M5 — Word/letter counter:** given a string, build a frequency dictionary and print the most common item — lists/dicts + iteration.
- **M6 — Temperature/units function:** write `celsius_to_fahrenheit(c)` with a docstring and use it in a loop over a list — functions + modules.
- **M7 — File summarizer:** read a small text file, count lines/words, handle a missing-file error with `try/except`.

**M8 — Capstone (`grader_agent`, 20 pts):** build a small complete program from a spec — e.g., a **number-guessing game** (random target, loop, hints, input validation) *or* a **simple text menu tool**. Learner submits code + a note on what they'd improve. Demonstrates outcomes 1–8. Self-check checklist and sample expected interaction provided.

## 12. Data Model

No new schema — MC0 harness + existing structure/quiz/syllabus tables. Course row: `catalog_slug='introduction-to-python'`, `catalog_category='Programming'`, `difficulty_level='beginner'`, `catalog_language='en'`, `price_cents=0`, `is_public=true` (recommended), `marketplace_listed=true`.

## 13. API Surface

None new. Served by existing course/storefront/claim/quiz endpoints. Follow-on courses ("Automate with Python", "Python for Data") would reuse the same pattern.

## 14. UI / UX

Existing storefront card, syllabus, module reader (with fenced code blocks via the course Markdown theme), and quiz player. Card copy: *"Introduction to Python — Learn to code from zero. Hands-on, no experience needed. Free."* If MC0's optional in-browser runner lands, embed "Run this" affordances; otherwise link the recommended online interpreter. Hero image: friendly, abstract code motif with alt text.

## 15. AI / ML Considerations

`grader_agent` grades submitted code/reflections for good-faith completeness against the rubric/expected behavior (not full execution) — reuses the intro-course grading path. Optional AI feedback on style/logic when enabled. **In-browser code execution/autograding is out of scope here** and tracked as an MC0 enhancement (§18).

## 16. Integration Points

MC0 harness; `coursemodulequiz` (incl. `code`/predict-the-output items); `grader_agent`; CI link-check + (if runner exists) example-execution over `content/en/introduction-to-python/**`.

## 17. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Setup friction blocks beginners | M | H | Module 1 offers a zero-install online interpreter first; local install optional |
| No code execution = weaker feedback | M | M | Predict-the-output + spot-the-bug auto-quizzes; expected output + self-check on every assignment; pursue MC0 runner (§18) |
| Examples break on a future Python version | L | M | Standard-library only; SME re-runs on current 3.x; pin narrative to docs.python.org |
| Recommended online tool changes/disappears | M | M | Recommend by capability, list 2 alternatives, prefer official python.org "Try/Shell" where available; link-check |
| Copying course text from other tutorials | M | H | Original prose + original exercises; cite, don't copy |

## 18. Rollout Plan

Provision to staging → SME runs every example on current Python 3 → a11y (axe) pass (code blocks are real text) → GA (list; set `is_public`). Rollback = unlist. **Open enhancement:** an in-browser Python runner + autograder in MC0 would upgrade assignments from submit-and-self-check to executed-and-graded; deferred to a follow-up.

## 19. References (verified, link-checked)

Authoritative, free sources backing the content. All URLs confirmed reachable during planning.

- **The Python Tutorial** (official, Python Software Foundation) — canonical source for every syntax/semantics claim — https://docs.python.org/3/tutorial/index.html
- **Python.org** — downloads/install + "Getting Started" — https://www.python.org/
- **PEP 8 — Style Guide for Python Code** (official) — style basics in Module 8 — https://peps.python.org/pep-0008/
- **Python for Everybody (PY4E)** — Dr. Charles Severance, University of Michigan; free book, videos, and autograded exercises — https://www.py4e.com/
- **CS50's Introduction to Programming with Python (CS50P)** — Harvard, free via OpenCourseWare — https://cs50.harvard.edu/python/
- **Automate the Boring Stuff with Python** — Al Sweigart; full book free to read online (CC-licensed) — https://automatetheboringstuff.com/
- **Python Standard Library reference** (official) — for `math`, `random`, `statistics`, file I/O — https://docs.python.org/3/library/index.html
- Internal: [MC0](../../completed/marketplace-courses/MC0-authoring-provisioning-foundation.md) (harness), `../marketplace/README.md` (commerce), `server/internal/service/introcourse` (pattern).
