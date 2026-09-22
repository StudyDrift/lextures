export type SubmissionWorkbenchPanel = 'document' | 'media'

/** Which submission tab to open. Media feedback is instructor media, so a text, file, or URL submission stays on the document tab. */
export function preferredSubmissionPanel(input: {
  annotationsActive: boolean
  feedbackMediaEnabled: boolean
  submissionAllowsFile: boolean
  submissionAllowsText: boolean
  submissionAllowsUrl: boolean
}): SubmissionWorkbenchPanel {
  const documentSurface =
    input.annotationsActive ||
    input.submissionAllowsFile ||
    input.submissionAllowsText ||
    input.submissionAllowsUrl
  if (!documentSurface && input.feedbackMediaEnabled) return 'media'
  return 'document'
}

/** Students submit under Text Entry. Staff file markup keeps the Annotations label. */
export function submissionDocumentTabLabel(input: {
  mode: 'staff' | 'student'
  annotationsActive: boolean
}): string {
  if (input.mode === 'staff' && input.annotationsActive) return 'Annotations'
  return 'Text Entry'
}

/** Students are not allowed to read an unposted grade. That is an empty state, not an error. */
export function studentGradeIsHidden(message: string): boolean {
  return /do not have permission to view grades/i.test(message)
}
