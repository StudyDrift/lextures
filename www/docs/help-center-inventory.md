# Help center inventory

This inventory maps every published help article to the shipped product area used to verify it. A code reference is a review starting point, not a claim that one package alone implements the feature. Re-run `node scripts/generate-help-center.mjs` after changing the launch inventory.

| Article | Category | Tier | Capability reference | Status | Verified |
|---|---|---:|---|---|---|
| [Creating a new course](/docs/getting-started/creating-a-new-course) | getting-started | 2–3 | `clients/web/src/components/onboarding` | Published | 2026-08-11 |
| [Finding your course](/docs/getting-started/finding-your-course) | getting-started | 2–3 | `clients/web/src/components/onboarding` | Published | 2026-08-11 |
| [Navigating the course interface](/docs/getting-started/navigating-the-course-interface) | getting-started | 2–3 | `clients/web/src/components/onboarding` | Published | 2026-08-11 |
| [Understanding account and course roles](/docs/getting-started/understanding-lextures-roles) | getting-started | 2–3 | `clients/web/src/components/onboarding` | Published | 2026-08-11 |
| [Course checklist: what ready to launch means](/docs/courses/course-readiness-checklist) | courses | 1 | `server/internal/models/course` | Published | 2026-08-11 |
| [Building modules and content pages](/docs/courses/building-modules-and-pages) | courses | 2–3 | `server/internal/models/course` | Published | 2026-08-11 |
| [Uploading and managing course files](/docs/courses/managing-course-files) | courses | 2–3 | `server/internal/models/course` | Published | 2026-08-11 |
| [Editing the course syllabus](/docs/courses/editing-the-course-syllabus) | courses | 2–3 | `server/internal/models/course` | Published | 2026-08-11 |
| [How question difficulty and IRT calibration work](/docs/assessment/question-difficulty-irt-calibration) | assessment | 1 | `server/internal/models/questionbank` | Published | 2026-08-11 |
| [Creating and hosting an interactive quiz](/docs/assessment/creating-interactive-quizzes) | assessment | 1 | `server/internal/models/questionbank` | Published | 2026-08-11 |
| [Academic integrity settings and what they do](/docs/assessment/academic-integrity-settings) | assessment | 1 | `server/internal/models/questionbank` | Published | 2026-08-11 |
| [Building and reusing question banks](/docs/assessment/building-question-banks) | assessment | 2–3 | `server/internal/models/questionbank` | Published | 2026-08-11 |
| [Building a rubric that grades consistently](/docs/grading/building-consistent-rubrics) | grading | 1 | `server/internal/gradecurve` | Published | 2026-08-11 |
| [Curving and scaling grades](/docs/grading/curving-and-scaling-grades) | grading | 1 | `server/internal/gradecurve` | Published | 2026-08-11 |
| [Running a peer review assignment](/docs/grading/running-peer-review) | grading | 1 | `server/internal/gradecurve` | Published | 2026-08-11 |
| [Using what-if grades](/docs/grading/using-what-if-grades) | grading | 2–3 | `server/internal/gradecurve` | Published | 2026-08-11 |
| [How adaptive content decides what a learner sees](/docs/adaptive/how-adaptive-content-decides) | adaptive | 1 | `server/internal/models/adaptivecontent` | Published | 2026-08-11 |
| [Setting up spaced review for a course](/docs/adaptive/setting-up-spaced-review) | adaptive | 1 | `server/internal/models/adaptivecontent` | Published | 2026-08-11 |
| [Reading the learner model](/docs/adaptive/reading-the-learner-model) | adaptive | 2–3 | `server/internal/models/adaptivecontent` | Published | 2026-08-11 |
| [Using misconception signals](/docs/adaptive/using-misconception-signals) | adaptive | 2–3 | `server/internal/models/adaptivecontent` | Published | 2026-08-11 |
| [Aligning assignments to standards and outcomes](/docs/outcomes/aligning-assignments-to-standards) | outcomes | 1 | `server/internal/models/courseoutcomesapi` | Published | 2026-08-11 |
| [Reading the mastery report](/docs/outcomes/reading-the-mastery-report) | outcomes | 1 | `server/internal/models/courseoutcomesapi` | Published | 2026-08-11 |
| [Creating course outcomes](/docs/outcomes/creating-course-outcomes) | outcomes | 2–3 | `server/internal/models/courseoutcomesapi` | Published | 2026-08-11 |
| [Exporting outcome and mastery results](/docs/outcomes/exporting-outcome-results) | outcomes | 2–3 | `server/internal/models/courseoutcomesapi` | Published | 2026-08-11 |
| [How roster sync from your SIS works](/docs/enrollment/sis-roster-sync) | enrollment | 1 | `server/internal/provisioning/oneroster` | Published | 2026-08-11 |
| [Inviting people to a course](/docs/enrollment/inviting-people-to-a-course) | enrollment | 2–3 | `server/internal/provisioning/oneroster` | Published | 2026-08-11 |
| [Managing course sections](/docs/enrollment/managing-sections) | enrollment | 2–3 | `server/internal/provisioning/oneroster` | Published | 2026-08-11 |
| [Using a test learner safely](/docs/enrollment/using-a-test-student) | enrollment | 2–3 | `server/internal/provisioning/oneroster` | Published | 2026-08-11 |
| [Setting up SAML/OIDC single sign-on](/docs/accounts/setting-up-saml-oidc-sso) | accounts | 1 | `server/internal/oidc` | Published | 2026-08-11 |
| [Setting up multifactor authentication](/docs/accounts/setting-up-mfa) | accounts | 2–3 | `server/internal/oidc` | Published | 2026-08-11 |
| [Reviewing and ending active sessions](/docs/accounts/managing-active-sessions) | accounts | 2–3 | `server/internal/oidc` | Published | 2026-08-11 |
| [Resetting your password](/docs/accounts/resetting-your-password) | accounts | 2–3 | `server/internal/oidc` | Published | 2026-08-11 |
| [How accommodations are applied automatically](/docs/accessibility/automatic-accommodations) | accessibility | 1 | `clients/web/src/components/a11y` | Published | 2026-08-11 |
| [Which WCAG 2.2 AA features Lextures supports](/docs/accessibility/wcag-22-aa-features) | accessibility | 1 | `clients/web/src/components/a11y` | Published | 2026-08-11 |
| [Setting learner accommodations](/docs/accessibility/setting-learner-accommodations) | accessibility | 2–3 | `clients/web/src/components/a11y` | Published | 2026-08-11 |
| [Authoring accessible course content](/docs/accessibility/accessible-content-authoring) | accessibility | 2–3 | `clients/web/src/components/a11y` | Published | 2026-08-11 |
| [Setting up the parent portal](/docs/parents/setting-up-parent-portal) | parents | 1 | `clients/web/src/components/learner-profile` | Published | 2026-08-11 |
| [Pairing a parent to a student account](/docs/parents/pairing-parent-and-student) | parents | 1 | `clients/web/src/components/learner-profile` | Published | 2026-08-11 |
| [Following multiple children](/docs/parents/following-multiple-children) | parents | 2–3 | `clients/web/src/components/learner-profile` | Published | 2026-08-11 |
| [Managing parent notifications](/docs/parents/managing-parent-notifications) | parents | 2–3 | `clients/web/src/components/learner-profile` | Published | 2026-08-11 |
| [Connecting Lextures to your LMS with LTI 1.3](/docs/integrations/connecting-lms-with-lti-13) | integrations | 1 | `server/internal/lti` | Published | 2026-08-11 |
| [Connecting Lextures to Zapier](/docs/integrations/connecting-lextures-to-zapier) | integrations | 2–3 | `server/internal/lti` | Published | 2026-08-11 |
| [Using Lextures with Make](/docs/integrations/using-lextures-with-make) | integrations | 2–3 | `server/internal/lti` | Published | 2026-08-11 |
| [Creating and monitoring webhooks](/docs/integrations/using-webhooks) | integrations | 2–3 | `server/internal/lti` | Published | 2026-08-11 |
| [Publishing a marketplace course listing](/docs/marketplace/publishing-course-listing) | marketplace | 2–3 | `server/internal/models` | Published | 2026-08-11 |
| [Creating course coupons](/docs/marketplace/creating-course-coupons) | marketplace | 2–3 | `server/internal/models` | Published | 2026-08-11 |
| [Understanding marketplace payouts](/docs/marketplace/understanding-payouts) | marketplace | 2–3 | `server/internal/models` | Published | 2026-08-11 |
| [Handling marketplace refunds](/docs/marketplace/handling-refunds) | marketplace | 2–3 | `server/internal/models` | Published | 2026-08-11 |
| [Exporting all of your institution's data](/docs/admin/exporting-institution-data) | admin | 1 | `server/internal/httpserver` | Published | 2026-08-11 |
| [Roles and permissions reference](/docs/admin/roles-permissions-reference) | admin | 1 | `server/internal/httpserver` | Published | 2026-08-11 |
| [Reviewing audit logs](/docs/admin/reviewing-audit-logs) | admin | 2–3 | `server/internal/httpserver` | Published | 2026-08-11 |
| [Managing organization hierarchy](/docs/admin/managing-organization-hierarchy) | admin | 2–3 | `server/internal/httpserver` | Published | 2026-08-11 |
| [Signing in on a mobile device](/docs/mobile/signing-in-on-mobile) | mobile | 2–3 | `clients/web/src` | Published | 2026-08-11 |
| [Managing mobile notifications](/docs/mobile/mobile-notifications) | mobile | 2–3 | `clients/web/src` | Published | 2026-08-11 |
| [Preparing course content for limited connectivity](/docs/mobile/using-course-content-offline) | mobile | 2–3 | `clients/web/src` | Published | 2026-08-11 |
| [Troubleshooting mobile access](/docs/mobile/troubleshooting-mobile-access) | mobile | 2–3 | `clients/web/src` | Published | 2026-08-11 |
| [Self-hosting Lextures: requirements and install](/docs/self-hosting/self-hosting-requirements-install) | self-hosting | 1 | `docker-compose.yml` | Published | 2026-08-11 |
| [Upgrading and backing up a self-hosted instance](/docs/self-hosting/upgrading-and-backing-up) | self-hosting | 1 | `docker-compose.yml` | Published | 2026-08-11 |
| [Configuring a self-hosted instance](/docs/self-hosting/self-hosting-configuration) | self-hosting | 2–3 | `docker-compose.yml` | Published | 2026-08-11 |
| [Monitoring a self-hosted instance](/docs/self-hosting/self-hosting-health-checks) | self-hosting | 2–3 | `docker-compose.yml` | Published | 2026-08-11 |
| [How Lextures handles FERPA-protected records](/docs/compliance/ferpa-protected-records) | compliance | 1 | `docs/plan/standards` | Published | 2026-08-11 |
| [What student data Lextures stores, and for how long](/docs/compliance/student-data-retention) | compliance | 1 | `docs/plan/standards` | Published | 2026-08-11 |
| [Responding to a data subject request](/docs/compliance/responding-to-data-requests) | compliance | 2–3 | `docs/plan/standards` | Published | 2026-08-11 |
| [Reviewing privacy subprocessors](/docs/compliance/privacy-subprocessors) | compliance | 2–3 | `docs/plan/standards` | Published | 2026-08-11 |
