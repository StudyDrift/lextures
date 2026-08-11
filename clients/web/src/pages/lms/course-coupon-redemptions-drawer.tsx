import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Button,
  EmptyState,
  ErrorState,
  Sheet,
  Skeleton,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  Badge,
} from '../../components/ui'
import { Download, Ticket } from 'lucide-react'
import {
  CourseCouponApiError,
  exportCouponRedemptionsCsv,
  fetchCouponRedemptions,
  formatCouponPerformanceSummary,
  type CouponRedemptionRow,
  type CouponSummaryRow,
  type CourseCoupon,
} from '../../lib/course-coupons-api'
import { formatMarketplacePrice } from '../../lib/marketplace-price'
import { formatDateTime } from '../../lib/format'
import { toastMutationError, toastSaveOk } from '../../lib/lms-toast'

export type CourseCouponRedemptionsDrawerProps = {
  open: boolean
  onClose: () => void
  courseCode: string
  coupon: CourseCoupon | null
  /** Optional performance figures (MKTC.7). */
  summary?: CouponSummaryRow
}

const PAGE_LIMIT = 25

export function CourseCouponRedemptionsDrawer({
  open,
  onClose,
  courseCode,
  coupon,
  summary,
}: CourseCouponRedemptionsDrawerProps) {
  const { t, i18n } = useTranslation('common')
  const [rows, setRows] = useState<CouponRedemptionRow[]>([])
  const [nextCursor, setNextCursor] = useState('')
  const [loading, setLoading] = useState(false)
  const [loadingMore, setLoadingMore] = useState(false)
  const [exporting, setExporting] = useState(false)
  const [error, setError] = useState(false)
  const [exportLive, setExportLive] = useState('')

  const load = useCallback(
    async (cursor?: string, append = false) => {
      if (!coupon) return
      if (append) setLoadingMore(true)
      else {
        setLoading(true)
        setError(false)
      }
      try {
        const result = await fetchCouponRedemptions(courseCode, coupon.id, {
          cursor,
          limit: PAGE_LIMIT,
        })
        setRows((prev) => (append ? [...prev, ...result.rows] : result.rows))
        setNextCursor(result.nextCursor)
      } catch (e) {
        setError(true)
        toastMutationError(
          e instanceof Error ? e.message : t('course.settings.coupons.error.redemptions'),
        )
      } finally {
        setLoading(false)
        setLoadingMore(false)
      }
    },
    [coupon, courseCode, t],
  )

  useEffect(() => {
    if (!open || !coupon) {
      setRows([])
      setNextCursor('')
      setError(false)
      return
    }
    void load()
  }, [open, coupon, load])

  const statusTone = (status: CouponRedemptionRow['status']) => {
    if (status === 'redeemed') return 'success' as const
    if (status === 'reserved') return 'info' as const
    return 'neutral' as const
  }

  const statusLabel = (status: CouponRedemptionRow['status']) => {
    if (status === 'redeemed') return t('course.settings.coupons.redemptionStatus.redeemed')
    if (status === 'reserved') return t('course.settings.coupons.redemptionStatus.reserved')
    return t('course.settings.coupons.redemptionStatus.released')
  }

  async function handleExport() {
    if (!coupon || exporting) return
    setExporting(true)
    setExportLive('')
    try {
      await exportCouponRedemptionsCsv(courseCode, coupon.id, coupon.code)
      const msg = t('course.settings.coupons.exportDone', { code: coupon.code })
      toastSaveOk(msg)
      setExportLive(msg)
    } catch (e) {
      if (e instanceof CourseCouponApiError && e.status === 429) {
        toastMutationError(t('course.settings.coupons.error.exportRateLimited'))
      } else {
        toastMutationError(
          e instanceof Error ? e.message : t('course.settings.coupons.error.export'),
        )
      }
    } finally {
      setExporting(false)
    }
  }

  const perf =
    coupon && summary
      ? formatCouponPerformanceSummary(summary, i18n.language, {
          empty: t('course.settings.coupons.performanceEmpty'),
          claimed: (n) => t('course.settings.coupons.performanceClaimed', { count: n }),
          off: (amount) => t('course.settings.coupons.performanceOff', { amount }),
          net: (amount) => t('course.settings.coupons.performanceNet', { amount }),
        })
      : null

  return (
    <Sheet
      open={open}
      onClose={onClose}
      title={
        coupon
          ? t('course.settings.coupons.redemptionsTitle', { code: coupon.code })
          : t('course.settings.coupons.redemptions')
      }
      closeLabel={t('dialogs.close')}
      panelClassName="max-w-xl w-full"
    >
      <div className="flex h-full flex-col gap-4 p-4">
        {coupon ? (
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div className="min-w-0 space-y-1">
              {perf ? (
                <p className="text-sm text-fg-muted" aria-label={perf.sr}>
                  {perf.visible}
                </p>
              ) : null}
              {summary && summary.refundedCount > 0 ? (
                <p className="text-xs text-fg-muted">
                  {t('course.settings.coupons.performanceRefunded', {
                    count: summary.refundedCount,
                  })}
                </p>
              ) : null}
            </div>
            <Button
              type="button"
              variant="secondary"
              size="sm"
              loading={exporting}
              disabled={!coupon}
              onClick={() => void handleExport()}
              aria-label={t('course.settings.coupons.exportCsvAria', { code: coupon.code })}
            >
              <Download className="h-4 w-4" aria-hidden />
              {t('course.settings.coupons.exportCsv')}
            </Button>
          </div>
        ) : null}
        <span className="sr-only" aria-live="polite">
          {exportLive}
        </span>
        {loading ? (
          <div className="space-y-2" aria-busy="true">
            <Skeleton className="h-10 w-full" label={t('course.settings.coupons.loading')} />
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
          </div>
        ) : error ? (
          <ErrorState
            title={t('course.settings.coupons.error.redemptions')}
            primaryAction={{
              label: t('marketplace.retry'),
              onClick: () => void load(),
            }}
          />
        ) : rows.length === 0 ? (
          <EmptyState
            icon={Ticket}
            title={t('course.settings.coupons.redemptionsEmptyTitle')}
            body={t('course.settings.coupons.redemptionsEmptyBody')}
          />
        ) : (
          <>
            <Table>
              <caption className="sr-only">
                {coupon
                  ? t('course.settings.coupons.redemptionsTitle', { code: coupon.code })
                  : t('course.settings.coupons.redemptions')}
              </caption>
              <TableHeader>
                <TableRow>
                  <TableHead scope="col">{t('course.settings.coupons.redemption.learner')}</TableHead>
                  <TableHead scope="col">{t('course.settings.coupons.redemption.status')}</TableHead>
                  <TableHead scope="col">{t('course.settings.coupons.redemption.charged')}</TableHead>
                  <TableHead scope="col">{t('course.settings.coupons.redemption.discount')}</TableHead>
                  <TableHead scope="col">{t('course.settings.coupons.redemption.date')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {rows.map((row) => {
                  const name = row.userName?.trim() || row.userEmail?.trim() || row.userId
                  const dateIso = row.redeemedAt || row.reservedAt
                  return (
                    <TableRow key={row.id}>
                      <TableCell>
                        <div className="min-w-0">
                          <p className="truncate font-medium text-fg-default">{name}</p>
                          {row.userEmail && row.userName ? (
                            <p className="truncate text-xs text-fg-muted">{row.userEmail}</p>
                          ) : null}
                        </div>
                      </TableCell>
                      <TableCell>
                        <Badge tone={statusTone(row.status)}>{statusLabel(row.status)}</Badge>
                      </TableCell>
                      <TableCell>
                        {formatMarketplacePrice(
                          row.chargedCents,
                          row.currency,
                          i18n.language,
                          t('course.settings.marketplace.free'),
                        )}
                      </TableCell>
                      <TableCell>
                        {formatMarketplacePrice(row.discountCents, row.currency, i18n.language, '—')}
                      </TableCell>
                      <TableCell className="whitespace-nowrap text-fg-muted">
                        {formatDateTime(dateIso)}
                      </TableCell>
                    </TableRow>
                  )
                })}
              </TableBody>
            </Table>
            {nextCursor ? (
              <div className="flex justify-center">
                <Button
                  type="button"
                  variant="secondary"
                  loading={loadingMore}
                  onClick={() => void load(nextCursor, true)}
                >
                  {t('course.settings.coupons.loadMore')}
                </Button>
              </div>
            ) : null}
          </>
        )}
      </div>
    </Sheet>
  )
}
