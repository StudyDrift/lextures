export type AddEnrollmentsResult = {
  added?: string[]
  alreadyEnrolled?: string[]
  notFound?: string[]
}

const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i

function label(value: string, nameByUserId: ReadonlyMap<string, string>): string {
  const name = nameByUserId.get(value)
  if (name) return name
  // Managed learners are reported by user id; never show a raw id to the person adding them.
  return UUID_RE.test(value) ? 'a learner' : value
}

function list(values: readonly string[], nameByUserId: ReadonlyMap<string, string>): string {
  return values.map((v) => label(v, nameByUserId)).join(', ')
}

/**
 * Builds the result line shown after "Add enrollment". Emails are echoed as typed; managed-learner
 * user ids are replaced with the learner's display name.
 */
export function formatAddEnrollmentsMessage(
  data: AddEnrollmentsResult,
  nameByUserId: ReadonlyMap<string, string> = new Map(),
): string {
  const parts: string[] = []
  if (data.added?.length) parts.push(`Added ${list(data.added, nameByUserId)}`)
  if (data.alreadyEnrolled?.length) {
    parts.push(`Already enrolled: ${list(data.alreadyEnrolled, nameByUserId)}`)
  }
  if (data.notFound?.length) parts.push(`No account for: ${list(data.notFound, nameByUserId)}`)
  return parts.length ? `${parts.join('. ')}.` : 'Done.'
}
