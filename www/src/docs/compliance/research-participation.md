---
title: "Control aggregate research participation"
description: "Record whether your organization may join future de-identified Lextures research, understand what opt-out changes, and verify the audit record."
date: 2026-08-11
updated: 2026-08-11
reviewDue: 2027-02-11
author: chase-willden
cluster: help-compliance
primaryQuestion: "How does an administrator control aggregate research participation?"
keywords: [aggregate research, privacy, opt out, organization settings]
category: compliance
roles: [admin]
segments: [k12, higher-ed, homeschool]
verifiedAgainst: "SEO.12 implementation and API contract, 2026-08-11"
supportTicketThemes: [research-participation, privacy-controls]
relatedTo: [/docs/compliance/ferpa-protected-records, /docs/compliance/student-data-retention, /docs/compliance/responding-to-data-requests]
contentContract: answer-first-v1
reviewedBy: chase-willden
reviewedAt: 2026-08-11
---

::: key-takeaways
- An unresolved participation decision excludes the organization from research extracts.
- An organization administrator can choose Participate or Opt out in organization settings.
- Opting out applies to all future extracts and every saved change creates an audit record.
- Published reports contain only large, de-identified aggregates and cannot identify one organization.
:::

::: answer
Open **Settings**, choose **Organizations**, and find **Aggregate research participation**. Select the
correct organization, choose **Participate** or **Opt out**, and save. An unresolved decision excludes
the organization. A saved opt-out blocks its records from every future research extract and creates an
administrator audit entry.
:::

## Who can change the decision?

An organization administrator can read and change the setting for their organization. A global
administrator can select an organization and manage its decision. Other signed-in members may read
the current decision when they have organization access, but they cannot change it.

The setting records a decision; it does not decide the legal default. Your contract, data processing
agreement, local law, and required notice determine whether participation can be offered. If those
questions are not resolved, leave the setting unresolved. Lextures treats that state as excluded.

## How do you opt in or opt out?

::: steps
1. Open **Settings → Organizations** and confirm the organization shown in the research card.
2. Read the linked public methodology, including the data, suppression, and correction rules.
3. Choose **Participate** only after your contract and jurisdiction review permits it. Otherwise choose **Opt out**.
4. Select **Save decision** and wait for the saved confirmation.
5. If your organization requires evidence, ask an audit-log reader to verify the `org_settings_change` event.
:::

The API accepts only `opt_in` or `opt_out`. A missing record is not treated as consent. Saving the same
decision again updates its timestamp and records the administrator action. Do not use a shared account
to make this change because the audit entry must identify the actual decision-maker.

## What does the choice affect?

Participation applies only to future original-research extracts. It does not change learning features,
grades, reports available to your staff, or ordinary product operations. An opt-out is checked before a
research extract reaches an analyst. The extraction process also runs an automated assertion against
the current opt-out list.

Analysts do not receive names, email addresses, user identifiers, course identifiers, instructor
identifiers, school identifiers, or tenant identifiers.[^2] Public cells require at least 50 learners[^1]
and 10 institutions. A second cell is hidden when subtraction could reveal a smaller cell. Report-specific
methods describe the sample, dates, exclusions, statistical methods, and limits.

Opting out does not rewrite an older, already published aggregate. Those figures cannot be assigned to
one organization and remain available so existing citations stay accurate. Future report versions and
extracts exclude the organization. For a contract question or suspected privacy issue, contact your
Lextures representative before changing the setting.

Read the [public research methodology](/resources/research/methodology), see the [research publication
schedule](/resources/research), and review [how Lextures handles FERPA-protected
records](/docs/compliance/ferpa-protected-records).

::: faq
### What happens if no decision has been recorded?
The organization is excluded from research extracts. Lextures does not treat silence, a missing row, or
an unavailable administrator as consent. Record a decision only after the contract and jurisdiction
rules are known.

### Does opting out remove the organization from product analytics?
No. This setting controls the original-research program described on the public methodology page. It
does not change operational logs or analytics needed to provide, secure, and improve the service under
the applicable agreement.

### Can a report identify our school?
Published reports cannot include a cell attributable to one school, tenant, course, or instructor.
Minimum population thresholds, coarse segments, de-identification, and complementary suppression are
checked before publication.

### How can I confirm the change was saved?
The settings card displays a saved confirmation. An authorized audit-log reader can also find the
organization settings change, including the earlier and new participation states, actor, and time.
:::

[^1]: https://lextures.com/resources/research/methodology
[^2]: https://csrc.nist.gov/pubs/sp/800/188/final
