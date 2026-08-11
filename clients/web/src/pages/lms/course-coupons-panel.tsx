import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Plus, Ticket } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import {
  Badge,
  Button,
  EmptyState,
  ErrorState,
  Skeleton,
  Switch,
} from '../../components/ui'
import { useConfirm } from '../../components/use-confirm'
import { useOnlineStatus } from '../../hooks/use-online-status'
import {
  archiveCourseCoupon,
  describeCouponWindow,
  describeDiscount,
  fetchCourseCoupons,
  fetchCouponSummary,
  formatCouponPerformanceSummary,
  updateCourseCoupon,
  type CouponSummaryRow,
  type CourseCoupon,
} from '../../lib/course-coupons-api'
import { emitCouponManagerTelemetry } from '../../lib/coupon-manager-telemetry'
import { formatDate } from '../../lib/format'
import { toastMutationError, toastSaveOk } from '../../lib/lms-toast'
import { CourseCouponCreateDialog } from './course-coupon-create-dialog'
import { CourseCouponRedemptionsDrawer } from './course-coupon-redemptions-drawer'
import { CourseCouponsList } from './course-coupons-list'

export type CourseCouponsPanelProps = {
  courseCode: string
  /** Saved catalog list price (not dirty fee form value). */
  priceCents: number
  priceCurrency: string
  /** True when fee form above has unsaved price changes. */
  priceFormDirty?: boolean
}

