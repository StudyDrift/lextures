import type { FeatureHelpTopic } from '../context/feature-help-context'

export type FeatureHelpMedia = {
  /** Bundled or CSP-allowed static asset URL (silent short MP4 preferred). */
  src: string
  /** i18n key in the `common` namespace for the accessible text alternative. */
  altKey: string
}

export const FEATURE_HELP_TITLES: Record<FeatureHelpTopic, string> = {
  gradebook: 'Gradebook',
  modules: 'Modules',
  'question-bank': 'Question bank',
  'quiz-authoring': 'Quiz authoring',
  syllabus: 'Syllabus',
  'content-page': 'Content pages',
}

export const FEATURE_HELP_BODY: Record<FeatureHelpTopic, string> = {
  gradebook:
    'Use arrow keys, Tab, and Enter to move between cells. Double-click to edit scores. Save writes everything to the server; Discard reloads the last saved snapshot. Rubric columns open a structured scoring panel.',
  modules:
    'Drag handles reorder your outline. Archive removes an item from the learner view but keeps it under Course settings → Archived content, where you can restore it.',
  'question-bank':
    'Draft → Active → Retired controls visibility. Each save creates a version you can restore. Use the bank to reuse stems across quizzes and adaptive pools.',
  'quiz-authoring':
    'Edit intro markdown and policies, then use Edit questions for the question list. Preview shows the learner experience before you publish.',
  syllabus:
    'Blocks stack in order; pick a markdown theme for readability. Images upload into the course file store and resolve for everyone enrolled.',
  'content-page':
    'Same block editor as the syllabus: autosave is explicit via Save so you always know when the server has your latest draft.',
}

/** Optional per-feature walkthrough media. Omit the key when no clip is available. */
export const FEATURE_HELP_MEDIA: Partial<Record<FeatureHelpTopic, FeatureHelpMedia>> = {
  modules: {
    src: '/feature-help/modules-walkthrough.mp4',
    altKey: 'featureHelp.media.modules.alt',
  },
}