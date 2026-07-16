const SETTINGS_SECTION_TITLES: Record<string, string> = {
  general: 'General',
  grading: 'Grading',
  plagiarism: 'Plagiarism',
  outcomes: 'Outcomes',
  features: 'Features',
  accessibility: 'Accessibility',
  translations: 'Translations',
  sections: 'Sections',
  'import-export': 'Import / export',
  blueprint: 'Blueprint',
  archive: 'Archive',
}

const EXACT_SUBPATH_TITLES: Record<string, string> = {
  feed: 'Feed',
  discussions: 'Discussions',
  files: 'Files',
  groups: 'Groups',
  syllabus: 'Syllabus',
  modules: 'Modules',
  diagnostic: 'Placement',
  questions: 'Question bank',
  'misconception-report': 'Misconception report',
  live: 'Live Sessions',
  'office-hours': 'Office Hours',
  notebook: 'Notebook',
  calendar: 'Calendar',
  'my-grades': 'My grades',
  gradebook: 'Gradebook',
  reports: 'Reports',
  'at-risk': 'At-risk students',
  'event-log': 'Event log',
  'standards-gradebook': 'Standards gradebook',
  'standards-coverage': 'Standards coverage',
  'mastery-heatmap': 'Mastery Heatmap',
  'outcomes-report': 'Outcomes report',
  'whats-working': "What's working",
  enrollments: 'Enrollments',
  attendance: 'Attendance',
  behavior: 'Behavior & PBIS',
  'report-cards': 'Report Cards',
  'reading-dashboard': 'Reading Dashboard',
  whiteboard: 'Whiteboard',
  boards: 'Boards',
  'live-quizzes': 'Live Quizzes',
  'final-grades': 'Submit Final Grades',
  evaluation: 'Course Evaluation',
  'evaluation-results': 'Evaluation Results',
}

/** Default browser tab page segment for routes under `/courses/:courseCode`. */
export function coursePageTitleFromPath(pathname: string): string | null {
  const match = pathname.match(/^\/courses\/[^/]+(?:\/(.*))?$/)
  if (!match) return null

  const sub = (match[1] ?? '').replace(/\/+$/, '')
  if (!sub) return 'Course'

  if (sub in EXACT_SUBPATH_TITLES) return EXACT_SUBPATH_TITLES[sub]

  if (sub.startsWith('settings/')) {
    const section = sub.slice('settings/'.length).split('/')[0]
    return SETTINGS_SECTION_TITLES[section] ?? 'Course settings'
  }

  if (/^modules\/content\//.test(sub)) return 'Content page'
  if (/^modules\/assignment\/[^/]+\/moderation$/.test(sub)) return 'Moderated grading'
  if (/^modules\/assignment\//.test(sub)) return 'Assignment'
  if (/^modules\/quiz\/[^/]+\/attempt$/.test(sub)) return 'Quiz attempt'
  if (/^modules\/quiz\//.test(sub)) return 'Quiz'
  if (/^modules\/external-link\//.test(sub)) return 'External link'
  if (/^modules\/h5p\//.test(sub)) return 'Interactive activity'
  if (/^modules\/scorm\//.test(sub)) return 'SCORM activity'
  if (/^modules\/lti\//.test(sub)) return 'LTI tool'
  if (/^modules\/vibe-activity\//.test(sub)) return 'Vibe Activity'
  if (/^modules\/textbook-resource\//.test(sub)) return 'Textbook'

  if (sub === 'collab-docs' || sub.startsWith('collab-docs/')) return 'Collaborative Documents'
  if (sub === 'boards' || sub.startsWith('boards/')) return 'Boards'
  if (sub === 'live-quizzes' || sub.startsWith('live-quizzes/')) return 'Live Quizzes'
  if (/^students\/[^/]+\/progress$/.test(sub)) return 'Student progress'
  if (/^whiteboard\//.test(sub)) return 'Whiteboard'

  return 'Course'
}