import { Smartphone } from 'lucide-react'
import {
  IOS_APP_LINK_LABEL,
  IOS_APP_STORE_URL,
  MOBILE_LEARNING_NOTE,
} from '../../lib/mobile-apps'
import { SideNavTooltip } from './side-nav-tooltip'

const linkClass =
  'inline-flex min-h-6 items-center rounded-lg font-medium text-fg-default underline-offset-2 hover:underline focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-solid/40'

/**
 * App Store path plus the mobile-web note. Shown in the shell footer on the
 * dashboard and on course pages (same sidebar). Android stays out until a
 * store listing exists.
 */
export function MobileAppPath({ collapsed }: { collapsed: boolean }) {
  if (collapsed) {
    return (
      <SideNavTooltip content={IOS_APP_LINK_LABEL}>
        <a
          href={IOS_APP_STORE_URL}
          target="_blank"
          rel="noopener noreferrer"
          aria-label={`${IOS_APP_LINK_LABEL} on the App Store. ${MOBILE_LEARNING_NOTE}`}
          className="mb-1 flex h-9 w-9 min-h-6 min-w-6 items-center justify-center rounded-lg text-fg-muted motion-safe:transition-colors hover:bg-surface-raised hover:text-fg-default focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-solid/40"
        >
          <Smartphone className="h-5 w-5" aria-hidden="true" />
        </a>
      </SideNavTooltip>
    )
  }

  return (
    <div className="mb-2 flex flex-col gap-1 px-0.5">
      <p>{MOBILE_LEARNING_NOTE}</p>
      <a href={IOS_APP_STORE_URL} target="_blank" rel="noopener noreferrer" className={linkClass}>
        {IOS_APP_LINK_LABEL}
        <span className="font-normal text-fg-muted"> · App Store</span>
        <span className="sr-only"> (opens in a new tab)</span>
      </a>
    </div>
  )
}
