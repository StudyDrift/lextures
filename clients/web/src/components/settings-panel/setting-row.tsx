/* eslint-disable react-refresh/only-export-components -- component + test-only cache reset */
import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useRef,
  useSyncExternalStore,
  type ReactNode,
  type SyntheticEvent,
} from 'react'
import { createPortal } from 'react-dom'
import { getSettingById } from '../../lib/settings-registry'
import { scheduleSettingsControlChanged } from '../../lib/settings-telemetry'
import {
  getPinHost,
  subscribePinHosts,
} from './pin-host-registry'
import { PinToggle } from './pin-toggle'
import { useSettingsPanelContext } from './settings-panel-context'

const warnedUnknownIds = new Set<string>()

function usePinHost(settingId: string): HTMLElement | null {
  return useSyncExternalStore(
    subscribePinHosts,
    () => getPinHost(settingId),
    () => null,
  )
}

/**
 * Wraps a single addressable settings control.
 * - Applies the panel search filter from context.
 * - Registers the id for pin/relocation layout (PS.3).
 * - When pinned, portals the control into the Pinned group (move-not-duplicate).
 */
export function SettingRow({
  settingId,
  children,
}: {
  settingId: string
  children: ReactNode
}) {
  const { matches, register, pins, surface, telemetryRole } = useSettingsPanelContext()
  const descriptor = getSettingById(settingId)
  const pinToggleRef = useRef<HTMLDivElement>(null)
  const wasPinnedRef = useRef(false)
  const pinHost = usePinHost(settingId)

  useEffect(() => {
    if (!descriptor && import.meta.env.DEV && !warnedUnknownIds.has(settingId)) {
      warnedUnknownIds.add(settingId)
      console.warn(
        `[settings-registry] Unknown settingId "${settingId}" passed to <SettingRow>. ` +
          'Add it to SETTINGS_REGISTRY or fix the prop.',
      )
    }
  }, [descriptor, settingId])

  useEffect(() => register(settingId), [register, settingId])

  const pinEnabled =
    pins?.enabled === true &&
    pins.status !== 'unavailable' &&
    descriptor?.pinnable === true
  const pinned = pinEnabled && pins.isPinned(settingId)

  // Debounced control-change telemetry (PS.4 FR-8/FR-11) — no values recorded.
  const onControlInteraction = useCallback(
    (e: SyntheticEvent) => {
      // Ignore pin-toggle clicks (they have their own events).
      const t = e.target
      if (t instanceof Element && t.closest('[data-pin-toggle]')) return
      if (!pins?.enabled) return
      scheduleSettingsControlChanged({
        surface,
        role: telemetryRole,
        settingId,
        enabled: true,
      })
    },
    [pins?.enabled, settingId, surface, telemetryRole],
  )

  // When pin state flips, keep keyboard focus on this control's pin toggle.
  useLayoutEffect(() => {
    if (wasPinnedRef.current !== pinned) {
      wasPinnedRef.current = pinned
      if (pinned && pinHost) {
        requestAnimationFrame(() => {
          const btn = pinToggleRef.current?.querySelector('button')
          if (btn instanceof HTMLElement) btn.focus()
        })
      }
    }
  }, [pinned, pinHost])

  if (!matches(settingId)) return null

  // Flag off / not pinnable: return children without wrapper so parent
  // `divide-y` / `space-y` sibling selectors keep working (PS.1 §14).
  // Still track control changes when the feature flag is on but the control is not pinnable.
  if (!pinEnabled) {
    if (pins?.enabled) {
      return (
        <div data-setting-row={settingId} onChange={onControlInteraction} onClick={onControlInteraction}>
          {children}
        </div>
      )
    }
    return <>{children}</>
  }

  const body = (
    <div
      data-setting-row={settingId}
      className="group/setting-row relative"
      onChange={onControlInteraction}
      onClick={onControlInteraction}
    >
      <div className="absolute end-0 top-1 z-10" ref={pinToggleRef}>
        <PinToggle
          label={descriptor?.label ?? settingId}
          pinned={!!pinned}
          disabledAtCap={!pinned && pins.atCap}
          onToggle={() => pins.togglePin(settingId)}
          alwaysVisible={!!pinned}
        />
      </div>
      <div className={pinned ? 'pe-0' : 'pe-7'}>{children}</div>
    </div>
  )

  // Move-not-duplicate: portal into the Pinned group host when ready.
  if (pinned) {
    if (pinHost) {
      return createPortal(body, pinHost)
    }
    // Host not mounted yet — hold off so we never double-render ids.
    return null
  }

  return body
}

/** Test-only: reset the dev-mode unknown-id warn set. */
export function __resetSettingRowWarnCacheForTests(): void {
  warnedUnknownIds.clear()
}
