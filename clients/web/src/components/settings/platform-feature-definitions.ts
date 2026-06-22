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
    key: 'ffRevenueShare',
    label: 'Creator revenue share & affiliates',
    description:
      'Creator earnings ledger, affiliate referral links, and Stripe Connect payouts for course sales.',
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
    key: 'ffApiTokens',
    label: 'API access keys',
    description:
      'Personal and institutional API tokens with scoped access for integrations, automation, and MCP agents (plan 16.2).',
  },
  {
    key: 'ffCalendarFeeds',
    label: 'Calendar feeds',
    description:
      'iCal and CalDAV calendar feed subscriptions for assignment and quiz deadlines (plan 16.5).',
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
