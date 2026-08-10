/**
 * UX.7 — Customise navigation sheet (pin / hide / reset).
 */

import { useEffect, useMemo, useState } from 'react'
import { SlidersHorizontal } from 'lucide-react'
import {
  destinationsForScope,
  emitNavTelemetry,
  navIcon,
  preferenceScopeFor,
  type NavScopeKind,
} from '../../../lib/nav'
import { useNavPreferences } from '../../../context/nav-preferences-context'
import { useShellNav } from '../use-shell-nav'
import { SideNavTooltip } from '../side-nav-tooltip'
import { Button } from '../../ui/button'
import { Dialog } from '../../ui/dialog'
import { NavCustomiseActions } from './registry-nav-links'

export function NavCustomiseSheetTrigger({
  scope,
  courseCode,
}: {
  scope: NavScopeKind
  courseCode?: string
}) {
  const { sideNavCollapsed } = useShellNav()
  const [open, setOpen] = useState(false)
  const prefsApi = useNavPreferences()
  const preferenceScope = preferenceScopeFor(scope, courseCode)
  const prefs = prefsApi.getPrefs(preferenceScope)

  const destinations = useMemo(
    () => destinationsForScope(scope).filter((d) => !d.utility),
    [scope],
  )

  useEffect(() => {
    if (open) void prefsApi.ensureLoaded(preferenceScope)
  }, [open, prefsApi, preferenceScope])

  return (
    <>
      <div className={`mt-2 ${sideNavCollapsed ? 'flex justify-center' : ''}`}>
        <SideNavTooltip content="Customise navigation">
          <Button
            type="button"
            variant="ghost"
            aria-label="Customise navigation"
            onClick={() => setOpen(true)}
            className={`flex items-center gap-2 rounded-xl px-3 py-2 text-sm font-medium text-fg-muted hover:bg-white/50 hover:text-fg-default dark:hover:bg-white/5 ${sideNavCollapsed ? 'h-10 w-10 justify-center px-0' : 'w-full'}`}
          >
            <SlidersHorizontal className="h-4 w-4 shrink-0" aria-hidden />
            {!sideNavCollapsed ? <span>Customise</span> : null}
          </Button>
        </SideNavTooltip>
      </div>

      <Dialog
        open={open}
        onClose={() => setOpen(false)}
        title={'Customise navigation'}
        description={
          'Pin destinations you use often, hide the rest. Hidden items stay available in search.'
        }
        size="md"
        footer={
          <div className="flex justify-end gap-2">
            <Button
              type="button"
              variant="secondary"
              onClick={() => {
                void prefsApi.reset(preferenceScope)
                emitNavTelemetry('nav_reset', { scope: preferenceScope })
              }}
            >
              Reset to default
            </Button>
            <Button type="button" variant="primary" onClick={() => setOpen(false)}>
              Done
            </Button>
          </div>
        }
      >
        <div className="flex max-h-[min(24rem,60vh)] flex-col gap-1 overflow-y-auto pe-1">
          {destinations.map((d) => {
            const isHidden = prefs.hidden.includes(d.id)
            const isPinned = prefs.pinned.includes(d.id)
            return (
              <div
                key={d.id}
                className={`flex items-center gap-2 rounded-lg border border-border-subtle px-2 py-1.5 ${ isHidden ? 'opacity-60' : '' }`}
              >
                <span className="flex h-5 w-5 shrink-0 items-center justify-center text-fg-muted">
                  {navIcon(d.icon, 'h-4 w-4')}
                </span>
                <span className="min-w-0 flex-1 truncate text-sm text-fg-default">
                  {d.label}
                  {isPinned ? (
                    <span className="ms-1 text-xs text-fg-muted">(pinned)</span>
                  ) : null}
                  {isHidden ? (
                    <span className="ms-1 text-xs text-fg-muted">(hidden)</span>
                  ) : null}
                </span>
                <NavCustomiseActions
                  scope={scope}
                  courseCode={courseCode}
                  destinationId={d.id}
                  essential={d.essential}
                />
              </div>
            )
          })}
        </div>
      </Dialog>
    </>
  )
}
