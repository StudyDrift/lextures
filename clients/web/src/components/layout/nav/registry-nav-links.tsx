/**
 * UX.7 — Render a resolved nav model as SideNav links.
 */

import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react'
import { ChevronDown, Pin, PinOff, EyeOff } from 'lucide-react'
import {
  emitNavTelemetry,
  navIcon,
  preferenceScopeFor,
  pushRecentDestination,
  readRecentDestinationIds,
  resolveNavModel,
  type NavAudience,
  type NavScopeKind,
  type ResolvedNavItem,
  type ResolvedNavModel,
} from '../../../lib/nav'
import { useNavPreferences } from '../../../context/nav-preferences-context'
import { usePlatformFeatures } from '../../../context/platform-features-context'
import { usePermissions } from '../../../context/use-permissions'
import { SideNavLink } from '../side-nav-link'
import { SideNavSectionLabel } from '../side-nav-section-label'
import { sideNavActiveClass } from '../side-nav-styles'
import { useShellNav } from '../use-shell-nav'
import { Button } from '../../ui/button'

export type RegistryNavLinksProps = {
  scope: NavScopeKind
  courseCode?: string
  audience: NavAudience
  /** Extra platform flags (e.g. enrollment visibility helpers). */
  platformExtras?: Record<string, unknown>
  courseFeatures?: Record<string, unknown>
  /** Optional badge/tooltip overrides by destination id. */
  itemExtras?: Record<
    string,
    { badge?: ReactNode; tooltip?: string; 'data-onboarding'?: string }
  >
  /** Footer slot (customise button). */
  footer?: ReactNode
}

function platformRecord(features: ReturnType<typeof usePlatformFeatures>): Record<string, unknown> {
  return features as unknown as Record<string, unknown>
}

function LinkRow({
  item,
  extras,
  source,
  preferenceScope,
  courseCode,
  onNavigate,
}: {
  item: ResolvedNavItem
  extras?: { badge?: ReactNode; tooltip?: string; 'data-onboarding'?: string }
  source: 'sidebar' | 'pinned' | 'recent' | 'more'
  preferenceScope: string
  courseCode?: string
  onNavigate: (id: string) => void
}) {
  return (
    <SideNavLink
      to={item.href}
      end={item.dest.end}
      icon={navIcon(item.dest.icon)}
      badge={extras?.badge}
      tooltip={extras?.tooltip ?? item.label}
      data-onboarding={extras?.['data-onboarding']}
      data-nav-id={item.dest.id}
      onClick={() => {
        onNavigate(item.dest.id)
        emitNavTelemetry('nav_item_click', {
          destinationId: item.dest.id,
          scope: preferenceScope,
          source,
        })
      }}
      className={
        item.dest.activePathPrefix
          ? () => {
              if (typeof window === 'undefined') return ''
              const prefix = item.dest.activePathPrefix!.includes(':courseCode')
                ? item.dest.activePathPrefix!.split(':courseCode').join(
                    encodeURIComponent(courseCode ?? ''),
                  )
                : item.dest.activePathPrefix!
              const path = window.location.pathname
              const on = path === prefix || path.startsWith(`${prefix}/`)
              return on ? sideNavActiveClass : ''
            }
          : undefined
      }
    >
      {item.label}
    </SideNavLink>
  )
}

function SectionBlock({
  label,
  first,
  collapsed,
  onToggle,
  children,
  more,
  moreOpen,
  onMoreToggle,
}: {
  label: string
  first?: boolean
  collapsed: boolean
  onToggle: () => void
  children: ReactNode
  more?: ReactNode
  moreOpen?: boolean
  onMoreToggle?: () => void
}) {
  const { sideNavCollapsed } = useShellNav()
  if (sideNavCollapsed) {
    return <>{children}</>
  }
  return (
    <div className="flex flex-col gap-1">
      <button
        type="button"
        className={`flex w-full items-center justify-between rounded-lg px-3 text-start text-sm font-bold tracking-tight text-fg-default outline-none hover:bg-white/40 focus-visible:ring-2 focus-visible:ring-slate-400/35 dark:hover:bg-white/5 ${ first ? 'pt-3' : 'pt-4' } pb-1`}
        aria-expanded={!collapsed}
        onClick={onToggle}
      >
        <span>{label}</span>
        <ChevronDown
          className={`h-4 w-4 shrink-0 text-fg-muted transition-transform ${collapsed ? '-rotate-90' : ''}`}
          aria-hidden
        />
      </button>
      {!collapsed ? (
        <>
          {children}
          {more ? (
            <div className="flex flex-col gap-1">
              <button
                type="button"
                className="px-3 py-1.5 text-start text-xs font-semibold text-fg-muted outline-none hover:text-fg-default focus-visible:ring-2 focus-visible:ring-slate-400/35"
                aria-expanded={moreOpen}
                onClick={onMoreToggle}
              >
                {moreOpen ? 'Show less' : 'More'}
              </button>
              {moreOpen ? more : null}
            </div>
          ) : null}
        </>
      ) : null}
    </div>
  )
}

