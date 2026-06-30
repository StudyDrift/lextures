import type { PlatformSettingsPayload } from './platform-settings-types'

export type PlatformBooleanFeatureKey = {
  [K in keyof PlatformSettingsPayload]: PlatformSettingsPayload[K] extends boolean ? K : never
}[keyof PlatformSettingsPayload]

export type PlatformFeatureDefinition = {
  key: PlatformBooleanFeatureKey
  label: string
  description: string
  sourceKey?: keyof PlatformSettingsPayload['sources']
}

const PLATFORM_FEATURE_DEFINITIONS_UNSORTED: PlatformFeatureDefinition[] = [
  {
    key: 'ffAccommodationsEngine',
    label: 'Accommodations audit log',
    description: 'Record accommodations engine decisions in the audit log for compliance review.',
  },
  {
    key: 'accommodationsEngineEnabled',
    label: 'Accommodations engine',
    description: 'Apply IEP/504 accommodation profiles to quizzes, assignments, and timed activities.',
  },
  {
    key: 'sessionManagementUiEnabled',
    label: 'Active sessions UI',
    description: 'Let users view and revoke their own active login sessions from account settings.',
  },
  {
    key: 'annotationEnabled',
    label: 'Annotations',
    description: 'Inline highlighting and notes on course content and submissions.',
    sourceKey: 'annotationEnabled',
  },
  {
    key: 'avScanningEnabled',
    label: 'Antivirus scanning',
    description: 'Scan uploaded files with ClamAV before they are stored or served.',
  },
  {
    key: 'atRiskAlertsEnabled',
    label: 'At-risk early-warning alerts',
    description: 'Surface engagement and grade signals so instructors can intervene early.',
  },
  {
    key: 'blindGradingEnabled',
    label: 'Blind grading',
    description: 'Hide student identity from graders until scores are released.',
    sourceKey: 'blindGradingEnabled',
  },
  {
    key: 'ffBookstoreIntegration',
    label: 'Bookstore / textbook integration',
    description:
      'VitalSource and RedShelf Inclusive Access deep links, opt-out banner, and launch analytics.',
  },
  {
    key: 'outcomesReportEnabled',
    label: 'Course outcomes report',
    description: 'Aggregate mastery and coverage reports across course learning outcomes.',
  },
  {
    key: 'ffWebhooks',
    label: 'Outbound webhooks',
    description: 'Let org admins register HTTPS endpoints for signed grade, enrollment, and submission events.',
  },
  {
    key: 'adminConsoleEnabled',
    label: 'Admin console',
    description:
      'Enables the org admin console at /admin with user/course management, settings, and audit log for org_admin and global admin users.',
  },
  {
    key: 'impersonationEnabled',
    label: 'Admin impersonation',
    description:
      'Allows org admins and global admins to view the application as a specific user (read-only) with a persistent banner and audit trail.',
  },
  {
    key: 'bulkCsvImportEnabled',
    label: 'Bulk user CSV import',
    description:
      'Enables org admins to upload CSV files to create, update, or deactivate users in bulk (plan 18.2).',
  },
  {
    key: 'adminSearchEnabled',
    label: 'Admin org-wide search',
    description:
      'Enables cross-course search for org admins across users, courses, and content within their organization.',
  },
  {
    key: 'customFieldsEnabled',
    label: 'Custom fields',
    description: 'Org metadata on users, courses, and enrollments (18.7).',
  },
  {
    key: 'seatManagementEnabled',
    label: 'Seat license management',
    description:
      'Enforces org seat limits, utilization dashboards, and super-admin license management (plan 18.8).',
  },
  {
    key: 'emailTemplateEditorEnabled',
    label: 'Email template editor',
    description:
      'Enables org admins to customize transactional email templates with merge fields, preview, and version history (plan 18.5).',
  },
  {
    key: 'maintenanceBannerEnabled',
    label: 'Maintenance banners',
    description:
      'Enables site-wide and org-scoped maintenance/outage banners with admin publishing and Statuspage webhook integration (plan 18.6).',
  },
  {
    key: 'ffZapierConnector',
    label: 'Zapier / Make connector',
    description: 'Enable REST-hook webhook subscriptions from Zapier and Make.com automation platforms.',
  },
  {
    key: 'ffTranscripts',
    label: 'Transcripts',
    description:
      'Let students request official transcripts from their institution via a configurable webhook.',
  },
  {
    key: 'ffAdvisingIntegration',
    label: 'Advising integration',
    description:
      'Degree progress widget, advising appointment links, and advisor notes on the student dashboard.',
  },
  {
    key: 'ffResearchConsent',
    label: 'Research / IRB consent',
    description:
      'IRB consent studies: present consent forms to students, record decisions, and gate research data export to consenting participants.',
  },
  {
    key: 'ffAccessibilityIntake',
    label: 'Accessibility services intake',
    description:
      'Accommodation profiles managed by accessibility coordinators, propagated automatically to assessment overrides; instructors see only that an accommodation is active.',
  },
  {
    key: 'ffLearningPaths',
    label: 'Learning paths / bundles',
    description:
      'Let creators build ordered multi-course specializations with bundle pricing, path enrollment, and learner progress tracking.',
  },
  {
    key: 'ffConditionalRelease',
    label: 'Conditional release & module requirements',
    description:
      'Rule-based module gating: per-item completion requirements, module prerequisites, date unlocks, and instructor progress reports.',
  },
  {
    key: 'ffPeerReview',
    label: 'Peer review & assessment',
    description:
      'Let instructors configure peer review on assignments with anonymous rubric reviews, allocation, and optional grade blending.',
  },
  {
    key: 'ffCompletionCredentials',
    label: 'Completion certificates (Open Badges)',
    description:
      'Issue verifiable Open Badges 3.0 certificates with PDF download and public verification when learners complete courses or paths.',
  },
  {
    key: 'ffConsortiumSharing',
    label: 'Consortium course sharing',
    description:
      'Multi-campus consortium agreements, cross-institutional enrollment, and partner course browse for shared online programs.',
  },
  {
    key: 'ffStripeBilling',
    label: 'Stripe billing (self-learner)',
    description:
      'Stripe Checkout for course purchases and subscriptions, entitlement gating, and learner billing portal.',
  },
  {
    key: 'ffPaymentsEnabled',
    label: 'Payment provider abstraction',
    description:
      'Multi-provider checkout (Stripe + PayPal), transaction history, async webhooks, and admin refunds (plan 16.8).',
  },
  {
    key: 'ffRedisCache',
    label: 'Redis object cache',
    description:
      'Cache hot read paths (course structure, enrollments, public catalog, calendar feeds) in shared Redis (plan 17.5).',
  },
  {
    key: 'ffRevenueShare',
    label: 'Creator revenue share & affiliates',
    description:
      'Creator earnings ledger, affiliate referral links, and Stripe Connect payouts for course sales.',
  },
  {
    key: 'ffTaxCollection',
    label: 'Tax collection (Stripe Tax)',
    description:
      'Calculate and collect sales tax, VAT, and GST at checkout; issue tax-compliant invoices and jurisdiction reports.',
  },
  {
    key: 'ffCoCurricularTranscript',
    label: 'Co-curricular transcript (CLR)',
    description:
      'Let students generate IMS CLR v2.0 comprehensive learner records with W3C verifiable credentials and public verification links.',
  },
  {
    key: 'ffCeuTracking',
    label: 'CEU seat-time tracking',
    description:
      'Track contact hours on module content, issue CEU certificates when thresholds are met, and provide CE transcripts.',
  },
  {
    key: 'ffEportfolio',
    label: 'ePortfolio / capstone artifacts',
    description:
      'Let students curate cross-course evidence into shareable ePortfolios with public links and rubric evaluation.',
  },
  {
    key: 'equationEditorEnabled',
    label: 'Equation editor',
    description: 'WYSIWYG math editor for pages, discussions, and rich-text fields.',
  },
  {
    key: 'feedbackMediaEnabled',
    label: 'Feedback media',
    description: 'Let instructors attach audio or video feedback on submissions.',
    sourceKey: 'feedbackMediaEnabled',
  },
  {
    key: 'gradePostingPoliciesEnabled',
    label: 'Grade posting policies',
    description: 'Control when assignment and quiz grades become visible to students.',
    sourceKey: 'gradePostingPoliciesEnabled',
  },
  {
    key: 'gradebookCsvEnabled',
    label: 'Gradebook CSV export',
    description: 'Export the course gradebook as a downloadable CSV file.',
    sourceKey: 'gradebookCsvEnabled',
  },
  {
    key: 'h5pEnabled',
    label: 'Interactive H5P content',
    description: 'Embed interactive H5P packages as module items and track attempt results.',
  },
  {
    key: 'scormIngestionEnabled',
    label: 'SCORM / cmi5 ingestion',
    description: 'Upload SCORM 1.2 packages as module items with grade and progress tracking.',
  },
  {
    key: 'selfReflectionEnabled',
    label: 'Learner self-reflection & coaching',
    description: 'Prompt learners to reflect on progress and receive lightweight coaching nudges.',
  },
  {
    key: 'ltiEnabled',
    label: 'LTI',
    description: 'Launch external LTI 1.3 tools from course modules and assignments.',
    sourceKey: 'ltiEnabled',
  },
  {
    key: 'moderatedGradingEnabled',
    label: 'Moderated grading',
    description: 'Multiple graders score anonymously before a moderator releases a final grade.',
    sourceKey: 'moderatedGradingEnabled',
  },
  {
    key: 'oerLibraryEnabled',
    label: 'OER library search',
    description: 'Search open educational resources when adding module content.',
  },
  {
    key: 'oerStub',
    label: 'OER stub catalog',
    description: 'Use a built-in stub OER catalog for local development and end-to-end tests.',
  },
  {
    key: 'oneRosterEnabled',
    label: 'OneRoster API',
    description: 'Expose OneRoster roster and grade sync endpoints for SIS integrations.',
    sourceKey: 'oneRosterEnabled',
  },
  {
    key: 'originalityDetectionEnabled',
    label: 'Originality detection',
    description: 'Run similarity checks on student submissions via configured providers.',
    sourceKey: 'originalityDetectionEnabled',
  },
  {
    key: 'originalityStubExternal',
    label: 'Originality stub external',
    description: 'Simulate external originality provider responses without live API calls.',
    sourceKey: 'originalityStubExternal',
  },
  {
    key: 'studentProgressEnabled',
    label: 'Per-student progress dashboards',
    description: 'Show learners and instructors module completion and activity summaries.',
  },
  {
    key: 'itemAnalysisEnabled',
    label: 'Quiz item analysis',
    description: 'Discrimination and difficulty statistics for quiz questions.',
  },
  {
    key: 'graderAgentEnabled',
    label: 'Grader agent',
    description: 'Instructor-authored AI grading agent in SpeedGrader with dry-run and batch runs.',
  },
  {
    key: 'graderAgentReviewInboxEnabled',
    label: 'Grader agent review inbox',
    description: 'Persistent held/flagged review queue and run history across grading-agent sessions.',
  },
  {
    key: 'graderAgentSuggestModeEnabled',
    label: 'Grader agent suggest mode',
    description: 'Suggest-only batch runs, bulk approve/reject, and posting control for AI grades.',
  },
  {
    key: 'graderAgentTextEntryGradingEnabled',
    label: 'Grader agent text-entry grading',
    description: 'Grade typed online text submissions without a file attachment.',
  },
  {
    key: 'graderAgentVisionGradingEnabled',
    label: 'Grader agent vision grading',
    description: 'Grade image-only or scanned submissions using a vision-capable grader model.',
  },
  {
    key: 'graderAgentRunFiltersEnabled',
    label: 'Grader agent run filters',
    description: 'Target batch runs to a section, project group, or explicit submission selection.',
  },
  {
    key: 'graderAgentCostEstimateEnabled',
    label: 'Grader agent cost estimate',
    description: 'Show approximate batch cost before running and optional per-run budget caps.',
  },
  {
    key: 'graderAgentCancelRunEnabled',
    label: 'Grader agent cancel run',
    description: 'Allow instructors to cancel in-progress grading-agent batch runs.',
  },
  {
    key: 'codeExecutionEnabled',
    label: 'Code execution',
    description: 'Sandboxed code execution for quiz code questions and the grader agent Code Test Runner node.',
  },
  {
    key: 'readingLevelEnabled',
    label: 'Reading level adaptation',
    description: 'Adjust displayed reading complexity for supported content types.',
  },
  {
    key: 'resubmissionWorkflowEnabled',
    label: 'Resubmission workflow',
    description: 'Allow structured resubmissions after instructor feedback on assignments.',
    sourceKey: 'resubmissionWorkflowEnabled',
  },
  {
    key: 'scimEnabled',
    label: 'SCIM 2.0 provisioning',
    description: 'Provision and deprovision users from an external identity system.',
    sourceKey: 'scimEnabled',
  },
  {
    key: 'speechToTextEnabled',
    label: 'Speech-to-text dictation',
    description: 'Dictation input in editors and response fields where supported.',
  },
  {
    key: 'storageQuotasEnabled',
    label: 'Storage quotas',
    description: 'Enforce per-user and per-course upload limits.',
  },
  {
    key: 'translationMemoryEnabled',
    label: 'Translation memory',
    description: 'Reuse prior translations when localizing course content.',
  },
  {
    key: 'ffWhatifGrades',
    label: 'What-if grades',
    description:
      'Let students model hypothetical scores and projected course grades on My Grades (plan 3.16).',
  },
  {
    key: 'ffGradeCurving',
    label: 'Grade curving',
    description:
      'Let instructors curve or scale assignment grades with preview, undo, and audit trail (plan 3.17).',
  },
  {
    key: 'ffOnboardingFlow',
    label: 'Self-learner onboarding',
    description:
      'Multi-step onboarding wizard with goal capture, optional diagnostic placement, and Start Here recommendations (plan 15.11).',
  },
  {
    key: 'ffAiStudyBuddy',
    label: 'AI study buddy',
    description:
      'Persistent self-learner AI companion with course-grounded answers, memory, and proactive study prompts (plan 15.12).',
  },
  {
    key: 'ffAcademicCalendar',
    label: 'Academic calendar',
    description:
      'Institution academic calendar with terms, holidays, and iCal feeds on the dashboard and admin tools (plan 14.6).',
  },
  {
    key: 'ffAltTextEnforcement',
    label: 'Alt-text hard block',
    description:
      'Block publishing course images that lack alt text when alt-text enforcement is enabled (accessibility plan 12.5).',
  },
  {
    key: 'ffApiTokens',
    label: 'API access keys',
    description:
      'Personal and institutional API tokens with scoped access for integrations, automation, and MCP agents (plan 16.2).',
  },
  {
    key: 'ffBotDiscord',
    label: 'Discord classroom bot',
    description: 'Connect Discord servers for assignment reminders and classroom announcements (plan 16.6).',
  },
  {
    key: 'ffBotSlack',
    label: 'Slack classroom bot',
    description: 'Connect Slack workspaces for assignment reminders and classroom announcements (plan 16.6).',
  },
  {
    key: 'ffBotTeams',
    label: 'Microsoft Teams classroom bot',
    description: 'Connect Microsoft Teams for assignment reminders and classroom announcements (plan 16.6).',
  },
  {
    key: 'ffBroadcasts',
    label: 'Institution broadcasts',
    description: 'Compose and send institution-wide broadcast messages from the admin console.',
  },
  {
    key: 'ffCalendarFeeds',
    label: 'Calendar feeds',
    description:
      'iCal and CalDAV calendar feed subscriptions for assignment and quiz deadlines (plan 16.5).',
  },
  {
    key: 'ffCatalogIntegration',
    label: 'Course catalog & registration',
    description:
      'Browse and register for catalog courses from the main navigation and learner dashboard (plan 14.2).',
  },
  {
    key: 'ffClassroomSignals',
    label: 'Classroom signals',
    description:
      'K-12 classroom engagement widgets on course and admin dashboards (plan 13.9).',
  },
  {
    key: 'ffConferenceScheduling',
    label: 'Parent-teacher conferences',
    description:
      'Schedule parent-teacher conferences from the parent portal and instructor dashboard (plan 13.12).',
  },
  {
    key: 'ffCourseEvaluations',
    label: 'Course evaluations',
    description:
      'Evaluation templates, learner surveys, and institutional evaluation reports (plan 14.7).',
  },
  {
    key: 'ffCourseReviews',
    label: 'Course reviews',
    description: 'Learner star ratings and written reviews on self-paced catalog courses.',
  },
  {
    key: 'ffDemographics',
    label: 'Student demographics reporting',
    description: 'Title I and student demographics admin reports for K-12 compliance (plan 13.x).',
  },
  {
    key: 'ffEnrollmentStateMachine',
    label: 'Enrollment lifecycle',
    description:
      'Formal enrollment states (active, dropped, withdrawn) with transitions on the course enrollments page.',
  },
  {
    key: 'ffGamification',
    label: 'Gamification & leaderboards',
    description:
      'Points, badges, and course leaderboards on the dashboard, course home, and learner profile.',
  },
  {
    key: 'ffGradeSubmission',
    label: 'Final grade submission',
    description:
      'Instructor final grade submission workflow and admin grade-submission status reporting (plan 14.x).',
  },
  {
    key: 'ffIncompleteGradeWorkflow',
    label: 'Incomplete grade workflow',
    description:
      'Track and resolve incomplete (I) grades from the admin incompletes view and gradebook (plan 14.4).',
  },
  {
    key: 'ffLibrary',
    label: 'Learner library & reading log',
    description:
      'Reading log, reading dashboard, and library catalog pages for independent reading programs (plan 14.10).',
  },
  {
    key: 'ffLibraryIntegration',
    label: 'Library system integration',
    description: 'Admin configuration for external library catalog integrations (plan 14.10).',
  },
  {
    key: 'ffParentPortal',
    label: 'Parent portal',
    description:
      'K-12 parent/guardian portal with child linking, read-only grade access, and notification preferences (plan 13.1).',
  },
  {
    key: 'ffProctoringIntegration',
    label: 'Proctoring integration',
    description:
      'Third-party proctoring launch and session hooks on high-stakes quizzes (plan 14.9).',
  },
  {
    key: 'ffPublicApi',
    label: 'Public REST API',
    description: 'Expose the documented public API for external integrations and developer access.',
  },
  {
    key: 'ffPublicCatalog',
    label: 'Public course catalog',
    description: 'Marketing-style public catalog browse and course detail pages for open enrollment.',
  },
  {
    key: 'ffReadAloud',
    label: 'Read-aloud (text-to-speech)',
    description:
      'Learner read-aloud controls in the top bar when read-aloud is enabled platform-wide (plan 12.8).',
  },
  {
    key: 'ffReadingPreferences',
    label: 'Reading preferences',
    description:
      'Learner reading preference controls (font, spacing, contrast) in the top bar (plan 12.x).',
  },
  {
    key: 'ffReportCards',
    label: 'Report cards',
    description: 'Standards-based report card generation and distribution for K-12 terms.',
  },
  {
    key: 'ffSelfPacedMode',
    label: 'Self-paced course mode',
    description: 'Self-paced enrollments, progress tracking, and dashboard sections for catalog learners.',
  },
  {
    key: 'ffSisIntegration',
    label: 'SIS integration',
    description: 'Student information system integration settings and sync tools in admin (plan 14.1).',
  },
  {
    key: 'ffUiMode',
    label: 'UI mode switcher',
    description: 'Let learners switch between simplified and standard interface modes where supported.',
  },
  {
    key: 'mfaEnabled',
    label: 'Two-factor authentication',
    description: 'Offer TOTP authenticator apps and passkeys as optional login factors.',
    sourceKey: 'mfaEnabled',
  },
  {
    key: 'virtualClassroomEnabled',
    label: 'Virtual classroom',
    description: 'Platform-wide live session tooling used by course live-session features.',
  },
  {
    key: 'xapiEmissionEnabled',
    label: 'xAPI / Caliper emission',
    description: 'Emit learning analytics statements to a configured LRS endpoint.',
  },
]

/** Platform boolean flags for Settings → Global platform, sorted alphabetically by label. */
export const PLATFORM_FEATURE_DEFINITIONS = [...PLATFORM_FEATURE_DEFINITIONS_UNSORTED].sort((a, b) =>
  a.label.localeCompare(b.label, undefined, { sensitivity: 'base' }),
)
