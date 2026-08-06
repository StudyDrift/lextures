import { useCallback, useEffect, useState } from 'react'
import {
  fetchAdaptiveContentOversight,
  postAdaptiveContentKillSwitch,
  type AdaptiveContentOversight,
} from '../../lib/courses-api'
import { toastMutationError, toastSaveOk } from '../../lib/lms-toast'
import { useConfirm } from '../use-confirm'

/**
 * AC.8 — admin oversight console summarizing ACE activity, disparity flags, and incident controls.
 */
export function AdaptiveContentOversightPanel() {
  const [data, setData] = useState<AdaptiveContentOversight | null>(null)
  const [loading, setLoading] = useState(true)
  const [busy, setBusy] = useState(false)
  const { confirm, ConfirmDialogHost } = useConfirm()

  const load = useCallback(async () => {
    try {
      const o = await fetchAdaptiveContentOversight()
      setData(o)
    } catch {
      setData(null)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  if (loading || !data) {
    return null
  }

  return (
    <section aria-labelledby="ace-oversight-heading" className="mt-8" data-testid="ace-oversight-panel">
      <h2
        id="ace-oversight-heading"
        className="text-base font-semibold text-fg-default"
      >
        Adaptive content oversight
      </h2>
      <p className="mt-1 text-sm text-fg-muted">
        Governance summary for the Adaptive Content Engine — disparity flags, incidents, and
        kill-switch.
      </p>

      <dl className="mt-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        {[
          ['Kill-switch', data.killSwitch ? 'Engaged' : 'Disengaged'],
          ['Generation paused', data.generationPaused ? 'Yes' : 'No'],
          [
            'Org enabled',
            data.orgEnabled === null || data.orgEnabled === undefined
              ? 'No opinion'
              : data.orgEnabled
                ? 'Allowed'
                : 'Disabled',
          ],
          ['Queue depth', String(data.queueDepth)],
          ['Open contests', String(data.openContests)],
          ['Disparity flags', String(data.disparityFlags)],
          ['Quarantined units', String(data.quarantinedUnits)],
          ['Regressing units', String(data.regressingUnits)],
          ['Gate blocks (7d)', String(data.gateBlocks7d)],
          ['AI cost (30d)', `$${data.costUsd30d.toFixed(2)}`],
        ].map(([label, value]) => (
          <div
            key={label}
            className="rounded-lg border border-border-default px-3 py-2 dark:border-border-default"
          >
            <dt className="text-xs font-medium uppercase tracking-wide text-fg-muted">{label}</dt>
            <dd className="mt-0.5 text-sm font-semibold text-fg-default">
              {value}
              {label === 'Disparity flags' && data.disparityFlags > 0 ? (
                <span className="ms-2 text-warning-fg" aria-label="Attention needed">
                  !
                </span>
              ) : null}
            </dd>
          </div>
        ))}
      </dl>

      <div className="mt-4 flex flex-wrap gap-3">
        <button
          type="button"
          disabled={busy}
          className="rounded-xl bg-rose-700 px-4 py-2 text-sm font-semibold text-white hover:bg-rose-600 disabled:opacity-60"
          onClick={() => {
            void (async () => {
              const ok = await confirm({
                title: data.killSwitch
                  ? 'Disengage the adaptive content kill-switch?'
                  : 'Engage the adaptive content kill-switch?',
                description: data.killSwitch
                  ? 'Generation and serving will resume subject to course flags and gates.'
                  : 'Generation and serving will stop immediately for all courses.',
                confirmLabel: data.killSwitch ? 'Disengage' : 'Engage kill-switch',
                variant: data.killSwitch ? 'default' : 'danger',
              })
              if (!ok) return
              setBusy(true)
              try {
                await postAdaptiveContentKillSwitch(!data.killSwitch)
                toastSaveOk(data.killSwitch ? 'Kill-switch disengaged.' : 'Kill-switch engaged.')
                await load()
              } catch (e) {
                toastMutationError(e instanceof Error ? e.message : 'Failed.')
              } finally {
                setBusy(false)
              }
            })()
          }}
        >
          {data.killSwitch ? 'Disengage kill-switch' : 'Engage kill-switch'}
        </button>
        {data.dpiaDocPath ? (
          <a
            href={data.dpiaDocPath}
            className="rounded-xl border border-border-default px-4 py-2 text-sm font-medium text-fg-muted dark:border-border-default dark:text-fg-default"
          >
            DPIA
          </a>
        ) : null}
        {data.aiActChecklistPath ? (
          <a
            href={data.aiActChecklistPath}
            className="rounded-xl border border-border-default px-4 py-2 text-sm font-medium text-fg-muted dark:border-border-default dark:text-fg-default"
          >
            EU AI Act checklist
          </a>
        ) : null}
      </div>
      {ConfirmDialogHost}
    </section>
  )
}
