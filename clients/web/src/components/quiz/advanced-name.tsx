import type { ReactElement } from 'react'
import { Tooltip } from '../ui'

/** Hover and focus tip that names the power-user control without making it the label. */
export function AdvancedNameTip({
  detail,
  children,
}: {
  detail?: string | null
  children: ReactElement
}) {
  if (!detail) return children
  return (
    <Tooltip content={detail} placement="bottom">
      {children}
    </Tooltip>
  )
}

/** Collapsed disclosure for the technical name of a plain-language control. */
export function AdvancedNameDisclosure({ detail }: { detail?: string | null }) {
  if (!detail) return null
  return (
    <details className="text-xs text-fg-muted">
      <summary className="cursor-pointer font-medium text-fg-default">Advanced</summary>
      <p className="mt-1">{detail}</p>
    </details>
  )
}
