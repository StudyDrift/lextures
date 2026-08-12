export type HelpCategory = {
  id: string
  title: string
  description: string
  order: number
  platformPath: string
}

export const HELP_CATEGORIES: HelpCategory[] = [
  ['getting-started', 'Getting started', 'Set up your account, understand Lextures roles, create your first course, and learn the basic navigation used by every learner and educator.', 1, '/get-started'],
  ['courses', 'Courses & content', 'Build course structure, modules, pages, files, syllabi, and reusable learning content. These guides take a course from an empty shell to a clear learning experience.', 2, '/platform'],
  ['assessment', 'Assessment & quizzes', 'Create question banks, quizzes, interactive activities, and assessment rules. Learn how difficulty, delivery, attempts, and integrity controls work together.', 3, '/platform/assessment'],
  ['grading', 'Grading & feedback', 'Configure gradebooks, rubrics, submissions, peer review, curves, and feedback workflows that remain understandable to educators and learners.', 4, '/platform/grading'],
  ['adaptive', 'Adaptive learning', 'Understand learner models, concept relationships, adaptive paths, spaced review, and misconception signals, including what educators control.', 5, '/platform/adaptive-learning'],
  ['outcomes', 'Outcomes & standards', 'Align learning activities to standards and outcomes, define mastery, and read the reports that connect evidence to instructional goals.', 6, '/platform/analytics'],
  ['enrollment', 'People & enrollment', 'Add people, manage sections and roles, sync rosters, handle invitations and waitlists, and use a test learner safely.', 7, '/platform'],
  ['accounts', 'Accounts & security', 'Manage sign-in, single sign-on, multifactor authentication, password rules, active sessions, and account recovery for your organization.', 8, '/security'],
  ['accessibility', 'Accessibility & accommodations', 'Apply learner accommodations and use accessible authoring, keyboard, display, and assistive-technology features throughout Lextures.', 9, '/platform/accessibility'],
  ['parents', 'Parents & guardians', 'Enable the parent portal, pair guardians with learners, manage notifications, and follow progress for one or more children.', 10, '/parents'],
  ['integrations', 'Integrations', 'Connect Lextures with LMS, SIS, automation, calendar, webhook, and API workflows while keeping ownership and failure handling clear.', 11, '/integrations'],
  ['marketplace', 'Marketplace & payments', 'Publish course listings, set pricing and coupons, understand payouts, and respond to refunds and tax-related marketplace states.', 12, '/courses'],
  ['admin', 'Administration', 'Operate institutions and organizations: hierarchy, permissions, platform settings, audit history, exports, and routine governance tasks.', 13, '/trust'],
  ['mobile', 'Mobile apps', 'Use Lextures on supported mobile devices, control notifications, prepare material for intermittent connectivity, and troubleshoot access.', 14, '/platform'],
  ['self-hosting', 'Self-hosting', 'Deploy and maintain a Lextures installation, including prerequisites, configuration, upgrades, backups, and operational checks.', 15, '/get-started'],
  ['compliance', 'Privacy & compliance', 'Understand how Lextures supports privacy obligations, data requests, retention choices, and institutional review without overstating certification.', 16, '/trust'],
].map(([id, title, description, order, platformPath]) => ({
  id: String(id), title: String(title), description: String(description), order: Number(order), platformPath: String(platformPath),
}))

export const HELP_CATEGORY_IDS = new Set(HELP_CATEGORIES.map(category => category.id))

export function getHelpCategory(id: string): HelpCategory | undefined {
  return HELP_CATEGORIES.find(category => category.id === id)
}
