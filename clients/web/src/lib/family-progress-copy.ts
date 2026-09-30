export type ProgressSurfaceCopy = {
  gradebookDescription: string
  gradebookEmptyTitle: string
  gradebookEmptyInvite: string
  gradebookFilterLabel: string
  gradebookFilterPlaceholder: string
  gradebookNoMatch: (query: string) => string
  gradebookPerson: string
  gradebookPeople: string
  reportsDescription: (courseCode: string) => string
  reportsEmpty: string
  reportsLoading: string
  reportsColumn: string
  outcomesDescription: string
  outcomesNotePlaceholder: string
  outcomesAverageLabel: string
  outcomesLegend: string
  learnerSwitcherLabel: string
  allLearners: string
  openLearnerProgress: string
  chooseLearner: string
}

const campus: ProgressSurfaceCopy = {
  gradebookDescription:
    'Spreadsheet-style grades for enrolled students and each course assignment or quiz. Use the arrows, Tab, Enter, and double-click to edit cells; open the cell menu for rubric scoring, submission grading, history, and excused status. Save writes your changes to the server.',
  gradebookEmptyTitle: 'No students in this course yet',
  gradebookEmptyInvite: 'Invite students from enrollments so the gradebook has rows.',
  gradebookFilterLabel: 'Student',
  gradebookFilterPlaceholder: 'Search by student name…',
  gradebookNoMatch: (query) =>
    `No students match "${query}". Try a different search or clear filters.`,
  gradebookPerson: 'Student',
  gradebookPeople: 'students',
  reportsDescription: (courseCode) =>
    `Student progress reports for course ${courseCode}. Select a student to view their report.`,
  reportsEmpty: 'No students enrolled yet.',
  reportsLoading: 'Loading students…',
  reportsColumn: 'Student',
  outcomesDescription:
    'Cohort achievement on course learning outcomes for accreditation and standards reporting.',
  outcomesNotePlaceholder: 'Qualitative notes for accreditation portfolio…',
  outcomesAverageLabel: 'class avg',
  outcomesLegend:
    'Chart legend: green indicates students who met the mastery threshold; red indicates students who did not.',
  learnerSwitcherLabel: 'Student',
  allLearners: 'All students',
  openLearnerProgress: 'Open a student progress report',
  chooseLearner: 'Choose a student',
}

const family: ProgressSurfaceCopy = {
  gradebookDescription:
    'Scores for each learner on this course’s assignments and quizzes. Pick a learner to focus on one child, then save any score changes.',
  gradebookEmptyTitle: 'No learners in this course yet',
  gradebookEmptyInvite: 'Enroll a learner so their scores show up here.',
  gradebookFilterLabel: 'Learner',
  gradebookFilterPlaceholder: 'Search by learner name…',
  gradebookNoMatch: (query) =>
    `No learners match "${query}". Try a different search or clear filters.`,
  gradebookPerson: 'Learner',
  gradebookPeople: 'learners',
  reportsDescription: (courseCode) =>
    `How each learner is doing in ${courseCode}. Open a card to see the full picture.`,
  reportsEmpty: 'No learners enrolled yet.',
  reportsLoading: 'Loading learners…',
  reportsColumn: 'Learner',
  outcomesDescription:
    'How learners are doing on this course’s learning goals. Pick a learner to open their progress.',
  outcomesNotePlaceholder: 'Notes on what to practice next…',
  outcomesAverageLabel: 'average',
  outcomesLegend:
    'Chart legend: green indicates learners who met the goal; red indicates learners who did not.',
  learnerSwitcherLabel: 'Learner',
  allLearners: 'All learners',
  openLearnerProgress: 'Open a learner’s progress',
  chooseLearner: 'Choose a learner',
}

export function progressSurfaceCopy(familyAudience: boolean): ProgressSurfaceCopy {
  return familyAudience ? family : campus
}
