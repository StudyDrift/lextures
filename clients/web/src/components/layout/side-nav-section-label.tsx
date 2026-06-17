import type { ReactNode } from 'react'
import { useShellNav } from './use-shell-nav'

type SideNavSectionLabelProps = {
  children: ReactNode
  /** Extra top padding when this is the first labeled group in a nav panel. */
  first?: boolean
}

export function SideNavSectionLabel({ children, first }: SideNavSectionLabelProps) {
  const { sideNavCollapsed } = useShellNav()
  if (sideNavCollapsed) return null
  return (
    <p
      className={`px-3 pb-1 text-sm font-bold tracking-tight text-slate-900 dark:text-neutral-100 ${
        first ? 'pt-3' : 'pt-4'
      }`}
    >
      {children}
    </p>
  )
}