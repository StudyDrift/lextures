/** Short family checklist shown for Homeschool, K–12, and managed-learner courses (#654). */

export const FAMILY_CHECKLIST_CATEGORY = 'family'

export type FamilyChecklistHelp = {
  title: string
  what: string
  why: string
  how: string
  whenToDismiss: string
}

const help: Record<string, FamilyChecklistHelp> = {
  'structure.modules-exist': {
    title: 'Set up a learning plan',
    what: 'This check looks for at least one module, which is the outline of lessons for the course.',
    why: 'A short list of lessons gives your family a path through the course.',
    how: 'Open Modules and add a module for the lessons you want to teach. Re-check when the outline is in place.',
    whenToDismiss: 'Skip this if the plan lives somewhere else and this course only holds one activity.',
  },
  'people.students-enrolled': {
    title: 'Enroll your kids',
    what: 'This check looks for at least one active learner enrolled in the course.',
    why: 'Add the learners who will take this course.',
    how: 'Open People and enroll or invite a learner. Pending invitations count as in progress.',
    whenToDismiss: 'Skip this if you are still building the course and will add learners later.',
  },
  'assessment.gradable-items': {
    title: 'Add a first lesson or quiz',
    what: 'This check looks for a lesson page, assignment, or quiz.',
    why: 'One lesson or quiz is enough to start.',
    how: 'Open Modules and add a lesson page, assignment, or quiz inside your plan.',
    whenToDismiss: 'Skip this if the first activity is a live conversation you do not track in the course.',
  },
  'launch.student-preview': {
    title: 'Check progress',
    what: 'This check is done when a learner and a lesson or quiz are both in place, so progress has something to show.',
    why: 'See how your kids are doing once they have something to work on.',
    how: 'Enroll a learner, add a lesson or quiz, then open Progress.',
    whenToDismiss: 'Skip this if you record progress outside the course.',
  },
  'structure.pacing-signal': {
    title: 'Set a pacing rhythm',
    what: 'This check looks for a course start and end date, or any due date on a lesson.',
    why: 'Dates or a due date keep the week from drifting.',
    how: 'Open course settings and set start and end dates, or put a due date on a lesson.',
    whenToDismiss: 'Skip this if your family works without dates.',
  },
}

export function isFamilyChecklist(categories: { id: string }[] | null | undefined): boolean {
  return (categories ?? []).some((category) => category.id === FAMILY_CHECKLIST_CATEGORY)
}

export function familyChecklistHelp(itemId: string): FamilyChecklistHelp | null {
  return help[itemId] ?? null
}