export function RegistryNavLinks({
  scope,
  courseCode,
  audience,
  platformExtras,
  courseFeatures = {},
  itemExtras,
  footer,
}: RegistryNavLinksProps) {
  const { allows, loading: permLoading } = usePermissions()
  const platformFeatures = usePlatformFeatures()
  const prefsApi = useNavPreferences()
  const preferenceScope = preferenceScopeFor(scope, courseCode)
  const prefs = prefsApi.getPrefs(preferenceScope)
  const navigationV2 = platformFeatures.ffNavigationV2 === true

  const [moreOpenBySection, setMoreOpenBySection] = useState<Record<string, boolean>>({})
  const [recentTick, setRecentTick] = useState(0)

  useEffect(() => {
    void prefsApi.ensureLoaded(preferenceScope)
  }, [prefsApi, preferenceScope])

  const recentIds = useMemo(() => {
    void recentTick
    return readRecentDestinationIds(preferenceScope)
  }, [preferenceScope, recentTick])

  const model: ResolvedNavModel = useMemo(() => {
    const platform = {
      ...platformRecord(platformFeatures),
      ...platformExtras,
      _recentIds: recentIds,
    }
    return resolveNavModel({
      scope,
      preferenceScope,
      courseCode,
      audience,
      allows,
      permLoading,
      platform,
      courseFeatures,
      navigationV2,
      prefs,
    })
  }, [
    scope,
    preferenceScope,
    courseCode,
    audience,
    allows,
    permLoading,
    platformFeatures,
    platformExtras,
    courseFeatures,
    navigationV2,
    prefs,
    recentIds,
  ])

  const onNavigate = useCallback(
    (id: string) => {
      pushRecentDestination(preferenceScope, id)
      setRecentTick((t) => t + 1)
    },
    [preferenceScope],
  )

  const renderItem = (item: ResolvedNavItem, source: 'sidebar' | 'pinned' | 'recent' | 'more') => (
    <LinkRow
      key={item.dest.id}
      item={item}
      extras={itemExtras?.[item.dest.id]}
      source={source}
      preferenceScope={preferenceScope}
      courseCode={courseCode}
      onNavigate={onNavigate}
    />
  )

  return (
    <>
      {model.utility.map((item) => renderItem(item, 'sidebar'))}

      {model.pinned.length > 0 ? (
        <>
          <SideNavSectionLabel first>Pinned</SideNavSectionLabel>
          {model.pinned.map((item) => renderItem(item, 'pinned'))}
        </>
      ) : null}

      {navigationV2 && model.recent.length > 0 ? (
        <>
          <SideNavSectionLabel first={model.pinned.length === 0}>Recent</SideNavSectionLabel>
          {model.recent.map((item) => renderItem(item, 'recent'))}
        </>
      ) : null}

      {model.primary.map((item) => renderItem(item, 'sidebar'))}

      {model.sections.map((section, idx) => {
        const first = model.primary.length === 0 && model.pinned.length === 0 && idx === 0
        const moreOpen = moreOpenBySection[section.id] === true
        return (
          <SectionBlock
            key={section.id}
            label={section.label}
            first={first && model.primary.length === 0}
            collapsed={section.collapsed}
            onToggle={() => {
              void prefsApi.toggleCollapsed(preferenceScope, section.id)
              emitNavTelemetry('nav_section_toggle', {
                scope: preferenceScope,
                sectionId: section.id,
                collapsed: !section.collapsed,
              })
            }}
            more={
              section.moreItems.length > 0
                ? section.moreItems.map((item) => renderItem(item, 'more'))
                : undefined
            }
            moreOpen={moreOpen}
            onMoreToggle={() =>
              setMoreOpenBySection((m) => ({ ...m, [section.id]: !moreOpen }))
            }
          >
            {section.items.map((item) => renderItem(item, 'sidebar'))}
          </SectionBlock>
        )
      })}

      {footer}
    </>
  )
}

/** Compact pin/hide controls used in the customise sheet. */
export function NavCustomiseActions({
  scope,
  courseCode,
  destinationId,
  essential,
}: {
  scope: NavScopeKind
  courseCode?: string
  destinationId: string
  essential?: boolean
}) {
  const prefsApi = useNavPreferences()
  const preferenceScope = preferenceScopeFor(scope, courseCode)
  const prefs = prefsApi.getPrefs(preferenceScope)
  const isPinned = prefs.pinned.includes(destinationId)
  const isHidden = prefs.hidden.includes(destinationId)

  return (
    <div className="flex items-center gap-1">
      <Button
        type="button"
        variant="ghost"
        size="sm"
        aria-label={isPinned ? 'Unpin' : 'Pin'}
        onClick={() => {
          if (isPinned) {
            void prefsApi.unpin(preferenceScope, destinationId)
            emitNavTelemetry('nav_pin_remove', { destinationId, scope: preferenceScope })
          } else {
            void prefsApi.pin(preferenceScope, destinationId)
            emitNavTelemetry('nav_pin_add', { destinationId, scope: preferenceScope })
          }
        }}
      >
        {isPinned ? <PinOff className="h-4 w-4" /> : <Pin className="h-4 w-4" />}
      </Button>
      {!essential ? (
        <Button
          type="button"
          variant="ghost"
          size="sm"
          aria-label={isHidden ? 'Show in sidebar' : 'Hide from sidebar'}
          onClick={() => {
            if (isHidden) {
              void prefsApi.unhide(preferenceScope, destinationId)
              emitNavTelemetry('nav_show', { destinationId, scope: preferenceScope })
            } else {
              void prefsApi.hide(preferenceScope, destinationId)
              emitNavTelemetry('nav_hide', { destinationId, scope: preferenceScope })
            }
          }}
        >
          <EyeOff className="h-4 w-4" />
        </Button>
      ) : null}
    </div>
  )
}
