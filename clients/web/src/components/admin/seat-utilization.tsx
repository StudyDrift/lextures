import type { OrgLicense } from '../../lib/admin-console-api'
import { seatUtilizationColor } from '../../lib/admin-console-api'

export function SeatUtilizationBar({
  license,
  labelId,
}: {
  license: OrgLicense
  labelId?: string
}) {
  if (license.unlimited) {
    return (
      <p className="text-sm text-fg-muted dark:text-fg-subtle">
        <span className="font-medium text-fg-default dark:text-slate-100">Seats used:</span>{' '}
        {license.usedSeats} / Unlimited
      </p>
    )
  }

  const percent = license.percentUsed ?? (license.maxSeats > 0 ? (license.usedSeats / license.maxSeats) * 100 : 0)
  const clamped = Math.min(100, Math.max(0, percent))
  const color = seatUtilizationColor(clamped)

  return (
    <div>
      <div className="flex items-baseline justify-between gap-2">
        <p id={labelId} className="text-sm font-medium text-fg-default dark:text-slate-100">
          Seats used: {license.usedSeats} / {license.maxSeats}
        </p>
        <span className="text-sm tabular-nums text-fg-muted dark:text-fg-subtle" aria-hidden>
          {clamped.toFixed(0)}%
        </span>
      </div>
      <div
        className="mt-2 h-2.5 w-full overflow-hidden rounded-full bg-slate-200 dark:bg-surface-overlay"
        role="progressbar"
        aria-labelledby={labelId}
        aria-valuemin={0}
        aria-valuemax={license.maxSeats}
        aria-valuenow={license.usedSeats}
        aria-valuetext={`${license.usedSeats} of ${license.maxSeats} seats used, ${clamped.toFixed(0)} percent`}
      >
        <div className={`h-full rounded-full transition-all ${color}`} style={{ width: `${clamped}%` }} />
      </div>
    </div>
  )
}

export function LicenseDetailsCard({ license }: { license: OrgLicense }) {
  const labelId = 'seat-utilization-label'
  return (
    <section className="rounded-xl border border-border-default bg-surface-raised p-4 shadow-sm dark:border-border-subtle dark:bg-surface-raised">
      <h2 className="text-base font-semibold text-fg-default dark:text-slate-100">License</h2>
      <dl className="mt-3 grid gap-2 text-sm sm:grid-cols-2">
        <div>
          <dt className="text-fg-muted dark:text-fg-subtle">Tier</dt>
          <dd className="font-medium capitalize text-fg-default dark:text-slate-100">{license.tier}</dd>
        </div>
        {license.contractStart ? (
          <div>
            <dt className="text-fg-muted dark:text-fg-subtle">Contract start</dt>
            <dd className="font-medium text-fg-default dark:text-slate-100">{license.contractStart}</dd>
          </div>
        ) : null}
        {license.contractEnd ? (
          <div>
            <dt className="text-fg-muted dark:text-fg-subtle">Contract end</dt>
            <dd className="font-medium text-fg-default dark:text-slate-100">{license.contractEnd}</dd>
          </div>
        ) : null}
      </dl>
      <div className="mt-4">
        <SeatUtilizationBar license={license} labelId={labelId} />
      </div>
      {license.notes ? (
        <p className="mt-3 text-sm text-fg-muted dark:text-fg-subtle">{license.notes}</p>
      ) : null}
    </section>
  )
}
