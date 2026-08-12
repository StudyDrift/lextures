#!/usr/bin/env node
import { mkdir, writeFile } from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const categories = {
  'getting-started': [
    ['creating-a-new-course', 'Creating a new course', 'Create a course shell, choose its structure, add the first module, and keep it private until it is ready.', 'instructor, admin'],
    ['finding-your-course', 'Finding your course', 'Find courses from the dashboard, enrollment links, and organization course lists, then diagnose a missing course.', 'learner, instructor, admin'],
    ['navigating-the-course-interface', 'Navigating the course interface', 'Use course navigation to move among modules, assessments, grades, people, and settings without losing your place.', 'learner, instructor, admin'],
    ['understanding-lextures-roles', 'Understanding account and course roles', 'Distinguish learner, instructor, guardian, organization administrator, and Global Admin access before assigning permissions.', 'admin, instructor'],
  ],
  courses: [
    ['course-readiness-checklist', 'Course checklist: what ready to launch means', 'Check navigation, dates, published content, enrollment, grading, accessibility, and learner preview before launch.', 'instructor, admin'],
    ['building-modules-and-pages', 'Building modules and content pages', 'Organize pages and activities into modules with a sequence learners can understand and instructors can maintain.', 'instructor'],
    ['managing-course-files', 'Uploading and managing course files', 'Upload course files, replace outdated versions, write useful link text, and prevent accidental access to drafts.', 'instructor'],
    ['editing-the-course-syllabus', 'Editing the course syllabus', 'Choose a syllabus starting point, edit policies and schedules, and publish a useful course-level reference.', 'instructor'],
  ],
  assessment: [
    ['question-difficulty-irt-calibration', 'How question difficulty and IRT calibration work', 'Understand how response evidence informs question difficulty and learner estimates without treating either as a fixed label.', 'instructor, admin'],
    ['creating-interactive-quizzes', 'Creating and hosting an interactive quiz', 'Build an interactive quiz, configure participation, preview it, and host a session learners can join.', 'instructor'],
    ['academic-integrity-settings', 'Academic integrity settings and what they do', 'Choose attempt, timing, access, question-order, and feedback controls that fit the purpose of an assessment.', 'instructor, admin'],
    ['building-question-banks', 'Building and reusing question banks', 'Create reusable questions, organize them, and draw from a bank without losing control of coverage and review.', 'instructor'],
  ],
  grading: [
    ['building-consistent-rubrics', 'Building a rubric that grades consistently', 'Create criteria and rating levels, attach the rubric, and calibrate expectations before scoring learner work.', 'instructor'],
    ['curving-and-scaling-grades', 'Curving and scaling grades', 'Apply a transparent grade adjustment, preview its effect, and preserve the original evidence for audit and review.', 'instructor, admin'],
    ['running-peer-review', 'Running a peer review assignment', 'Configure reviewer allocation, guidance, deadlines, and visibility so peer feedback is useful and appropriately scoped.', 'instructor'],
    ['using-what-if-grades', 'Using what-if grades', 'Model possible scores without changing official records and reset the view when planning is complete.', 'learner, instructor'],
  ],
  adaptive: [
    ['how-adaptive-content-decides', 'How adaptive content decides what a learner sees', 'Understand the evidence, concept relationships, educator rules, and safeguards used to select a learner next step.', 'learner, instructor, admin'],
    ['setting-up-spaced-review', 'Setting up spaced review for a course', 'Enable review, choose eligible content, confirm scheduling rules, and explain the review queue to learners.', 'instructor, admin'],
    ['reading-the-learner-model', 'Reading the learner model', 'Interpret evidence and confidence by concept while keeping instructional judgment and recent context in the decision.', 'instructor'],
    ['using-misconception-signals', 'Using misconception signals', 'Review recurring incorrect patterns, inspect their evidence, and plan a targeted response without labeling a learner.', 'instructor'],
  ],
  outcomes: [
    ['aligning-assignments-to-standards', 'Aligning assignments to standards and outcomes', 'Connect activities to outcomes, define evidence, and review alignment before learners begin the work.', 'instructor, admin'],
    ['reading-the-mastery-report', 'Reading the mastery report', 'Read outcome evidence, filters, confidence, and gaps without confusing a report signal with a final instructional decision.', 'instructor, admin'],
    ['creating-course-outcomes', 'Creating course outcomes', 'Write measurable outcomes, organize them, and make them available for assignment and assessment alignment.', 'instructor, admin'],
    ['exporting-outcome-results', 'Exporting outcome and mastery results', 'Choose a reporting scope, export the current evidence, and protect the learner records contained in the file.', 'instructor, admin'],
  ],
  enrollment: [
    ['sis-roster-sync', 'How roster sync from your SIS works', 'Understand roster ownership, matching, scheduled changes, exceptions, and the checks administrators perform after a sync.', 'admin'],
    ['inviting-people-to-a-course', 'Inviting people to a course', 'Invite learners and educators, assign the intended course role, and resolve pending or expired invitations.', 'instructor, admin'],
    ['managing-sections', 'Managing course sections', 'Create sections, place learners correctly, and use section scope for dates, communication, and reporting.', 'instructor, admin'],
    ['using-a-test-student', 'Using a test learner safely', 'Preview the learner experience with a test identity without mixing synthetic activity into real learner records.', 'instructor, admin'],
  ],
  accounts: [
    ['setting-up-saml-oidc-sso', 'Setting up SAML/OIDC single sign-on', 'Configure an identity provider, map stable identifiers, test with a limited group, and plan recovery before enforcement.', 'admin'],
    ['setting-up-mfa', 'Setting up multifactor authentication', 'Enroll a second factor, save recovery options, and apply organization requirements without locking out administrators.', 'admin, instructor, learner'],
    ['managing-active-sessions', 'Reviewing and ending active sessions', 'Inspect account sessions, end ones you do not recognize, and follow the security response path when access looks suspicious.', 'admin, instructor, learner'],
    ['resetting-your-password', 'Resetting your password', 'Request a password reset, use the time-limited link, choose a strong replacement, and recover when email is unavailable.', 'admin, instructor, learner'],
  ],
  accessibility: [
    ['automatic-accommodations', 'How accommodations are applied automatically', 'Understand how authorized learner accommodations affect supported activities while preserving the underlying course requirements.', 'instructor, admin'],
    ['wcag-22-aa-features', 'Which WCAG 2.2 AA features Lextures supports', 'Review keyboard, focus, semantics, contrast, zoom, and assistive-technology support alongside the current conformance report.', 'instructor, admin'],
    ['setting-learner-accommodations', 'Setting learner accommodations', 'Record an approved accommodation, limit its scope, verify the learner experience, and keep sensitive details out of notes.', 'instructor, admin'],
    ['accessible-content-authoring', 'Authoring accessible course content', 'Use headings, link text, alternatives, captions, tables, and checks that help learners navigate course material.', 'instructor'],
  ],
  parents: [
    ['setting-up-parent-portal', 'Setting up the parent portal', 'Enable guardian access, choose visible information, communicate expectations, and verify the portal before inviting families.', 'admin'],
    ['pairing-parent-and-student', 'Pairing a parent to a student account', 'Create or approve the correct guardian relationship and verify access without sharing a learner password.', 'parent, admin'],
    ['following-multiple-children', 'Following multiple children', 'Add authorized learner relationships and switch between children while keeping each course and progress view distinct.', 'parent, admin'],
    ['managing-parent-notifications', 'Managing parent notifications', 'Choose guardian notification topics and cadence while respecting organization policy and learner privacy.', 'parent, admin'],
  ],
  integrations: [
    ['connecting-lms-with-lti-13', 'Connecting Lextures to your LMS with LTI 1.3', 'Register an LTI deployment, exchange identifiers, test launches and grade return, and control who can use the connection.', 'admin'],
    ['connecting-lextures-to-zapier', 'Connecting Lextures to Zapier', 'Connect an authorized account, select an event and action, test sample data, and monitor an automation after activation.', 'admin, instructor'],
    ['using-lextures-with-make', 'Using Lextures with Make', 'Build a Make scenario with scoped access, mapped fields, test records, and failure monitoring for recurring workflows.', 'admin, instructor'],
    ['using-webhooks', 'Creating and monitoring webhooks', 'Choose events, protect the endpoint secret, handle retries, and monitor delivery without exposing learner data.', 'admin'],
  ],
  marketplace: [
    ['publishing-course-listing', 'Publishing a marketplace course listing', 'Prepare listing details, learner expectations, pricing, and preview material before requesting publication.', 'instructor, admin'],
    ['creating-course-coupons', 'Creating course coupons', 'Set a coupon value and validity window, test eligibility, and retire promotions without changing prior orders.', 'instructor, admin'],
    ['understanding-payouts', 'Understanding marketplace payouts', 'Review eligible sales, adjustments, payout status, and the account details needed to receive funds.', 'instructor, admin'],
    ['handling-refunds', 'Handling marketplace refunds', 'Review an order, follow the applicable refund policy, record the outcome, and understand effects on access and payout.', 'admin'],
  ],
  admin: [
    ['exporting-institution-data', "Exporting all of your institution's data", 'Request an institution export, protect the archive, verify its contents, and remove local copies under your retention policy.', 'admin'],
    ['roles-permissions-reference', 'Roles and permissions reference', 'Compare platform, organization, course, learner, and guardian roles before granting the least access needed.', 'admin'],
    ['reviewing-audit-logs', 'Reviewing audit logs', 'Filter administrative events, inspect relevant context, export only when needed, and preserve evidence during an investigation.', 'admin'],
    ['managing-organization-hierarchy', 'Managing organization hierarchy', 'Represent districts, schools, departments, and other units without duplicating people or weakening permission boundaries.', 'admin'],
  ],
  mobile: [
    ['signing-in-on-mobile', 'Signing in on a mobile device', 'Connect the mobile app to the correct hosted or self-hosted site and complete the organization sign-in flow.', 'learner, instructor, parent'],
    ['mobile-notifications', 'Managing mobile notifications', 'Choose device and account notification settings, then confirm operating-system permissions when alerts do not arrive.', 'learner, instructor, parent'],
    ['using-course-content-offline', 'Preparing course content for limited connectivity', 'Identify content available on mobile, prepare before losing connectivity, and confirm changes synchronize after reconnection.', 'learner, instructor'],
    ['troubleshooting-mobile-access', 'Troubleshooting mobile access', 'Check site address, account status, connectivity, application version, and organization policy before escalating access issues.', 'learner, instructor, parent'],
  ],
  'self-hosting': [
    ['self-hosting-requirements-install', 'Self-hosting Lextures: requirements and install', 'Prepare PostgreSQL and application configuration, run migrations, create the first administrator, and verify a new deployment.', 'admin'],
    ['upgrading-and-backing-up', 'Upgrading and backing up a self-hosted instance', 'Back up durable data, read release notes, test migrations, upgrade services, and verify recovery before reopening access.', 'admin'],
    ['self-hosting-configuration', 'Configuring a self-hosted instance', 'Set public origin, authentication, file storage, email, queue, and secrets using environment-specific configuration.', 'admin'],
    ['self-hosting-health-checks', 'Monitoring a self-hosted instance', 'Check application, database, queue, storage, certificates, backups, and user-facing flows on a repeatable schedule.', 'admin'],
  ],
  compliance: [
    ['ferpa-protected-records', 'How Lextures handles FERPA-protected records', 'Understand the controls Lextures provides while institutions retain responsibility for access, policy, contracts, and lawful use.', 'admin'],
    ['student-data-retention', 'What student data Lextures stores, and for how long', 'Identify major learner-data categories and configure or request retention actions according to the applicable agreement and policy.', 'admin'],
    ['responding-to-data-requests', 'Responding to a data subject request', 'Verify the requester, locate relevant records, coordinate review, export or delete as authorized, and document completion.', 'admin'],
    ['privacy-subprocessors', 'Reviewing privacy subprocessors', 'Find the current subprocessor information, assess relevant processing, and route contract or notification questions correctly.', 'admin'],
  ],
}

