export type SearchEntityType = 'course' | 'person' | 'page' | 'action' | 'content' | 'goto' | 'ai'

export type ParsedSearchQuery = {
  /** Free-text tokens after stripping scope/type prefixes. */
  text: string
  /** Lowercase course code from `@code` or `in:code`. */
  scopeCourseCode: string | null
  /** When set, only these entity types are included. */
  types: Set<SearchEntityType> | null
  /** Original trimmed query. */
  raw: string
}

const TYPE_PREFIX_RE = /^type:(course|person|page|action|content)\s+/i
const IN_SCOPE_RE = /^in:([^\s]+)\s+/i
const AT_SCOPE_RE = /^@([^\s]+)\s+/i

function normalizeScope(raw: string): string {
  try {
    return decodeURIComponent(raw.trim()).toLowerCase()
  } catch {
    return raw.trim().toLowerCase()
  }
}

/** Parse command-palette query modifiers (`@bio`, `in:biol101`, `type:person`). */
export function parseSearchQuery(query: string): ParsedSearchQuery {
  let rest = query.trim()
  const types = new Set<SearchEntityType>()
  let scopeCourseCode: string | null = null

  for (;;) {
    const typeMatch = TYPE_PREFIX_RE.exec(rest)
    if (typeMatch) {
      types.add(typeMatch[1]!.toLowerCase() as SearchEntityType)
      rest = rest.slice(typeMatch[0].length).trimStart()
      continue
    }
    const inMatch = IN_SCOPE_RE.exec(rest)
    if (inMatch) {
      scopeCourseCode = normalizeScope(inMatch[1]!)
      rest = rest.slice(inMatch[0].length).trimStart()
      continue
    }
    const atMatch = AT_SCOPE_RE.exec(rest)
    if (atMatch) {
      scopeCourseCode = normalizeScope(atMatch[1]!)
      rest = rest.slice(atMatch[0].length).trimStart()
      continue
    }
    break
  }

  return {
    text: rest.toLowerCase(),
    scopeCourseCode,
    types: types.size > 0 ? types : null,
    raw: query.trim(),
  }
}

export function courseMatchesScope(courseCode: string, scope: string | null): boolean {
  if (!scope) return true
  return courseCode.trim().toLowerCase() === scope
}

export type CoursePickerState = {
  active: boolean
  /** Lowercase filter typed after `@` (empty = show all courses). */
  filter: string
  /** Index of `@` in the raw query (for splicing on select). */
  atIndex: number
}

const INCOMPLETE_AT_SUFFIX_RE = /(?:^|\s)@([^\s]*)$/

/**
 * True while the user is choosing a course scope (`@` or `@partial` without a trailing space).
 * Completed scopes like `@bio ` or `@bio gradebook` are not picker mode.
 */
export function parseCoursePickerState(query: string): CoursePickerState {
  const inactive: CoursePickerState = { active: false, filter: '', atIndex: -1 }
  const trimmed = query
  if (!trimmed.includes('@')) return inactive

  const parsed = parseSearchQuery(query)
  if (parsed.scopeCourseCode !== null) return inactive

  const suffix = INCOMPLETE_AT_SUFFIX_RE.exec(query)
  if (!suffix) return inactive

  const atIndex = query.lastIndexOf('@')
  if (atIndex < 0) return inactive

  return {
    active: true,
    filter: suffix[1]!.toLowerCase(),
    atIndex,
  }
}

/** Replace an in-progress `@filter` suffix with a completed `@courseCode ` scope. */
export function applyCoursePickerSelection(
  query: string,
  atIndex: number,
  courseCode: string,
): string {
  const prefix = query.slice(0, atIndex)
  return `${prefix}@${courseCode} `
}
