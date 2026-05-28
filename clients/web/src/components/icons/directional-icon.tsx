import type { LucideIcon } from 'lucide-react'
import { useDocumentDirection } from '../../i18n/use-document-direction'

type DirectionalIconProps = {
  icon: LucideIcon
  className?: string
  /** When true, mirror in RTL (back arrows, chevrons). */
  mirror?: boolean
}

/**
 * Wraps directional Lucide icons so they mirror in RTL reading direction.
 */
export function DirectionalIcon({ icon: Icon, className, mirror = true }: DirectionalIconProps) {
  const dir = useDocumentDirection()
  const rtlMirror = mirror && dir === 'rtl'
  return (
    <Icon
      className={[className, rtlMirror ? 'rtl-mirror' : ''].filter(Boolean).join(' ')}
      aria-hidden
    />
  )
}
