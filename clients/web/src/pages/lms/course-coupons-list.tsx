import type { MutableRefObject, ReactNode } from 'react'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '../../components/ui'
import { formatDateTime } from '../../lib/format'
import type { CourseCoupon } from '../../lib/course-coupons-api'
import { CourseCouponRowActions } from './course-coupon-row-actions'

export type CourseCouponsListProps = {
  coupons: CourseCoupon[]
  mutationsDisabled: boolean
  statusBadge: (status: CourseCoupon['status']) => ReactNode
  usageLabel: (c: CourseCoupon) => { visible: string; sr: string }
  windowLabel: (c: CourseCoupon) => string
  discountLabel: (c: CourseCoupon) => string
  performanceLabel: (c: CourseCoupon) => { visible: string; sr: string }
  copyRefs: MutableRefObject<Map<string, HTMLButtonElement>>
  announce: (message: string) => void
  onEdit: (coupon: CourseCoupon) => void
  onPause: (coupon: CourseCoupon) => void
  onResume: (coupon: CourseCoupon) => void
  onArchive: (coupon: CourseCoupon) => void
  onViewRedemptions: (coupon: CourseCoupon) => void
  title: string
  labels: {
    code: string
    discount: string
    window: string
    usage: string
    performance: string
    status: string
    actions: string
  }
}

export function CourseCouponsList({
  coupons,
  mutationsDisabled,
  statusBadge,
  usageLabel,
  windowLabel,
  discountLabel,
  performanceLabel,
  copyRefs,
  announce,
  onEdit,
  onPause,
  onResume,
  onArchive,
  onViewRedemptions,
  title,
  labels,
}: CourseCouponsListProps) {
  return (
    <>
      <div className="hidden sm:block">
        <Table>
          <caption className="sr-only">{title}</caption>
          <TableHeader className="bg-transparent">
            <TableRow>
              <TableHead scope="col">{labels.code}</TableHead>
              <TableHead scope="col">{labels.discount}</TableHead>
              <TableHead scope="col">{labels.window}</TableHead>
              <TableHead scope="col">{labels.usage}</TableHead>
              <TableHead scope="col">{labels.performance}</TableHead>
              <TableHead scope="col">{labels.status}</TableHead>
              <TableHead scope="col">
                <span className="sr-only">{labels.actions}</span>
              </TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {coupons.map((c) => {
              const usage = usageLabel(c)
              const perf = performanceLabel(c)
              return (
                <TableRow key={c.id} data-coupon-id={c.id}>
                  <TableCell>
                    <code className="select-all font-mono text-sm font-semibold tracking-wide text-fg-default">
                      {c.code}
                    </code>
                    {c.note ? (
                      <p className="mt-0.5 max-w-xs truncate text-xs text-fg-muted" title={c.note}>
                        {c.note}
                      </p>
                    ) : null}
                  </TableCell>
                  <TableCell>{discountLabel(c)}</TableCell>
                  <TableCell>
                    <span title={c.startsAt || c.endsAt ? formatDateTime(c.startsAt || c.endsAt) : undefined}>
                      {windowLabel(c)}
                    </span>
                  </TableCell>
                  <TableCell>
                    <span aria-hidden>{usage.visible}</span>
                    <span className="sr-only">{usage.sr}</span>
                  </TableCell>
                  <TableCell>
                    <span className="text-sm text-fg-muted" aria-hidden>
                      {perf.visible}
                    </span>
                    <span className="sr-only">{perf.sr}</span>
                  </TableCell>
                  <TableCell>{statusBadge(c.status)}</TableCell>
                  <TableCell className="text-end">
                    <CourseCouponRowActions
                      coupon={c}
                      disabled={mutationsDisabled}
                      announce={announce}
                      copyButtonRef={(el) => {
                        if (el) copyRefs.current.set(c.id, el)
                        else copyRefs.current.delete(c.id)
                      }}
                      onEdit={onEdit}
                      onPause={onPause}
                      onResume={onResume}
                      onArchive={onArchive}
                      onViewRedemptions={onViewRedemptions}
                    />
                  </TableCell>
                </TableRow>
              )
            })}
          </TableBody>
        </Table>
      </div>

      <ul className="space-y-3 sm:hidden">
        {coupons.map((c) => {
          const usage = usageLabel(c)
          const perf = performanceLabel(c)
          return (
            <li
              key={c.id}
              className="rounded-xl border border-border-default bg-surface-base p-4"
              data-coupon-id={c.id}
            >
              <div className="flex items-start justify-between gap-2">
                <code className="select-all font-mono text-base font-semibold tracking-wide">
                  {c.code}
                </code>
                {statusBadge(c.status)}
              </div>
              <dl className="mt-3 space-y-1 text-sm">
                <div className="flex justify-between gap-2">
                  <dt className="text-fg-muted">{labels.discount}</dt>
                  <dd>{discountLabel(c)}</dd>
                </div>
                <div className="flex justify-between gap-2">
                  <dt className="text-fg-muted">{labels.window}</dt>
                  <dd>{windowLabel(c)}</dd>
                </div>
                <div className="flex justify-between gap-2">
                  <dt className="text-fg-muted">{labels.usage}</dt>
                  <dd>
                    <span aria-hidden>{usage.visible}</span>
                    <span className="sr-only">{usage.sr}</span>
                  </dd>
                </div>
                <div className="flex justify-between gap-2">
                  <dt className="text-fg-muted">{labels.performance}</dt>
                  <dd>
                    <span aria-hidden>{perf.visible}</span>
                    <span className="sr-only">{perf.sr}</span>
                  </dd>
                </div>
              </dl>
              <div className="mt-3">
                <CourseCouponRowActions
                  coupon={c}
                  disabled={mutationsDisabled}
                  announce={announce}
                  copyButtonRef={(el) => {
                    if (el) copyRefs.current.set(c.id, el)
                    else copyRefs.current.delete(c.id)
                  }}
                  onEdit={onEdit}
                  onPause={onPause}
                  onResume={onResume}
                  onArchive={onArchive}
                  onViewRedemptions={onViewRedemptions}
                />
              </div>
            </li>
          )
        })}
      </ul>
    </>
  )
}
