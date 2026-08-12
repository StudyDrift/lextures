/**
 * Curated llms.txt entries (SEO.2 FR-12/13).
 *
 * Descriptions state the *question the page answers* so a model can decide
 * whether to fetch — not a restatement of the title.
 *
 * Cap: ≤ 200 links. Order: Product → Segments → Pricing → Help → Guides →
 * Trust → Courses hub only.
 */

export type LlmsLink = {
  title: string
  /** Path starting with / */
  path: string
  /** One-sentence description of the question this page answers. */
  description: string
}

export type LlmsSection = {
  heading: string
  links: LlmsLink[]
}

export const LLMS_SUMMARY =
  'Lextures is an adaptive learning management system for K–12, higher education, and homeschool: IRT-routed quizzes, gradebook, roster sync, self-hosting under AGPL-3.0, and a public course marketplace.'

export const LLMS_SECTIONS: LlmsSection[] = [
  {
    heading: 'Product',
    links: [
      {
        title: 'Lextures home',
        path: '/',
        description:
          'What Lextures is, how adaptive quizzing and grading fit in one platform, and who it is for.',
      },
      {
        title: 'Get started',
        path: '/get-started',
        description:
          'How to create a free account for a school or start as a homeschool learner, hosted or self-hosted.',
      },
      {
        title: 'About Lextures',
        path: '/about',
        description:
          'Who builds Lextures, founding date, mission, ownership, and official sameAs profiles.',
      },
      {
        title: 'Course marketplace',
        path: '/courses',
        description:
          'Browse public marketplace courses available to enroll in on Lextures.',
      },
    ],
  },
  {
    heading: 'Segments',
    links: [
      {
        title: 'Higher education',
        path: '/higher-ed',
        description:
          'SSO, SCIM, LTI 1.3, grade audit trails, and self-hosting options for colleges and universities.',
      },
      {
        title: 'K–12 schools',
        path: '/k12',
        description:
          'Standards-aligned gradebook, roster sync, spaced repetition, and misconception flags for districts.',
      },
      {
        title: 'Homeschool',
        path: '/homeschool',
        description:
          'How families create or enroll in courses, practice with adaptive quizzes, and self-host for free.',
      },
      {
        title: 'Parents & guardians',
        path: '/parents',
        description:
          'What parents can see (grades, due dates) when a district enables the Lextures parent portal.',
      },
    ],
  },
  {
    heading: 'Pricing',
    links: [
      {
        title: 'Pricing',
        path: '/pricing',
        description:
          'Pricing for all three segments, including per-student institution bands and the homeschool free tier.',
      },
      {
        title: 'Pricing calculator',
        path: '/pricing/calculator',
        description:
          'Estimate hosted Lextures cost by enrollment size with automatic bulk discounts.',
      },
      {
        title: 'Request information',
        path: '/request-information',
        description:
          'How institutions request a demo or quote with enrollment size and SSO requirements.',
      },
    ],
  },
  {
    heading: 'Help center',
    links: [
      {
        title: 'Documentation hub',
        path: '/docs',
        description: 'Index of operator and instructor guides for Lextures.',
      },
      ...[
        ['Enrollment help', '/docs/enrollment', 'How to manage rosters, sections, invitations, roles, and test learners.'],
        ['Accounts and security help', '/docs/accounts', 'How to configure sign-in, SSO, MFA, passwords, and sessions.'],
        ['Integrations help', '/docs/integrations', 'How to connect Lextures through LTI, automation tools, and webhooks.'],
        ['Accessibility help', '/docs/accessibility', 'How accommodations and accessible product and authoring features work.'],
        ['Compliance help', '/docs/compliance', 'How to review privacy, learner records, retention, and data requests.'],
        ['Administration help', '/docs/admin', 'How to manage permissions, exports, audit logs, and organization structure.'],
        ['Adaptive learning help', '/docs/adaptive', 'How adaptive selection, spaced review, learner models, and misconception signals work.'],
        ['Assessment help', '/docs/assessment', 'How to create question banks, quizzes, IRT-based assessment, and integrity rules.'],
        ['Outcomes help', '/docs/outcomes', 'How to align standards and interpret mastery evidence.'],
        ['Grading help', '/docs/grading', 'How to use rubrics, curves, peer review, and what-if grades.'],
        ['Parents and guardians help', '/docs/parents', 'How to set up guardian access, pairing, multi-child views, and notifications.'],
        ['Courses and content help', '/docs/courses', 'How to build, check, and publish course content.'],
        ['Self-hosting help', '/docs/self-hosting', 'How to install, configure, back up, upgrade, and monitor Lextures.'],
        ['SIS roster sync', '/docs/enrollment/sis-roster-sync', 'How SIS roster ownership, matching, changes, and sync checks work.'],
        ['SAML and OIDC SSO', '/docs/accounts/setting-up-saml-oidc-sso', 'How administrators configure and safely test institutional single sign-on.'],
        ['LTI 1.3 connection', '/docs/integrations/connecting-lms-with-lti-13', 'How to register an LTI deployment and verify launch and grade return.'],
        ['Automatic accommodations', '/docs/accessibility/automatic-accommodations', 'How authorized learner accommodations affect supported activities.'],
        ['WCAG 2.2 AA features', '/docs/accessibility/wcag-22-aa-features', 'Which keyboard, semantic, contrast, zoom, and assistive-technology features are supported.'],
        ['FERPA-protected records', '/docs/compliance/ferpa-protected-records', 'How Lextures controls support an institution handling education records.'],
        ['Learner data retention', '/docs/compliance/student-data-retention', 'What major learner-data categories exist and how retention actions are handled.'],
        ['Institution data export', '/docs/admin/exporting-institution-data', 'How an administrator requests, verifies, protects, and removes an institution export.'],
        ['Roles and permissions', '/docs/admin/roles-permissions-reference', 'How platform, organization, course, learner, and guardian roles differ.'],
        ['Adaptive content selection', '/docs/adaptive/how-adaptive-content-decides', 'What evidence and educator rules influence a learner next step.'],
        ['IRT calibration', '/docs/assessment/question-difficulty-irt-calibration', 'How question difficulty and learner estimates use response evidence.'],
        ['Spaced review', '/docs/adaptive/setting-up-spaced-review', 'How to enable, scope, and verify spaced review for a course.'],
        ['Standards alignment', '/docs/outcomes/aligning-assignments-to-standards', 'How to connect activities to standards and outcomes.'],
        ['Mastery report', '/docs/outcomes/reading-the-mastery-report', 'How to interpret outcome evidence, filters, confidence, and gaps.'],
        ['Consistent rubrics', '/docs/grading/building-consistent-rubrics', 'How to define and calibrate rubric criteria and rating levels.'],
        ['Curving grades', '/docs/grading/curving-and-scaling-grades', 'How to preview and apply a transparent grade adjustment.'],
        ['Peer review', '/docs/grading/running-peer-review', 'How to configure reviewer allocation, guidance, deadlines, and visibility.'],
        ['Parent portal', '/docs/parents/setting-up-parent-portal', 'How administrators enable, scope, and verify guardian access.'],
        ['Parent pairing', '/docs/parents/pairing-parent-and-student', 'How to create or approve a guardian relationship without sharing learner credentials.'],
        ['Interactive quizzes', '/docs/assessment/creating-interactive-quizzes', 'How to build, preview, and host an interactive quiz.'],
        ['Academic integrity settings', '/docs/assessment/academic-integrity-settings', 'How attempt, timing, access, ordering, and feedback controls work.'],
        ['Course readiness checklist', '/docs/courses/course-readiness-checklist', 'What to verify before publishing and enrolling learners in a course.'],
        ['Self-hosting upgrades and backups', '/docs/self-hosting/upgrading-and-backing-up', 'How to back up durable data, test migrations, upgrade, and verify recovery.'],
      ].map(([title, path, description]) => ({ title, path, description })),
      {
        title: 'Creating a new course',
        path: '/docs/getting-started/creating-a-new-course',
        description: 'Step-by-step how an instructor creates and publishes a course in Lextures.',
      },
      {
        title: 'Finding your course',
        path: '/docs/getting-started/finding-your-course',
        description: 'How learners and instructors locate a course after signup or enrollment.',
      },
      {
        title: 'Navigating the course interface',
        path: '/docs/getting-started/navigating-the-course-interface',
        description: 'Where modules, quizzes, gradebook, and settings live inside a course.',
      },
      {
        title: 'Self-hosting Lextures',
        path: '/docs/self-hosting/self-hosting-requirements-install',
        description:
          'How to install Lextures with Docker Compose, bootstrap the first Global Admin, and run locally.',
      },
      {
        title: 'Zapier integration',
        path: '/docs/integrations/connecting-lextures-to-zapier',
        description: 'How to connect Lextures events and actions to Zapier automations.',
      },
      {
        title: 'Make.com integration',
        path: '/docs/integrations/using-lextures-with-make',
        description: 'How to connect Lextures to Make.com scenarios for roster and grade workflows.',
      },
    ],
  },
  {
    heading: 'Guides & research',
    links: [
      {
        title: 'Blog',
        path: '/blog',
        description: 'Essays on adaptive learning, assessment design, and AI in education.',
      },
      {
        title: 'Adaptive AI and personalized learning',
        path: '/blog/adaptive-ai-and-education',
        description:
          'What Item Response Theory-based personalization means versus simple branching logic.',
      },
      {
        title: "Bloom's taxonomy in the age of AI",
        path: '/blog/blooms-taxonomy-in-the-age-of-ai',
        description:
          'How generative AI changes assessment design across Bloom levels.',
      },
      {
        title: 'Effective rubrics with AI',
        path: '/blog/effective-rubrics-in-the-age-of-ai',
        description: 'How to write rubrics that stay valid when students and graders use AI tools.',
      },
      {
        title: 'Rethinking assessment in the AI era',
        path: '/blog/rethinking-assessment-in-the-ai-era',
        description: 'Practical shifts in assessment design when AI writing tools are ubiquitous.',
      },
      {
        title: 'The synthetic renaissance',
        path: '/blog/the-synthetic-renaissance',
        description: 'How synthetic media and AI interact with culture and education systems.',
      },
    ],
  },
  {
    heading: 'Trust & compliance',
    links: [
      {
        title: 'Security',
        path: '/security',
        description: 'Security practices and how to report a vulnerability to Lextures.',
      },
      {
        title: 'Accessibility',
        path: '/accessibility',
        description: 'Accessibility conformance statement and WCAG 2.2 Level AA commitment.',
      },
      {
        title: 'VPAT',
        path: '/accessibility/vpat',
        description:
          'Voluntary Product Accessibility Template (VPAT 2.5) accessibility conformance report.',
      },
      {
        title: 'Privacy Policy',
        path: '/privacy',
        description: 'How Lextures collects, uses, and protects personal data.',
      },
      {
        title: 'Terms of Service',
        path: '/terms',
        description: 'Terms governing Lextures software, hosted services, and marketplace courses.',
      },
      {
        title: 'California privacy rights',
        path: '/privacy-rights/california',
        description: 'CCPA/CPRA rights when using Lextures products and services.',
      },
    ],
  },
]

