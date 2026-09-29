/** Subtitle under the Enrollments page H1. Prefer course title over raw course code. */
export function enrollmentsPageSubtitle(
  courseTitle: string | null | undefined,
  courseCode: string | null | undefined,
): string {
  const title = courseTitle?.trim()
  if (title) return `People and roles for ${title}.`
  const code = courseCode?.trim()
  if (code) return `People and roles for course ${code}.`
  return 'Course enrollments'
}
