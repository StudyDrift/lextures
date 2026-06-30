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
      <p className="text-sm text-slate-600 dark:text-slate-400">
        <span className="font-medium text-slate-900 dark:text-slate-100">Seats used:</span>{' '}
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
        <p id={labelId} className="text-sm font-medium text-slate-900 dark:text-slate-100">
          Seats used: {license.usedSeats} / {license.maxSeats}
        </p>
        <span className="text-sm tabular-nums text-slate-500 dark:text-slate-400" aria-hidden>
          {clamped.toFixed(0)}%
        </span>
      </div>
      <div
        className="mt-2 h-2.5 w-full overflow-hidden rounded-full bg-slate-200 dark:bg-neutral-800"
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
    <section className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm dark:border-neutral-800 dark:bg-neutral-900">
      <h2 className="text-base font-semibold text-slate-900 dark:text-slate-100">License</h2>
      <dl className="mt-3 grid gap-2 text-sm sm:grid-cols-2">
        <div>
          <dt className="text-slate-500 dark:text-slate-400">Tier</dt>
          <dd className="font-medium capitalize text-slate-900 dark:text-slate-100">{license.tier}</dd>
        </div>
        {license.contractStart ? (
          <div>
            <dt className="text-slate-500 dark:text-slate-400">Contract start</dt>
            <dd className="font-medium text-slate-900 dark:text-slate-100">{license.contractStart}</dd>
          </div>
        ) : null}
        {license.contractEnd ? (
          <div>
            <dt className="text-slate-500 dark:text-slate-400">Contract end</dt>
            <dd className="font-medium text-slate-900 dark:text-slate-100">{license.contractEnd}</dd>
          </div>
        ) : null}
      </dl>
      <div className="mt-4">
        <SeatUtilizationBar license={license} labelId={labelId} />
      </div>
      {license.notes ? (
        <p className="mt-3 text-sm text-slate-600 dark:text-slate-400">{license.notes}</p>
      ) : null}
    </section>
  )
}
