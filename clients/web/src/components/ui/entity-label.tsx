import { formatEntityLabel, type EntityLabelInput } from '../../lib/format-entity-label'

type EntityLabelProps = EntityLabelInput & {
  className?: string
}

/** Readable entity label — use instead of rendering raw ID prefixes. */
export function EntityLabel({ name, pseudonym, fallback, className }: EntityLabelProps) {
  return <span className={className}>{formatEntityLabel({ name, pseudonym, fallback })}</span>
}