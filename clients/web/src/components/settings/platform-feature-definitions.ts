import type { PlatformSettingsPayload } from './platform-settings-types'

export type PlatformBooleanFeatureKey = {
  [K in keyof PlatformSettingsPayload]: PlatformSettingsPayload[K] extends boolean ? K : never
}[keyof PlatformSettingsPayload]

/** Groups the visible Global platform toggles into operator-facing capability packs. */
export type PlatformFeaturePack =
  | 'core'
  | 'compliance'
  | 'k12'
  | 'higherEd'
  | 'marketplace'
  | 'ai'
  | 'integrations'
  | 'accessibility'
  | 'admin'

export type PlatformFeatureDefinition = {
  key: PlatformBooleanFeatureKey
  label: string
  description: string
  sourceKey?: keyof PlatformSettingsPayload['sources']
  pack?: PlatformFeaturePack
  /** When set, UI shows as credential-gated / auto-derived (DERIVE set). */
  deriveFrom?: string
}

const PLATFORM_FEATURE_DEFINITIONS_UNSORTED: PlatformFeatureDefinition[] = [
  {
    key: 'accommodationsEngineEnabled',
    label: 'Accommodations engine',
    description:
      'Apply IEP/504 accommodation profiles to quizzes, assignments, and timed activities. Decisions are always recorded in the audit log.',
    pack: 'accessibility',
  },
  {
    key: 'sessionManagementUiEnabled',
    label: 'Active sessions UI',
    description: 'Let users view and revoke their own active login sessions from account settings.',
    pack: 'core',
  },
  {
    key: 'annotationEnabled',
    label: 'Annotations',
    description: 'Inline highlighting and notes on course content and submissions.',
    sourceKey: 'annotationEnabled',
    pack: 'core',
  },
  {
    key: 'avScanningEnabled',
    label: 'Antivirus scanning',
    description: 'Scan uploaded files with ClamAV before they are stored or served.',
    pack: 'integrations',
    deriveFrom: 'Reachable clamd instance (or dev stub)',
  },
  {
    key: 'atRiskAlertsEnabled',
    label: 'At-risk early-warning alerts',
    description: 'Surface engagement and grade signals so instructors can intervene early.',
    pack: 'core',
  },
  {
    key: 'blindGradingEnabled',
    label: 'Blind grading',
    description: 'Hide student identity from graders until scores are released.',
    sourceKey: 'blindGradingEnabled',
    pack: 'core',
  },
  {
    key: 'ffBookstoreIntegration',
    label: 'Bookstore / textbook integration',
    description:
      'VitalSource and RedShelf Inclusive Access deep links, opt-out banner, and launch analytics.',
    pack: 'higherEd',
  },
  {
    key: 'outcomesReportEnabled',
    label: 'Course outcomes report',
    description: 'Aggregate mastery and coverage reports across course learning outcomes.',
    pack: 'core',
  },
  {
    key: 'ffWebhooks',
    label: 'Outbound webhooks',
    description: 'Let org admins register HTTPS endpoints for signed grade, enrollment, and submission events.',
    pack: 'integrations',
  },
  {
    key: 'adminConsoleEnabled',
    label: 'Admin console',
    description:
      'Enables the org admin console at /admin with user/course management, settings, and audit log for org_admin and global admin users.',
    pack: 'admin',
  },
  {
    key: 'impersonationEnabled',
    label: 'Admin impersonation',
    description:
      'Allows org admins and global admins to view the application as a specific user (read-only) with a persistent banner and audit trail.',
    pack: 'admin',
  },
  {
    key: 'bulkCsvImportEnabled',
    label: 'Bulk user CSV import',
    description:
      'Enables org admins to upload CSV files to create, update, or deactivate users in bulk.',
    pack: 'admin',
  },
  {
    key: 'adminSearchEnabled',
    label: 'Admin org-wide search',
    description:
      'Enables cross-course search for org admins across users, courses, and content within their organization.',
    pack: 'admin',
  },
  {
    key: 'customFieldsEnabled',
    label: 'Custom fields',
    description: 'Org metadata on users, courses, and enrollments (18.7).',
    pack: 'admin',
  },
  {
    key: 'seatManagementEnabled',
    label: 'Seat license management',
    description: 'Org seat limits, utilization dashboards, and super-admin license management (18.8).',
    pack: 'admin',
  },
  {
    key: 'emailTemplateEditorEnabled',
    label: 'Email template editor',
    description:
      'Enables org admins to customize transactional email templates with merge fields, preview, and version history.',
    pack: 'admin',
  },
  {
    key: 'maintenanceBannerEnabled',
    label: 'Maintenance banners',
    description:
      'Enables site-wide and org-scoped maintenance/outage banners with admin publishing and Statuspage webhook integration.',
    pack: 'admin',
  },
  {
    key: 'ffZapierConnector',
    label: 'Zapier / Make connector',
    description: 'Enable REST-hook webhook subscriptions from Zapier and Make.com automation platforms.',
    pack: 'integrations',
  },
  {
    key: 'ffTranscripts',
    label: 'Transcripts',
    description:
      'Academic transcript preview and issuance (PDF + PESC XML), plus optional institution webhook delivery requests.',
    pack: 'higherEd',
  },
  {
    key: 'ffTranscriptInbound',
    label: 'Transcript inbound intake',
    description:
      'Receive transcripts from other institutions (PESC/PDF), parse and match to applicants, and review them in the registrar intake queue.',
    pack: 'higherEd',
  },
  {
    key: 'ffDiplomas',
    label: 'Diplomas & certificates',
    description:
      'Diploma and certificate templates, issuance, learner wallet display, and public credential verification (T11).',
    pack: 'higherEd',
  },
  {
    key: 'ffAdvisingIntegration',
    label: 'Advising integration',
    description:
      'Degree progress widget, advising appointment links, and advisor notes on the student dashboard.',
    pack: 'higherEd',
  },
  {
    key: 'ffResearchConsent',
    label: 'Research / IRB consent',
    description:
      'IRB consent studies: present consent forms to students, record decisions, and gate research data export to consenting participants.',
    pack: 'compliance',
  },
  {
    key: 'ffAccessibilityIntake',
    label: 'Accessibility services intake',
    description:
      'Accommodation profiles managed by accessibility coordinators, propagated automatically to assessment overrides; instructors see only that an accommodation is active.',
    pack: 'compliance',
  },
  {
    key: 'ffLearningPaths',
    label: 'Learning paths / bundles',
    description:
      'Let creators build ordered multi-course specializations with bundle pricing, path enrollment, and learner progress tracking.',
    pack: 'marketplace',
  },
  {
    key: 'ffConditionalRelease',
    label: 'Conditional release & module requirements',
    description:
      'Rule-based module gating: per-item completion requirements, module prerequisites, date unlocks, and instructor progress reports.',
    pack: 'core',
  },
  {
    key: 'ffPeerReview',
    label: 'Peer review & assessment',
    description:
      'Let instructors configure peer review on assignments with anonymous rubric reviews, allocation, and optional grade blending.',
    pack: 'core',
  },
  {
    key: 'ffCompletionCredentials',
    label: 'Completion certificates (Open Badges)',
    description:
      'Issue verifiable Open Badges 3.0 certificates with PDF download and public verification when learners complete courses or paths.',
    pack: 'marketplace',
  },
  {
    key: 'ffCompetencyBadges',
    label: 'Competency micro-badges',
    description:
      'Let instructors define and award signed Open Badges for learning outcomes, with public learner backpack pages and independent verification.',
    pack: 'marketplace',
  },
  {
    key: 'badgesDefaultPublic',
    label: 'New badges public by default',
    description:
      'When competency badges are enabled, newly awarded badges default to public (learners can still make them private). Minors stay private.',
    pack: 'marketplace',
  },
  {
    key: 'ffConsortiumSharing',
    label: 'Consortium course sharing',
    description:
      'Multi-campus consortium agreements, cross-institutional enrollment, and partner course browse for shared online programs.',
    pack: 'higherEd',
  },
  {
    key: 'ffStripeBilling',
    label: 'Stripe billing (self-learner)',
    description:
      'Stripe Checkout for course purchases and subscriptions, entitlement gating, and learner billing portal.',
    pack: 'marketplace',
    deriveFrom: 'Stripe secret key + webhook secret',
  },
  {
    key: 'ffPaymentsEnabled',
    label: 'Payment provider abstraction',
    description:
      'Multi-provider checkout (Stripe + PayPal), transaction history, async webhooks, and admin refunds.',
    pack: 'marketplace',
    deriveFrom: 'Stripe or PayPal provider credentials',
  },
  {
    key: 'ffRedisCache',
    label: 'Redis object cache',
    description:
      'Cache hot read paths (course structure, enrollments, public catalog, calendar feeds) in shared Redis.',
    pack: 'integrations',
    deriveFrom: 'REDIS_URL configured',
  },
  {
    key: 'ffRevenueShare',
    label: 'Creator revenue share & affiliates',
    description:
      'Creator earnings ledger, affiliate referral links, and Stripe Connect payouts for course sales.',
    pack: 'marketplace',
    deriveFrom: 'Stripe Connect configuration',
  },
  {
    key: 'ffTaxCollection',
    label: 'Tax collection (Stripe Tax)',
    description:
      'Calculate and collect sales tax, VAT, and GST at checkout; issue tax-compliant invoices and jurisdiction reports.',
    pack: 'marketplace',
    deriveFrom: 'Stripe Tax configuration',
  },
  {
    key: 'ffCoCurricularTranscript',
    label: 'Co-curricular transcript (CLR)',
    description:
      'Let students generate IMS CLR v2.0 comprehensive learner records with W3C verifiable credentials and public verification links.',
    pack: 'higherEd',
  },
  {
    key: 'ffCeuTracking',
    label: 'CEU seat-time tracking',
    description:
      'Track contact hours on module content, issue CEU certificates when thresholds are met, and provide CE transcripts.',
    pack: 'higherEd',
  },
  {
    key: 'ffEportfolio',
    label: 'ePortfolio / capstone artifacts',
    description:
      'Let students curate cross-course evidence into shareable ePortfolios with public links and rubric evaluation.',
    pack: 'higherEd',
  },
  {
    key: 'equationEditorEnabled',
    label: 'Equation editor',
    description: 'WYSIWYG math editor for pages, discussions, and rich-text fields.',
    pack: 'core',
  },
  {
    key: 'feedbackMediaEnabled',
    label: 'Feedback media',
    description: 'Let instructors attach audio or video feedback on submissions.',
    sourceKey: 'feedbackMediaEnabled',
    pack: 'core',
  },
  {
    key: 'gradePostingPoliciesEnabled',
    label: 'Grade posting policies',
    description: 'Control when assignment and quiz grades become visible to students.',
    sourceKey: 'gradePostingPoliciesEnabled',
    pack: 'core',
  },
  {
    key: 'gradebookCsvEnabled',
    label: 'Gradebook CSV export',
    description: 'Export the course gradebook as a downloadable CSV file.',
    sourceKey: 'gradebookCsvEnabled',
    pack: 'core',
  },
  {
    key: 'h5pEnabled',
    label: 'Interactive H5P content',
    description: 'Embed interactive H5P packages as module items and track attempt results.',
    pack: 'integrations',
  },
  {
    key: 'scormIngestionEnabled',
    label: 'SCORM / cmi5 ingestion',
    description: 'Upload SCORM 1.2 packages as module items with grade and progress tracking.',
    pack: 'integrations',
  },
  {
    key: 'selfReflectionEnabled',
    label: 'Learner self-reflection & coaching',
    description: 'Prompt learners to reflect on progress and receive lightweight coaching nudges.',
    pack: 'integrations',
  },
  {
    key: 'learnerProfileEnabled',
    label: 'Learner profile',
    description:
      'Autonomous cross-course learner profile with provenance-backed facets (LP01 foundation).',
    pack: 'core',
  },
  {
    key: 'lpAdaptRecommendationsEnabled',
    label: 'Learner profile adaptation',
    description:
      'Use the learner profile to personalize suggested next steps, spaced-repetition review priority, preferred content modality, and AI tutor scaffolding (LP09).',
    pack: 'ai',
  },
  {
    key: 'introCourseEnabled',
    label: 'Intro course ("Welcome to Lextures")',
    description:
      'Auto-enroll every new user as a student in the guided intro course. On by default.',
    pack: 'core',
  },
  {
    key: 'ltiEnabled',
    label: 'LTI',
    description: 'Launch external LTI 1.3 tools from course modules and assignments.',
    sourceKey: 'ltiEnabled',
    pack: 'integrations',
    deriveFrom: 'LTI RSA private key + key ID',
  },
  {
    key: 'moderatedGradingEnabled',
    label: 'Moderated grading',
    description: 'Multiple graders score anonymously before a moderator releases a final grade.',
    sourceKey: 'moderatedGradingEnabled',
    pack: 'core',
  },
  {
    key: 'oerLibraryEnabled',
    label: 'OER library search',
    description: 'Search open educational resources when adding module content.',
    pack: 'integrations',
  },
  {
    key: 'oneRosterEnabled',
    label: 'OneRoster API',
    description: 'Expose OneRoster roster and grade sync endpoints for SIS integrations.',
    sourceKey: 'oneRosterEnabled',
    pack: 'integrations',
    deriveFrom: 'OneRoster bearer token',
  },
  {
    key: 'originalityDetectionEnabled',
    label: 'Originality detection',
    description: 'Run similarity checks on student submissions via configured providers.',
    sourceKey: 'originalityDetectionEnabled',
    pack: 'integrations',
  },
  {
    key: 'studentProgressEnabled',
    label: 'Per-student progress dashboards',
    description: 'Show learners and instructors module completion and activity summaries.',
    pack: 'core',
  },
  {
    key: 'itemAnalysisEnabled',
    label: 'Quiz item analysis',
    description: 'Discrimination and difficulty statistics for quiz questions.',
    pack: 'core',
  },
  {
    key: 'graderAgentEnabled',
    label: 'Grader agent',
    description: 'Instructor-authored AI grading agent in SpeedGrader with dry-run and batch runs.',
    pack: 'ai',
  },
  {
    key: 'graderAgentVisionGradingEnabled',
    label: 'Grader agent vision grading',
    description: 'Grade image-only or scanned submissions using a vision-capable grader model.',
    pack: 'ai',
  },
  {
    key: 'codeExecutionEnabled',
    label: 'Code execution',
    description: 'Sandboxed code execution for quiz code questions and the grader agent Code Test Runner node.',
    pack: 'ai',
  },
  {
    key: 'readingLevelEnabled',
    label: 'Reading level adaptation',
    description: 'Adjust displayed reading complexity for supported content types.',
    pack: 'ai',
  },
  {
    key: 'resubmissionWorkflowEnabled',
    label: 'Resubmission workflow',
    description: 'Allow structured resubmissions after instructor feedback on assignments.',
    sourceKey: 'resubmissionWorkflowEnabled',
    pack: 'core',
  },
  {
    key: 'scimEnabled',
    label: 'SCIM 2.0 provisioning',
    description: 'Provision and deprovision users from an external identity system.',
    sourceKey: 'scimEnabled',
    pack: 'integrations',
    deriveFrom: 'SCIM bearer token',
  },
  {
    key: 'speechToTextEnabled',
    label: 'Speech-to-text dictation',
    description: 'Dictation input in editors and response fields where supported.',
    pack: 'integrations',
  },
  {
    key: 'storageQuotasEnabled',
    label: 'Storage quotas',
    description: 'Enforce per-user and per-course upload limits.',
    pack: 'integrations',
  },
  {
    key: 'translationMemoryEnabled',
    label: 'Translation memory',
    description: 'Reuse prior translations when localizing course content.',
    pack: 'integrations',
  },
  {
    key: 'ffWhatifGrades',
    label: 'What-if grades',
    description:
      'Let students model hypothetical scores and projected course grades on My Grades.',
    pack: 'core',
  },
  {
    key: 'ffGradeCurving',
    label: 'Grade curving',
    description:
      'Let instructors curve or scale assignment grades with preview, undo, and audit trail.',
    pack: 'core',
  },
  {
    key: 'ffOnboardingFlow',
    label: 'Self-learner onboarding',
    description:
      'Multi-step onboarding wizard with goal capture, optional diagnostic placement, and Start Here recommendations.',
    pack: 'marketplace',
  },
  {
    key: 'ffAiStudyBuddy',
    label: 'AI study buddy',
    description:
      'Persistent self-learner AI companion with course-grounded answers, memory, and proactive study prompts.',
    pack: 'ai',
  },
  {
    key: 'ffLessonGenerator',
    label: 'AI lesson generator',
    description:
      'Instructor wizard to generate lesson plans, differentiated activities, formative quizzes, and rubrics from learning objectives.',
    pack: 'ai',
  },
  {
    key: 'ffPersistentTutor',
    label: 'Persistent AI tutor',
    description:
      'Named tutor sessions with conversation history, RAG citations, and instructor concept-confusion digests.',
    pack: 'ai',
  },
  {
    key: 'ffAcademicCalendar',
    label: 'Academic calendar',
    description:
      'Institution academic calendar with terms, holidays, and iCal feeds on the dashboard and admin tools.',
    pack: 'higherEd',
  },
  {
    key: 'ffAltTextEnforcement',
    label: 'Alt text enforcement',
    description:
      'Prompt for and, when set to block, enforce alt text on course images before they can be published.',
    pack: 'accessibility',
  },
  {
    key: 'ffApiTokens',
    label: 'API access keys',
    description:
      'Personal and institutional API tokens with scoped access for integrations, automation, and MCP agents.',
    pack: 'integrations',
  },
  {
    key: 'ffBotDiscord',
    label: 'Discord classroom bot',
    description: 'Connect Discord servers for assignment reminders and classroom announcements.',
    pack: 'integrations',
    deriveFrom: 'Discord bot client ID + secret + public key',
  },
  {
    key: 'ffBotSlack',
    label: 'Slack classroom bot',
    description: 'Connect Slack workspaces for assignment reminders and classroom announcements.',
    pack: 'integrations',
    deriveFrom: 'Slack app client ID + secret',
  },
  {
    key: 'ffBotTeams',
    label: 'Microsoft Teams classroom bot',
    description: 'Connect Microsoft Teams for assignment reminders and classroom announcements.',
    pack: 'integrations',
    deriveFrom: 'Microsoft Teams bot client ID + secret',
  },
  {
    key: 'ffBroadcasts',
    label: 'Institution broadcasts',
    description: 'Compose and send institution-wide broadcast messages from the admin console.',
    pack: 'k12',
  },
  {
    key: 'ffCalendarFeeds',
    label: 'Calendar feeds',
    description:
      'iCal and CalDAV calendar feed subscriptions for assignment and quiz deadlines.',
    pack: 'core',
  },
  {
    key: 'ffCatalogIntegration',
    label: 'Course catalog & registration',
    description:
      'Browse and register for catalog courses from the main navigation and learner dashboard.',
    pack: 'higherEd',
  },
  {
    key: 'ffClassroomSignals',
    label: 'Classroom signals',
    description:
      'K-12 classroom engagement widgets on course and admin dashboards.',
    pack: 'k12',
  },
  {
    key: 'ffConferenceScheduling',
    label: 'Parent-teacher conferences',
    description:
      'Schedule parent-teacher conferences from the parent portal and instructor dashboard.',
    pack: 'k12',
  },
  {
    key: 'ffCourseEvaluations',
    label: 'Course evaluations',
    description:
      'Evaluation templates, learner surveys, and institutional evaluation reports.',
    pack: 'higherEd',
  },
  {
    key: 'ffCourseReviews',
    label: 'Course reviews',
    description: 'Learner star ratings and written reviews on self-paced catalog courses.',
    pack: 'marketplace',
  },
  {
    key: 'ffDemographics',
    label: 'Student demographics reporting',
    description: 'Title I and student demographics admin reports for K-12 compliance.',
    pack: 'k12',
  },
  {
    key: 'ffEnrollmentStateMachine',
    label: 'Enrollment lifecycle',
    description:
      'Formal enrollment states (active, dropped, withdrawn) with transitions on the course enrollments page.',
    pack: 'higherEd',
  },
  {
    key: 'ffGamification',
    label: 'Gamification & leaderboards',
    description:
      'Points, badges, and course leaderboards on the dashboard, course home, and learner profile.',
    pack: 'marketplace',
  },
  {
    key: 'ffGradeSubmission',
    label: 'Final grade submission',
    description:
      'Instructor final grade submission workflow and admin grade-submission status reporting.',
    pack: 'higherEd',
  },
  {
    key: 'ffIncompleteGradeWorkflow',
    label: 'Incomplete grade workflow',
    description:
      'Track and resolve incomplete (I) grades from the admin incompletes view and gradebook.',
    pack: 'higherEd',
  },
  {
    key: 'ffLibrary',
    label: 'Learner library & reading log',
    description:
      'Reading log, reading dashboard, and library catalog pages for independent reading programs.',
    pack: 'k12',
  },
  {
    key: 'ffLibraryIntegration',
    label: 'Library system integration',
    description: 'Admin configuration for external library catalog integrations.',
    pack: 'higherEd',
  },
  {
    key: 'ffParentPortal',
    label: 'Parent portal',
    description:
      'K-12 parent/guardian portal with child linking, read-only grade access, notification preferences, attendance, behavior, report cards, and message-teacher actions.',
    pack: 'k12',
  },
  {
    key: 'ffProctoringIntegration',
    label: 'Proctoring integration',
    description:
      'Third-party proctoring launch and session hooks on high-stakes quizzes.',
    pack: 'higherEd',
  },
  {
    key: 'ffPublicApi',
    label: 'Public REST API',
    description: 'Expose the documented public API for external integrations and developer access.',
    pack: 'integrations',
  },
  {
    key: 'ffPublicCatalog',
    label: 'Public course catalog',
    description: 'Marketing-style public catalog browse and course detail pages for open enrollment.',
    pack: 'marketplace',
  },
  {
    key: 'ffCourseMarketplace',
    label: 'Course marketplace',
    description:
      'Let learners discover and enroll in courses through an in-app storefront. Instructors opt individual courses in from course settings. Distinct from the plugin marketplace.',
    pack: 'marketplace',
  },
  {
    key: 'ffFeedback',
    label: 'In-app product feedback',
    description:
      'Let signed-in users submit product feedback from web and mobile clients. Admins triage submissions from the feedback queue.',
    pack: 'core',
  },
  {
    key: 'ffIqPublicKitCatalog',
    label: 'Live quiz public kit catalog',
    description:
      'Enable a curated public catalog of shareable quiz kits. Submissions stay pending until moderated. Org sharing works without this flag.',
    pack: 'marketplace',
  },
  {
    key: 'ffIqGuestJoin',
    label: 'Live quiz guest join',
    description:
      'Allow unauthenticated guest players when a host enables guests for a game. Off by default; blocked for courses with minors (COPPA). Requires nickname moderation (IQ.9).',
    pack: 'k12',
  },
  {
    key: 'ffIqAiGeneration',
    label: 'Live quiz AI generation',
    description:
      'Let instructors draft quiz-kit questions with AI from a topic, passage, or course content. Requires Live Quizzes on the course, configured AI providers, and teacher review before hosting.',
    pack: 'ai',
  },
  {
    key: 'ffBoardsRealtime',
    label: 'Board realtime sync',
    description: 'Y.js WebSocket sync and presence for collaboration boards enabled per-course.',
    pack: 'integrations',
  },
  {
    key: 'ffBoardsExternalSharing',
    label: 'Board external sharing',
    description:
      'Allow unlisted share links and public read-only boards. Off by default; requires VC.7 moderation (approval, filter, lock/freeze). Contribute links honour approval mode and content filtering.',
    pack: 'integrations',
  },
  {
    key: 'ffEmailSes',
    label: 'Amazon SES email provider',
    description:
      'Allow selecting Amazon SES as the transactional email backend (Settings → Global platform → Outgoing email). Disabled by default; SMTP remains available. Other providers can be added later.',
    pack: 'integrations',
    deriveFrom: 'SES region + verified sender identity',
  },
  {
    key: 'ffReadAloud',
    label: 'Read-aloud (text-to-speech)',
    description:
      'Learner read-aloud controls in the top bar when read-aloud is enabled platform-wide.',
    pack: 'accessibility',
  },
  {
    key: 'ffReadingPreferences',
    label: 'Reading preferences',
    description:
      'Learner reading preference controls (font, spacing, contrast) in the top bar.',
    pack: 'accessibility',
  },
  {
    key: 'ffMotionNavigation',
    label: 'Motion / animation',
    description:
      'Kill-switch for splash handoff, route/section transitions, load choreography, list insert/remove/reorder, and overlay enter/exit motion. Turn off to disable all platform motion instantly (AN.2–AN.5).',
    pack: 'accessibility',
  },
  {
    key: 'ffMobileCreateCourse',
    label: 'Mobile create course',
    description:
      'Show the New course entry and create wizard on iOS and Android, including competency authoring, Canvas import entry, and draft resume (M11.5, MOB.1).',
    pack: 'core',
  },
  {
    key: 'ffMobileCanvasImport',
    label: 'Mobile Canvas import',
    description:
      'Import a Canvas course on iOS and Android with credentials, scope toggles, and live progress (MOB.2).',
    pack: 'core',
  },
  {
    key: 'ffMobileAdminConsole',
    label: 'Mobile admin console',
    description:
      'Settings/Admin hub on iOS and Android with web-parity menu groups and audit log (MOB.3).',
    pack: 'admin',
  },
  {
    key: 'ffMobileEnrollmentAdd',
    label: 'Mobile enrollment add',
    description:
      'Add people to a course roster from iOS and Android People, with role selection and state actions (MOB.4).',
    pack: 'core',
  },
  {
    key: 'ffMobileLiveQuiz',
    label: 'Mobile live quiz',
    description:
      'Join and play interactive live quizzes from iOS and Android with join codes, answer surfaces, and leaderboards (MOB.5).',
    pack: 'core',
  },
  {
    key: 'ffMobileWhiteboardEdit',
    label: 'Mobile whiteboard edit',
    description:
      'Create, draw, save, and delete course whiteboards on iOS and Android with web-compatible canvas data (MOB.6).',
    pack: 'core',
  },
  {
    key: 'ffMobileMarketplacePurchase',
    label: 'Mobile marketplace purchases',
    description:
      'Claim free marketplace courses, buy paid courses via Stripe checkout handoff, and browse Purchased courses on iOS and Android (MOB.7).',
    pack: 'marketplace',
  },
  {
    key: 'ffReportCards',
    label: 'Report cards',
    description: 'Standards-based report card generation and distribution for K-12 terms.',
    pack: 'k12',
  },
  {
    key: 'ffSelfPacedMode',
    label: 'Self-paced course mode',
    description: 'Self-paced enrollments, progress tracking, and dashboard sections for catalog learners.',
    pack: 'marketplace',
  },
  {
    key: 'ffSisIntegration',
    label: 'SIS integration',
    description: 'Student information system integration settings and sync tools in admin.',
    pack: 'k12',
  },
  {
    key: 'ffUiMode',
    label: 'UI mode switcher',
    description: 'Let learners switch between simplified and standard interface modes where supported.',
    pack: 'k12',
  },
  {
    key: 'mfaEnabled',
    label: 'Two-factor authentication',
    description: 'Offer TOTP authenticator apps and passkeys as optional login factors.',
    sourceKey: 'mfaEnabled',
    pack: 'core',
  },
  {
    key: 'virtualClassroomEnabled',
    label: 'Virtual classroom',
    description: 'Platform-wide live session tooling used by course live-session features.',
    pack: 'integrations',
  },
  {
    key: 'xapiEmissionEnabled',
    label: 'xAPI / Caliper emission',
    description: 'Emit learning analytics statements to a configured LRS endpoint.',
    pack: 'integrations',
    deriveFrom: 'Configured LRS endpoint',
  },
]

/** Platform boolean flags for Settings → Global platform, sorted alphabetically by label. */
export const PLATFORM_FEATURE_DEFINITIONS = [...PLATFORM_FEATURE_DEFINITIONS_UNSORTED].sort((a, b) =>
  a.label.localeCompare(b.label, undefined, { sensitivity: 'base' }),
)
