export type EntityLabelInput = {
  /** Resolved display name when identity may be shown. */
  name?: string | null
  /** Stable pseudonym for anonymous contexts (e.g. "Student 3"). */
  pseudonym?: string | null
  /** Neutral fallback when name/pseudonym are unavailable — never a raw ID prefix. */
  fallback: string
}

/** Prefer name → pseudonym → fallback. Never returns a truncated UUID. */
export function formatEntityLabel(input: EntityLabelInput): string {
  const name = input.name?.trim()
  if (name) return name
  const pseudonym = input.pseudonym?.trim()
  if (pseudonym) return pseudonym
  return input.fallback
}