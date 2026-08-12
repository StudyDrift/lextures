---
title: "Which WCAG 2.2 AA features Lextures supports"
description: "Review keyboard, focus, semantics, contrast, zoom, and assistive-technology support alongside the current conformance report. Learn the key checks and safe ne."
date: 2026-08-11
updated: 2026-08-11
author: chase-willden
cluster: help-accessibility
primaryQuestion: "How do I use which wcag 2.2 aa features lextures supports in Lextures?"
keywords: [Lextures, accessibility, help]
category: accessibility
roles: [instructor, admin]
segments: [k12, higher-ed, homeschool]
verifiedAgainst: "2026-08-11 source"
supportTicketThemes: [accessibility-wcag-22-aa-features]
relatedTo: [/docs/accessibility/automatic-accommodations, /docs/accessibility/setting-learner-accommodations, /docs/accessibility/accessible-content-authoring]
contentContract: answer-first-v1
---

::: key-takeaways
- Review keyboard, focus, semantics, contrast, zoom, and assistive-technology support alongside the current conformance report.
- Confirm the available controls in your own organization because permissions and enabled features determine what appears.
- Use a test record or limited pilot first, then review the resulting learner and administrator experience.
:::

::: answer
Review keyboard, focus, semantics, contrast, zoom, and assistive-technology support alongside the current conformance report. Open the relevant settings, confirm your role permits the change, test with limited scope, and review the result from the affected user's view. Keep organization policy and learner privacy in the approval path.
:::

## Who can use this feature?

This guide applies to **instructor, admin** roles in K–12, higher education, and homeschool settings. The exact control is shown only when the signed-in person has the required permission and the capability is enabled for the organization. Global access does not replace course or organization policy. If the named page or action is absent, ask an administrator to confirm permissions before changing configuration.

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

Keep a support-safe note of the scope, time, and outcome. Do not copy passwords, tokens, accommodation details, or unnecessary learner records into that note. For broader context, visit the [help center](/docs), the [accessibility help category](/docs/accessibility), and [Lextures platform overview](/platform). See also [How accommodations are applied automatically](/docs/accessibility/automatic-accommodations), [Setting learner accommodations](/docs/accessibility/setting-learner-accommodations), and [Authoring accessible course content](/docs/accessibility/accessible-content-authoring).

The public platform overview provides the product context used by this help article.[^2]

::: faq
### Why can I not see this setting?
Your role, organization policy, course state, or deployed version may not expose it. Confirm that you are in the correct organization or course, then ask an administrator to review the relevant permission and feature configuration. Avoid working around a missing control with shared credentials.

### Should I test the change first?
Yes. Use the smallest realistic scope and synthetic data where possible. For identity, roster, grading, accommodation, payment, or integration changes, verify both the intended path and a user who should remain unaffected before expanding the configuration.

### What information should I send to support?
Send the page name, approximate time, your role, the affected scope, and a description of expected and observed behavior. Remove learner records and secrets. Include a screenshot only when it contains synthetic data or has been reviewed and redacted.
:::

[^1]: https://github.com/lextures/lextures
[^2]: https://lextures.com/platform
