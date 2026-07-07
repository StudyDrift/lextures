# Lextures CLI — Feature Parity Plan

Implementation plans to close the gap between the **Lextures HTTP API** (~843 routes across
~120 route-registration groups in `server/internal/httpserver`) and the **`lextures` CLI**
(`clients/cli`), which today exposes ~9 command groups (roughly 10 % of the API surface).

Each plan follows [`../_TEMPLATE.md`](../_TEMPLATE.md). One plan per CLI **command group**
(the `lextures <noun>` surface). Scope decision: **full API parity** — every server capability
that can be driven from a terminal gets a command group, including student-facing flows.

## Conventions

- File naming: `C{NN}-{kebab-slug}.md` (e.g. `C11-enrollments-sections.md`).
- CLI shape: `lextures <noun> <verb> [flags]`, matching the existing Cobra layout in
  `clients/cli/cmd/*.go`. Global flags: `--server`, `--api-key`, `--profile`, `--config`,
  `--json`. Exit codes: `0` success, `1` bad input/usage, `2` API/server error.
- Every command MUST support `--json` for scripting and a human tabwriter table by default.
- A plan is "ready" when every template section is filled (no `…` placeholders).

## Severity legend (CLI-adoption lens)

- **BLOCKER** — automation/admin workflows are impossible without it; the primary reason to
  adopt a CLI (bulk provisioning, roster sync, CI/CD content, reporting exports).
- **MAJOR** — significant admin/instructor automation gap.
- **MINOR** — parity / power-user nicety.

## Status legend

- **MISSING** — no command exists.
- **PARTIAL** — command group exists but is missing verbs/flags.
- **THIN** — exists but too shallow to be useful for automation.

---

## Current CLI surface (baseline)

| Group | Verbs today | Plan that expands it |
|---|---|---|
| `auth` | login, logout, status | C39 |
| `courses` | list, get, create, delete | [C01](C01-courses.md) |
| `assignments` | list, get, create, submit | [C03](C03-assignments.md) |
| `grades` | list, update, export | [C06](C06-gradebook-final-grades.md) |
| `users` | list, get, create, enroll, update, suspend, reactivate, delete, import, search | [C15](../completed/cli/C15-people-provisioning.md) |
| `imports` | submit, status, list | [C15](../completed/cli/C15-people-provisioning.md) |
| `custom-fields` | list, create, update, delete | [C15](../completed/cli/C15-people-provisioning.md) |
| `people` | list | [C15](../completed/cli/C15-people-provisioning.md) |
| `enrollments` | list, export, import, add, remove, set-state, self-enroll | [C11](../completed/cli/C11-enrollments-sections.md) |
| `sections` | list, create, update, delete, move, cross-list | [C11](../completed/cli/C11-enrollments-sections.md) |
| `orgs` | list, get, create, update, archive, settings, branding, role-grants, parent-links, cross-list-groups | [C14](../completed/cli/C14-org-administration.md) |
| `org-units` | list, create, update, delete, move | [C14](../completed/cli/C14-org-administration.md) |
| `terms` | list, create, update, delete | [C14](../completed/cli/C14-org-administration.md) |
| `groups` | list, create, delete, add, remove, members | [C13](../completed/cli/C13-groups-collaboration.md) |
| `discussions` | list, export, post, lock | [C13](../completed/cli/C13-groups-collaboration.md) |
| `collab-docs` | list, export | [C13](../completed/cli/C13-groups-collaboration.md) |
| `whiteboards` | list, export | [C13](../completed/cli/C13-groups-collaboration.md) |
| `files` | list, mkdir, upload, download, rename, move, delete | [C21](C21-platform-settings.md) |
| `feed` | channels, post, recent | [C34](C34-messaging-broadcasts.md) |
| `questions` | list, create, import | [C04](C04-quizzes-question-banks.md) |
| `attendance` | list, record, import, export, summary | [C12](../completed/cli/C12-attendance-behavior.md) |
| `behavior` | list, export, award | [C12](../completed/cli/C12-attendance-behavior.md) |
| `seat-time` | report | [C12](../completed/cli/C12-attendance-behavior.md) |
| `hall-pass` | list, issue, return | [C12](../completed/cli/C12-attendance-behavior.md) |

---

## Plans

### A. Course & content authoring

| ID | Plan | Severity | Status | Effort |
|---|---|---|---|---|
| C01 | [Courses (expand)](C01-courses.md) | BLOCKER | PARTIAL | M |
| C02 | [Modules & course structure](C02-modules-course-structure.md) | BLOCKER | MISSING | M |
| C03 | [Assignments (expand)](C03-assignments.md) | MAJOR | PARTIAL | M |
| C04 | [Quizzes & question banks](C04-quizzes-question-banks.md) | BLOCKER | PARTIAL | L |
| C05 | [Content extras (pages, glossary, H5P, SCORM, tools)](../completed/cli/C05-content-extras.md) | MINOR | COMPLETE | M |

### B. Assessment & grading

| ID | Plan | Severity | Status | Effort |
|---|---|---|---|---|
| C06 | [Gradebook & final grades (expand)](C06-gradebook-final-grades.md) | BLOCKER | PARTIAL | M |
| C07 | [Outcomes, standards & SBG report cards](C07-outcomes-standards-sbg.md) | MAJOR | MISSING | M |
| C08 | [Peer review, evaluations & surveys](../completed/cli/C08-peer-review-evaluations-surveys.md) | MINOR | COMPLETE | M |
| C09 | [AI grading agents](../completed/cli/C09-ai-grading-agents.md) | MAJOR | COMPLETE | M |
| C10 | [Plagiarism & originality](../completed/cli/C10-plagiarism-originality.md) | MAJOR | COMPLETE | S |