/** Flatten all curated links (for caps / validation). */
export function flattenLlmsLinks(sections: LlmsSection[] = LLMS_SECTIONS): LlmsLink[] {
  return sections.flatMap(s => s.links)
}

/**
 * Render llms.txt (llmstxt.org style).
 */
export function renderLlmsTxt(
  siteOrigin: string,
  sections: LlmsSection[] = LLMS_SECTIONS,
  summary: string = LLMS_SUMMARY,
): string {
  const origin = siteOrigin.replace(/\/$/, '')
  const links = flattenLlmsLinks(sections)
  if (links.length > 200) {
    throw new Error(`llms.txt exceeds 200-link cap (${links.length})`)
  }
  for (const link of links) {
    if (!link.description?.trim()) {
      throw new Error(`llms.txt link missing description: ${link.path}`)
    }
  }

  const lines: string[] = [
    '# Lextures',
    '',
    `> ${summary}`,
    '',
  ]

  for (const section of sections) {
    lines.push(`## ${section.heading}`)
    lines.push('')
    for (const link of section.links) {
      const url = `${origin}${link.path === '/' ? '/' : link.path}`
      lines.push(`- [${link.title}](${url}): ${link.description}`)
    }
    lines.push('')
  }

  lines.push('## Optional')
  lines.push('')
  lines.push(
    `- [Full text corpus](${origin}/llms-full.txt): concatenated help-center and blog markdown for offline grounding.`,
  )
  lines.push('')

  return lines.join('\n')
}
