import type { ReactNode } from 'react'
import { ImpersonationBanner } from './ImpersonationBanner'
import { useImpersonationBannerOffset } from '../hooks/use-impersonation-banner-offset'

type ImpersonationChromeProps = {
  shellClassName: string
  children: ReactNode
}

/** Lazy-loaded shell wrapper that adds the impersonation banner and top offset (plan 18.3). */
export function ImpersonationChrome({ shellClassName, children }: ImpersonationChromeProps) {
  const bannerOffset = useImpersonationBannerOffset()

  return (
    <>
      <ImpersonationBanner />
      <div className={`${shellClassName} ${bannerOffset}`}>{children}</div>
    </>
  )
}
