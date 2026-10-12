import { useCallback, useEffect, useState } from 'react'
import { MonitorSmartphone } from 'lucide-react'
import { Button, InlineAlert, LinkButton } from '../ui'
import { FAMILY_PICKER_PATH, getStoredFamilyDevice } from '../../lib/family-device'
import {
  listFamilyDevices,
  registerThisFamilyDevice,
  revokeFamilyDevice,
  type FamilyDevice,
} from '../../lib/family-sign-in-api'

/** Parent turns this browser into a shared family device (#771). */
export function FamilyDevicePanel() {
  const [devices, setDevices] = useState<FamilyDevice[]>([])
  const [thisDeviceId, setThisDeviceId] = useState<string | null>(() => getStoredFamilyDevice()?.id ?? null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const reload = useCallback(async () => {
    try {
      setDevices(await listFamilyDevices())
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load family devices')
    }
  }, [])

  useEffect(() => {
    void reload()
  }, [reload])

  const thisDeviceActive = thisDeviceId !== null && devices.some((d) => d.id === thisDeviceId)

  async function markThisDevice() {
    setBusy(true)
    setError(null)
    try {
      const dev = await registerThisFamilyDevice()
      setThisDeviceId(dev.id)
      await reload()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to set up family device')
    } finally {
      setBusy(false)
    }
  }

  async function revoke(id: string) {
    setBusy(true)
    setError(null)
    try {
      await revokeFamilyDevice(id)
      if (id === thisDeviceId) setThisDeviceId(null)
      await reload()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to turn off family device')
    } finally {
      setBusy(false)
    }
  }

  return (
    <section
      aria-labelledby="family-device-heading"
      className="space-y-3 rounded-2xl border border-border-default bg-surface-raised p-4 shadow-sm"
    >
      <div className="flex items-start gap-3">
        <MonitorSmartphone className="mt-0.5 h-5 w-5 shrink-0 text-fg-muted" aria-hidden />
        <div className="min-w-0 space-y-1">
          <h2 id="family-device-heading" className="text-body-sm font-semibold text-fg-default">
            Family device sign-in
          </h2>
          <p className="text-body-sm text-fg-muted">
            On a shared computer or tablet, learners can pick their name on &quot;Who&apos;s learning?&quot; and sign in
            with their own PIN or picture password. Your account stays signed in or out exactly as you leave it.
          </p>
        </div>
      </div>
      {error ? <InlineAlert tone="danger">{error}</InlineAlert> : null}
      <div className="flex flex-wrap items-center gap-2">
        {thisDeviceActive ? (
          <>
            <LinkButton to={FAMILY_PICKER_PATH} size="sm">
              Open &quot;Who&apos;s learning?&quot;
            </LinkButton>
            <span className="text-caption text-fg-muted">This browser is a family device.</span>
          </>
        ) : (
          <Button type="button" size="sm" loading={busy} onClick={() => void markThisDevice()}>
            Use this browser as a family device
          </Button>
        )}
      </div>
      {devices.length > 0 ? (
        <ul className="divide-y divide-border-default rounded-xl border border-border-default">
          {devices.map((d) => (
            <li key={d.id} className="flex flex-wrap items-center justify-between gap-2 px-3 py-2 text-body-sm">
              <span className="text-fg-default">
                {d.label}
                {d.id === thisDeviceId ? <span className="text-fg-muted"> · this browser</span> : null}
                <span className="block text-caption text-fg-muted">
                  {d.lastUsedAt
                    ? `Last learner sign-in ${new Date(d.lastUsedAt).toLocaleString()}`
                    : `Added ${new Date(d.createdAt).toLocaleDateString()}`}
                </span>
              </span>
              <Button type="button" variant="ghost" size="sm" disabled={busy} onClick={() => void revoke(d.id)}>
                Turn off
              </Button>
            </li>
          ))}
        </ul>
      ) : null}
    </section>
  )
}