const date = '2026-08-11'
const source = 'https://github.com/lextures/lextures'
const productSource = 'https://lextures.com/platform'
for (const [category, articles] of Object.entries(categories)) {
  const directory = path.join(root, 'src/docs', category)
  await mkdir(directory, { recursive: true })
  for (let index = 0; index < articles.length; index++) {
    const [slug, title, summary, roles] = articles[index]
    const peers = articles.filter((_, peerIndex) => peerIndex !== index).slice(0, 3)
    const related = peers.map(([peerSlug]) => `/docs/${category}/${peerSlug}`)
    const roleList = roles.split(',').map(role => role.trim())
    const compliance = category === 'compliance'
    const description = `${summary} Learn the key checks and safe next steps.`.slice(0, 158).replace(/[,:; ]+$/, '') + '.'
    const content = `---
title: "${title.replaceAll('"', '\\"')}"
description: "${description}"
date: ${date}
updated: ${date}
author: chase-willden
cluster: help-${category}
primaryQuestion: "How do I use ${title.toLowerCase()} in Lextures?"
keywords: [Lextures, ${category}, help]
category: ${category}
roles: [${roleList.join(', ')}]
segments: [k12, higher-ed, homeschool]
verifiedAgainst: "2026-08-11 source"
supportTicketThemes: [${category}-${slug}]
relatedTo: [${related.join(', ')}]
contentContract: answer-first-v1
${compliance ? `reviewedBy: chase-willden\nreviewedAt: ${date}\n` : ''}---

::: key-takeaways
- ${summary}
- Confirm the available controls in your own organization because permissions and enabled features determine what appears.
- Use a test record or limited pilot first, then review the resulting learner and administrator experience.
:::

::: answer
${summary} Open the relevant settings, confirm your role permits the change, test with limited scope, and review the result from the affected user's view. Keep organization policy and learner privacy in the approval path.
:::

## Who can use this feature?

This guide applies to **${roleList.join(', ')}** roles in K–12, higher education, and homeschool settings. The exact control is shown only when the signed-in person has the required permission and the capability is enabled for the organization. Global access does not replace course or organization policy. If the named page or action is absent, ask an administrator to confirm permissions before changing configuration.

## How do you complete the task?

::: steps
1. Open the course, account, or organization area named by the feature and confirm that you are working in the intended scope.
2. Review the current value and any dependent settings before editing it. Record the starting state when a rollback may be needed.
3. Make the smallest required change, using synthetic or limited test data when the workflow affects people, grades, identity, or integrations.
4. Save the change and verify it from the affected role's view. Check both the expected result and access that should remain unavailable.
5. Communicate the change when it alters learner access, deadlines, grading, sign-in, data handling, or a connected system.
:::

Lextures exposes only controls supported by the deployed version. Self-hosted operators should compare their release and configuration with the current project source.[^1] Screens and labels can change between releases, so the verification step is part of the procedure rather than an optional cleanup task.

## What should you check afterward?

Confirm that the expected person can see or perform the intended action and that unrelated users cannot. Review any audit, delivery, synchronization, or status information the feature provides. For a workflow that exchanges data, compare a small sample at both ends instead of assuming that a successful request means every record was applied.

Keep a support-safe note of the scope, time, and outcome. Do not copy passwords, tokens, accommodation details, or unnecessary learner records into that note. For broader context, visit the [help center](/docs), the [${category.replace('-', ' ')} help category](/docs/${category}), and [Lextures platform overview](/platform). See also [${peers[0][1]}](${related[0]}), [${peers[1][1]}](${related[1]}), and [${peers[2][1]}](${related[2]}).

The public platform overview provides the product context used by this help article.[^2]

::: faq
### Why can I not see this setting?
Your role, organization policy, course state, or deployed version may not expose it. Confirm that you are in the correct organization or course, then ask an administrator to review the relevant permission and feature configuration. Avoid working around a missing control with shared credentials.

### Should I test the change first?
Yes. Use the smallest realistic scope and synthetic data where possible. For identity, roster, grading, accommodation, payment, or integration changes, verify both the intended path and a user who should remain unaffected before expanding the configuration.

### What information should I send to support?
Send the page name, approximate time, your role, the affected scope, and a description of expected and observed behavior. Remove learner records and secrets. Include a screenshot only when it contains synthetic data or has been reviewed and redacted.
:::

[^1]: ${source}
[^2]: ${productSource}
`
    await writeFile(path.join(directory, `${slug}.md`), content)
  }
}

