/** Feature flag for per-student progress dashboards (plan 9.1). */
export function studentProgressFeatureEnabled(): boolean {
  return import.meta.env.VITE_FEATURE_STUDENT_PROGRESS === 'true'
}

export const studentProgressI18n = {
  submitted: 'Assignments submitted',
  missing: 'Missing or late',
  lastActive: 'Last active',
  avgScore: 'Average grade',
  notes: 'Instructor notes',
  lastUpdated: 'Last updated',
  modulesViewed: 'Modules viewed',
  avgQuiz: 'Average quiz score',
  progressTitle: 'Student progress',
  myProgressTitle: 'My progress',
  tabOverview: 'Overview',
  tabActivity: 'Activity',
  tabAssignments: 'Assignments',
  tabQuizzes: 'Quizzes',
  tabNotes: 'Notes',
} as const
