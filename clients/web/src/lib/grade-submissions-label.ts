/**
 * Shared label for the instructor "Grade submissions" button on assignment and quiz pages.
 * `ungradedCount` is the number of students whose work still needs manual grading.
 */
export function gradeSubmissionsLabel(
  ungradedCount: number | null | undefined,
  enrolledCount: number | null | undefined,
): string {
  if (ungradedCount == null) return 'Grade submissions'
  if (ungradedCount <= 0) return 'Grade submissions · all graded'
  if (enrolledCount != null && enrolledCount >= ungradedCount) {
    return `Grade submissions · ${ungradedCount} of ${enrolledCount} to grade`
  }
  return `Grade submissions · ${ungradedCount} to grade`
}