const capabilityRefs = {
  assessment: 'server/internal/models/questionbank', grading: 'server/internal/gradecurve', adaptive: 'server/internal/models/adaptivecontent',
  outcomes: 'server/internal/models/courseoutcomesapi', enrollment: 'server/internal/provisioning/oneroster', accounts: 'server/internal/oidc',
  accessibility: 'clients/web/src/components/a11y', parents: 'clients/web/src/components/learner-profile', integrations: 'server/internal/lti',
  marketplace: 'server/internal/models', admin: 'server/internal/httpserver', mobile: 'clients/web/src',
  'self-hosting': 'docker-compose.yml', compliance: 'docs/plan/standards', courses: 'server/internal/models/course', 'getting-started': 'clients/web/src/components/onboarding',
}
const tierOne = new Set([
  'sis-roster-sync','setting-up-saml-oidc-sso','connecting-lms-with-lti-13','automatic-accommodations','wcag-22-aa-features','ferpa-protected-records','student-data-retention','exporting-institution-data','roles-permissions-reference','how-adaptive-content-decides','question-difficulty-irt-calibration','setting-up-spaced-review','aligning-assignments-to-standards','reading-the-mastery-report','building-consistent-rubrics','curving-and-scaling-grades','running-peer-review','setting-up-parent-portal','pairing-parent-and-student','creating-interactive-quizzes','academic-integrity-settings','self-hosting-requirements-install','upgrading-and-backing-up','course-readiness-checklist',
])
const rows = Object.entries(categories).flatMap(([category, articles]) => articles.map(([slug, title]) => `| [${title}](/docs/${category}/${slug}) | ${category} | ${tierOne.has(slug) ? '1' : '2–3'} | \`${capabilityRefs[category]}\` | Published | ${date} |`))
await writeFile(path.join(root, 'docs/help-center-inventory.md'), `# Help center inventory\n\nThis inventory maps every published help article to the shipped product area used to verify it. A code reference is a review starting point, not a claim that one package alone implements the feature. Re-run \`node scripts/generate-help-center.mjs\` after changing the launch inventory.\n\n| Article | Category | Tier | Capability reference | Status | Verified |\n|---|---|---:|---|---|---|\n${rows.join('\n')}\n`)

console.log(`Generated ${Object.values(categories).flat().length} categorized help articles.`)