export function CourseCouponsPanel({
  courseCode,
  priceCents,
  priceCurrency,
  priceFormDirty = false,
}: CourseCouponsPanelProps) {
  const { t, i18n } = useTranslation('common')
  const { confirm, ConfirmDialogHost } = useConfirm()
  const isOnline = useOnlineStatus()
  const [coupons, setCoupons] = useState<CourseCoupon[]>([])
  const [summaryById, setSummaryById] = useState<Record<string, CouponSummaryRow>>({})
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(false)
  const [includeArchived, setIncludeArchived] = useState(false)
  const [createOpen, setCreateOpen] = useState(false)
  const [editing, setEditing] = useState<CourseCoupon | null>(null)
  const [redemptionsCoupon, setRedemptionsCoupon] = useState<CourseCoupon | null>(null)
  const [liveMessage, setLiveMessage] = useState('')
  const [focusCopyId, setFocusCopyId] = useState<string | null>(null)
  const copyRefs = useRef(new Map<string, HTMLButtonElement>())
  const openedRef = useRef(false)

  const announce = useCallback((message: string) => {
    setLiveMessage('')
    // Force re-announce even when the message text is identical.
    queueMicrotask(() => setLiveMessage(message))
  }, [])

  const reload = useCallback(async () => {
    setLoading(true)
    setError(false)
    try {
      const [rows, summary] = await Promise.all([
        fetchCourseCoupons(courseCode, { includeArchived }),
        fetchCouponSummary(courseCode).catch(() => ({ rows: [] as CouponSummaryRow[], currency: priceCurrency })),
      ])
      setCoupons(rows)
      const map: Record<string, CouponSummaryRow> = {}
      for (const r of summary.rows) {
        map[r.couponId] = r
      }
      setSummaryById(map)
    } catch (e) {
      setError(true)
      toastMutationError(
        e instanceof Error ? e.message : t('course.settings.coupons.error.load'),
      )
    } finally {
      setLoading(false)
    }
    // t is stable under real i18next; omit from deps to avoid reload loops with test mocks.
    // eslint-disable-next-line react-hooks/exhaustive-deps -- t intentionally excluded
  }, [courseCode, includeArchived, priceCurrency])

  useEffect(() => {
    void reload()
  }, [reload])

  useEffect(() => {
    if (openedRef.current) return
    openedRef.current = true
    emitCouponManagerTelemetry('coupon_manager_opened')
  }, [])

  useEffect(() => {
    if (!focusCopyId) return
    const el = copyRefs.current.get(focusCopyId)
    el?.focus()
    setFocusCopyId(null)
  }, [focusCopyId, coupons])

  const isFreeCourse = priceCents <= 0

  const statusBadge = (status: CourseCoupon['status']) => {
    if (status === 'active') {
      return <Badge tone="success">{t('course.settings.coupons.statusActive')}</Badge>
    }
    if (status === 'disabled') {
      return <Badge tone="warning">{t('course.settings.coupons.statusPaused')}</Badge>
    }
    return <Badge tone="neutral">{t('course.settings.coupons.statusArchived')}</Badge>
  }

  const usageLabel = (c: CourseCoupon) => {
    const used = c.seats.consumed
    if (c.maxRedemptions == null) {
      return {
        visible: `${used} / ∞`,
        sr: t('course.settings.coupons.usageUnlimited', { used }),
      }
    }
    return {
      visible: `${used} / ${c.maxRedemptions}`,
      sr: t('course.settings.coupons.usageOf', { used, limit: c.maxRedemptions }),
    }
  }

  const windowLabel = (c: CourseCoupon) =>
    describeCouponWindow(c.startsAt, c.endsAt, (iso) => formatDate(iso, { month: 'short', day: 'numeric' }), {
      always: t('course.settings.coupons.windowAlways'),
      until: (date) => t('course.settings.coupons.windowUntil', { date }),
      from: (date) => t('course.settings.coupons.windowFrom', { date }),
      range: (start, end) => t('course.settings.coupons.windowRange', { start, end }),
    }).label

  const discountLabel = (c: CourseCoupon) =>
    describeDiscount(c, i18n.language, {
      percentOff: (p) => t('course.settings.coupons.percentOff', { percent: p }),
      amountOff: (amount) => t('course.settings.coupons.amountOff', { amount }),
    })

  const performanceLabel = (c: CourseCoupon) =>
    formatCouponPerformanceSummary(summaryById[c.id], i18n.language, {
      empty: t('course.settings.coupons.performanceEmpty'),
      claimed: (n) => t('course.settings.coupons.performanceClaimed', { count: n }),
      off: (amount) => t('course.settings.coupons.performanceOff', { amount }),
      net: (amount) => t('course.settings.coupons.performanceNet', { amount }),
    })

  async function handlePause(c: CourseCoupon) {
    if (!isOnline) return
    const prev = coupons
    setCoupons((rows) => rows.map((r) => (r.id === c.id ? { ...r, status: 'disabled' } : r)))
    try {
      const updated = await updateCourseCoupon(courseCode, c.id, { status: 'disabled' })
      setCoupons((rows) => rows.map((r) => (r.id === updated.id ? updated : r)))
      emitCouponManagerTelemetry('coupon_paused', { status: 'disabled' })
      toastSaveOk(t('course.settings.coupons.pausedToast', { code: c.code }))
    } catch (e) {
      setCoupons(prev)
      toastMutationError(e instanceof Error ? e.message : t('course.settings.coupons.error.save'))
    }
  }

  async function handleResume(c: CourseCoupon) {
    if (!isOnline) return
    const prev = coupons
    setCoupons((rows) => rows.map((r) => (r.id === c.id ? { ...r, status: 'active' } : r)))
    try {
      const updated = await updateCourseCoupon(courseCode, c.id, { status: 'active' })
      setCoupons((rows) => rows.map((r) => (r.id === updated.id ? updated : r)))
      toastSaveOk(t('course.settings.coupons.resumedToast', { code: c.code }))
    } catch (e) {
      setCoupons(prev)
      toastMutationError(e instanceof Error ? e.message : t('course.settings.coupons.error.save'))
    }
  }

  async function handleArchive(c: CourseCoupon) {
    if (!isOnline) return
    const ok = await confirm({
      title: t('course.settings.coupons.archive'),
      description: t('course.settings.coupons.archiveConfirm', { code: c.code }),
      confirmLabel: t('course.settings.coupons.archive'),
      variant: 'danger',
    })
    if (!ok) return
    const prev = coupons
    setCoupons((rows) =>
      includeArchived
        ? rows.map((r) => (r.id === c.id ? { ...r, status: 'archived' as const } : r))
        : rows.filter((r) => r.id !== c.id),
    )
    try {
      const updated = await archiveCourseCoupon(courseCode, c.id)
      if (includeArchived) {
        setCoupons((rows) => rows.map((r) => (r.id === updated.id ? updated : r)))
      }
      emitCouponManagerTelemetry('coupon_archived')
      toastSaveOk(t('course.settings.coupons.archivedToast', { code: c.code }))
    } catch (e) {
      setCoupons(prev)
      toastMutationError(e instanceof Error ? e.message : t('course.settings.coupons.error.save'))
    }
  }

  function handleCreated(coupon: CourseCoupon) {
    setCoupons((rows) => [coupon, ...rows.filter((r) => r.id !== coupon.id)])
    setFocusCopyId(coupon.id)
    announce(t('course.settings.coupons.created'))
  }

  function handleUpdated(coupon: CourseCoupon) {
    setCoupons((rows) => rows.map((r) => (r.id === coupon.id ? coupon : r)))
  }

  const visibleCoupons = useMemo(() => coupons, [coupons])
  const mutationsDisabled = !isOnline || isFreeCourse

  return (
    <section
      className="mt-6 space-y-4 rounded-2xl border border-border-default bg-surface-raised p-5 shadow-sm"
      aria-labelledby="course-coupons-heading"
    >
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <h2 id="course-coupons-heading" className="text-sm font-semibold text-fg-default">
            {t('course.settings.coupons.title')}
          </h2>
          <p className="mt-1 text-sm text-fg-muted">{t('course.settings.coupons.description')}</p>
          {priceFormDirty ? (
            <p className="mt-1 text-xs text-warning-fg" role="status">
              {t('course.settings.coupons.priceDirtyHint')}
            </p>
          ) : null}
        </div>
        {!isFreeCourse ? (
          <div className="flex flex-wrap items-center gap-3">
            <Switch
              checked={includeArchived}
              onCheckedChange={setIncludeArchived}
              label={t('course.settings.coupons.showArchived')}
              disabled={loading}
            />
            <Button
              type="button"
              variant="primary"
              size="sm"
              disabled={mutationsDisabled}
              onClick={() => {
                setEditing(null)
                setCreateOpen(true)
              }}
            >
              <Plus className="h-4 w-4" aria-hidden />
              {t('course.settings.coupons.new')}
            </Button>
          </div>
        ) : null}
      </div>

      {!isOnline ? (
        <p
          className="rounded-lg border border-warning-fg/30 bg-warning-surface px-3 py-2 text-sm text-warning-fg"
          role="status"
        >
          {t('course.settings.coupons.offline')}
        </p>
      ) : null}

      <span className="sr-only" aria-live="polite">
        {liveMessage}
      </span>

      {isFreeCourse ? (
        <EmptyState
          icon={Ticket}
          title={t('course.settings.coupons.freeCourseTitle')}
          body={t('course.settings.coupons.freeCourse')}
        />
      ) : loading ? (
        <div className="space-y-2" aria-busy="true">
          <Skeleton className="h-12 w-full" label={t('course.settings.coupons.loading')} />
          <Skeleton className="h-12 w-full" />
          <Skeleton className="h-12 w-full" />
        </div>
      ) : error ? (
        <ErrorState
          title={t('course.settings.coupons.error.load')}
          primaryAction={{
            label: t('marketplace.retry'),
            onClick: () => void reload(),
          }}
        />
      ) : visibleCoupons.length === 0 ? (
        <EmptyState
          icon={Ticket}
          title={t('course.settings.coupons.emptyTitle')}
          body={t('course.settings.coupons.emptyBody')}
          primaryAction={
            mutationsDisabled
              ? undefined
              : {
                  label: t('course.settings.coupons.new'),
                  onClick: () => {
                    setEditing(null)
                    setCreateOpen(true)
                  },
                }
          }
        />
      ) : (
        <CourseCouponsList
          coupons={visibleCoupons}
          mutationsDisabled={mutationsDisabled}
          statusBadge={statusBadge}
          usageLabel={usageLabel}
          windowLabel={windowLabel}
          discountLabel={discountLabel}
          performanceLabel={performanceLabel}
          copyRefs={copyRefs}
          announce={announce}
          onEdit={(coupon) => {
            setEditing(coupon)
            setCreateOpen(true)
          }}
          onPause={(coupon) => void handlePause(coupon)}
          onResume={(coupon) => void handleResume(coupon)}
          onArchive={(coupon) => void handleArchive(coupon)}
          onViewRedemptions={(coupon) => {
            setRedemptionsCoupon(coupon)
            emitCouponManagerTelemetry('coupon_redemptions_viewed')
          }}
          title={t('course.settings.coupons.title')}
          labels={{
            code: t('course.settings.coupons.code'),
            discount: t('course.settings.coupons.discount'),
            window: t('course.settings.coupons.window'),
            usage: t('course.settings.coupons.usage'),
            performance: t('course.settings.coupons.performance'),
            status: t('course.settings.coupons.status'),
            actions: t('course.settings.coupons.actions'),
          }}
        />
      )}

      <CourseCouponCreateDialog
        open={createOpen}
        onClose={() => {
          setCreateOpen(false)
          setEditing(null)
        }}
        courseCode={courseCode}
        coursePriceCents={priceCents}
        courseCurrency={priceCurrency}
        editing={editing}
        onCreated={handleCreated}
        onUpdated={handleUpdated}
      />

      <CourseCouponRedemptionsDrawer
        open={Boolean(redemptionsCoupon)}
        onClose={() => setRedemptionsCoupon(null)}
        courseCode={courseCode}
        coupon={redemptionsCoupon}
        summary={redemptionsCoupon ? summaryById[redemptionsCoupon.id] : undefined}
      />

      {ConfirmDialogHost}
    </section>
  )
}
