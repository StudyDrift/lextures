import type { ParentGradeItem } from './parent-api'

export function parentChildLabel(displayName: string | null | undefined, email: string): string {
  const n = displayName?.trim()
  if (n) return n
  return email
}

export function parentGradeScoreLabel(item: ParentGradeItem): string {
  if (item.percentage != null && Number.isFinite(item.percentage)) {
    return `${item.score} (${item.percentage}%)`
  }
  return item.score
}

export function parentMessageTeacherHref(params: {
  teacherEmail: string
  subject: string
}): string {
  const q = new URLSearchParams({
    compose: '1',
    to: params.teacherEmail,
    subject: params.subject,
  })
  return `/inbox?${q.toString()}`
}

export function parentGradeItemsForCourse(
  course: { items?: ParentGradeItem[]; grades?: Record<string, string> },
): ParentGradeItem[] {
  if (course.items && course.items.length > 0) {
    return course.items
  }
  return Object.entries(course.grades ?? {}).map(([itemId, score]) => ({
    itemId,
    title: 'Graded assignment',
    score,
    status: 'graded',
  }))
}
