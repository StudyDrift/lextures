import { useMemo } from 'react'
import { useInboxUnreadCount } from '../../context/use-inbox-unread'
import { RegistryNavLinks } from './nav/registry-nav-links'
import { NavCustomiseSheetTrigger } from './nav/customise-nav-sheet'

/**
 * Global shell destinations — rendered from the UX.7 navigation registry.
 */
export function SideNavMainLinks() {
  const unreadInboxCount = useInboxUnreadCount()

  const itemExtras = useMemo(() => {
    const unreadBadge =
      unreadInboxCount > 0 ? (
        <span
          className="inline-flex min-h-5 min-w-5 shrink-0 items-center justify-center rounded-full bg-red-600 px-1.5 text-[11px] font-semibold tabular-nums leading-none text-white"
          aria-label={`${unreadInboxCount} unread`}
        >
          {unreadInboxCount > 99 ? '99+' : unreadInboxCount}
        </span>
      ) : undefined
    return {
      'global.inbox': {
        badge: unreadBadge,
        'data-onboarding': 'nav-inbox',
      },
      'global.settings': {
        'data-onboarding': 'nav-settings',
      },
    }
  }, [unreadInboxCount])

  return (
    <RegistryNavLinks
      scope="global"
      audience="any"
      itemExtras={itemExtras}
      footer={<NavCustomiseSheetTrigger scope="global" />}
    />
  )
}