### C. Roster & classroom

| ID | Plan | Severity | Status | Effort |
|---|---|---|---|---|
| C11 | [Enrollments & sections](../completed/cli/C11-enrollments-sections.md) | BLOCKER | COMPLETE | M |
| C12 | [Attendance, behavior & seat-time](../completed/cli/C12-attendance-behavior.md) | MAJOR | COMPLETE | M |
| C13 | [Groups & collaboration](../completed/cli/C13-groups-collaboration.md) | MINOR | COMPLETE | S |

### D. Admin & governance

| ID | Plan | Severity | Status | Effort |
|---|---|---|---|---|
| C14 | [Org & org-unit administration](../completed/cli/C14-org-administration.md) | MAJOR | COMPLETE | M |
| C15 | [People, provisioning & bulk import](../completed/cli/C15-people-provisioning.md) | BLOCKER | COMPLETE | L |
| C16 | [Roles & permissions (RBAC)](C16-roles-permissions.md) | BLOCKER | MISSING | M |
| C17 | [Licenses, entitlements & marketplace](C17-licenses-entitlements.md) | MAJOR | MISSING | M |
| C18 | [Jobs, scheduler, quarantine & backups](C18-jobs-scheduler-backups.md) | MAJOR | MISSING | M |
| C19 | [Audit log, impersonation & admin search](C19-audit-impersonation-search.md) | MAJOR | MISSING | S |
| C20 | [Email templates & banners](C20-email-templates-banners.md) | MINOR | MISSING | S |
| C21 | [Platform settings & configuration](C21-platform-settings.md) | MAJOR | MISSING | M |

### E. Integrations & interoperability

| ID | Plan | Severity | Status | Effort |
|---|---|---|---|---|
| C22 | [SIS, SCIM & OneRoster](C22-sis-scim-oneroster.md) | BLOCKER | MISSING | L |
| C23 | [LTI & developer keys](C23-lti-developer-keys.md) | MAJOR | MISSING | M |
| C24 | [Canvas & content import](C24-canvas-content-import.md) | MAJOR | MISSING | M |
| C25 | [Integrations, cloud providers, webhooks & bots](C25-integrations-webhooks-bots.md) | MAJOR | PARTIAL | M |
| C26 | [xAPI / LRS / SCORM runtime & engagement](C26-xapi-lrs-engagement.md) | MINOR | MISSING | S |

### F. Reporting & insights

| ID | Plan | Severity | Status | Effort |
|---|---|---|---|---|
| C27 | [Reports & exports](C27-reports-exports.md) | BLOCKER | MISSING | M |
| C28 | [Insights, at-risk & classroom signals](C28-insights-at-risk.md) | MAJOR | MISSING | M |

### G. Compliance & trust

| ID | Plan | Severity | Status | Effort |
|---|---|---|---|---|
| C29 | [Compliance, privacy & trust](C29-compliance-privacy.md) | MAJOR | MISSING | L |

### H. Commerce

| ID | Plan | Severity | Status | Effort |
|---|---|---|---|---|
| C30 | [Billing, payments, tax & revenue](C30-billing-payments-tax.md) | MAJOR | MISSING | M |

### I. Academic records

| ID | Plan | Severity | Status | Effort |
|---|---|---|---|---|
| C31 | [Credentials, transcripts, advising & degree progress](C31-credentials-transcripts-advising.md) | MAJOR | MISSING | M |

### J. Catalog & materials

| ID | Plan | Severity | Status | Effort |
|---|---|---|---|---|
| C32 | [Catalog, library, OER & bookstore](C32-catalog-library-oer.md) | MINOR | MISSING | M |

### K. Accessibility & localization

| ID | Plan | Severity | Status | Effort |
|---|---|---|---|---|
| C33 | [Accessibility, media & localization](C33-accessibility-media-localization.md) | MINOR | MISSING | M |

### L. Communication

| ID | Plan | Severity | Status | Effort |
|---|---|---|---|---|
| C34 | [Messaging, broadcasts & notifications](C34-messaging-broadcasts.md) | MAJOR | PARTIAL | M |
| C35 | [Meetings, office hours & conferences](C35-meetings-office-hours.md) | MINOR | MISSING | S |

### M. Student experience (full-parity)

| ID | Plan | Severity | Status | Effort |
|---|---|---|---|---|
| C36 | [AI tutor, study buddy & diagnostics](C36-tutor-study-buddy.md) | MINOR | MISSING | M |
| C37 | [Student workspace, notebooks, goals & gamification](C37-student-workspace.md) | MINOR | MISSING | M |
| C38 | [Portfolios & eportfolio](C38-portfolios.md) | MINOR | MISSING | S |
| C39 | [Profile, account, security & personas (me, parent)](C39-profile-account-personas.md) | MAJOR | PARTIAL | M |

### N. CLI framework

| ID | Plan | Severity | Status | Effort |
|---|---|---|---|---|
| C40 | [CLI framework & ergonomics](C40-cli-framework.md) | BLOCKER | THIN | M |

---

## Recommended sequencing

1. **Foundation first** — C40 (framework: output/pagination/wait/bulk) unblocks quality of every
   other command.
2. **Automation core** — C15, C11, C22, C16, C01, C06, C27 (provisioning, roster, RBAC, courses,
   grades, reports): the workflows admins buy a CLI for.
3. **Content & assessment** — C02, C04, C03, C07, C09, C10.
4. **Integrations & governance** — C23, C24, C25, C14, C17, C18, C19, C21, C29, C30.
5. **Everything else** — C05, C08, C12, C13, C20, C26, C28, C31–C39 as demand dictates.
