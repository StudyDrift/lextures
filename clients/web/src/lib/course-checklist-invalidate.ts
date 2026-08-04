type Listener = (courseCode: string) => void

const listeners = new Set<Listener>()

/**
 * Notify the checklist summary provider that course data changed on a known
 * checklist target path (settings, modules, outcomes, enrollments).
 */
export function invalidateChecklist(courseCode: string): void {
  const code = courseCode.trim()
  if (!code) return
  for (const listener of listeners) {
    listener(code)
  }
}

export function subscribeChecklistInvalidation(listener: Listener): () => void {
  listeners.add(listener)
  return () => {
    listeners.delete(listener)
  }
}
