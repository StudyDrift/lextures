import type { ReactNode } from 'react'
import { ImpersonationBanner } from './ImpersonationBanner'

type ImpersonationChromeProps = {
  shellClassName: string
  children: ReactNode
}

/**
 * Lazy-loaded shell wrapper (plan 18.3). The view-as / Learn-as banner sits in normal flow above
 * the app shell, so the header and side nav are pushed down by exactly the banner's height (it can
 * wrap to several lines on a phone) instead of being covered by a fixed overlay.
 */
export function ImpersonationChrome({ shellClassName, children }: ImpersonationChromeProps) {
  return (
    <div className="flex h-dvh min-h-0 flex-col overflow-hidden">
      <ImpersonationBanner />
      <div className={shellClassName}>{children}</div>
    </div>
  )
}
