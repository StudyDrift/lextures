import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { CourseCoupon } from '../../../lib/course-coupons-api'

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string, opts?: Record<string, unknown>) => {
      if (!opts) return key
      let s = key
      for (const [k, v] of Object.entries(opts)) {
        s = s.replace(`{{${k}}}`, String(v))
      }
      return s
    },
    i18n: { language: 'en-US' },
  }),
}))

vi.mock('../../../components/use-confirm', () => ({
  useConfirm: () => ({
    confirm: vi.fn(async () => true),
    ConfirmDialogHost: null,
  }),
}))

vi.mock('../../../hooks/use-online-status', () => ({
  useOnlineStatus: () => true,
}))

vi.mock('../../../lib/lms-toast', () => ({
  toastMutationError: vi.fn(),
  toastSaveOk: vi.fn(),
}))

vi.mock('../../../lib/coupon-manager-telemetry', () => ({
  emitCouponManagerTelemetry: vi.fn(),
}))

vi.mock('../course-coupon-create-dialog', () => ({
  CourseCouponCreateDialog: () => null,
}))

vi.mock('../course-coupon-redemptions-drawer', () => ({
  CourseCouponRedemptionsDrawer: () => null,
}))

vi.mock('../course-coupon-row-actions', () => ({
  CourseCouponRowActions: ({
    coupon,
    onPause,
  }: {
    coupon: CourseCoupon
    onPause: (c: CourseCoupon) => void
  }) => (
    <button type="button" onClick={() => onPause(coupon)}>
      {`pause-${coupon.code}`}
    </button>
  ),
}))

vi.mock('../../../lib/course-coupons-api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../../lib/course-coupons-api')>()
  return {
    ...actual,
    fetchCourseCoupons: vi.fn(),
    fetchCouponSummary: vi.fn(),
    updateCourseCoupon: vi.fn(),
    archiveCourseCoupon: vi.fn(),
  }
})

import { CourseCouponsPanel } from '../course-coupons-panel'
import * as couponsApi from '../../../lib/course-coupons-api'

const fetchCourseCoupons = vi.mocked(couponsApi.fetchCourseCoupons)
const fetchCouponSummary = vi.mocked(couponsApi.fetchCouponSummary)
const updateCourseCoupon = vi.mocked(couponsApi.updateCourseCoupon)

function sampleCoupon(overrides: Partial<CourseCoupon> = {}): CourseCoupon {
  return {
    id: 'c1',
    courseId: 'course-1',
    code: 'LAUNCH25',
    discountType: 'percent',
    percentOff: 25,
    amountOffCents: null,
    currency: null,
    startsAt: null,
    endsAt: null,
    maxRedemptions: 50,
    maxRedemptionsPerUser: 1,
    seats: { consumed: 12, reserved: 0, redeemed: 12, remaining: 38 },
    status: 'active',
    note: null,
    shareUrl: 'http://localhost:5173/marketplace/demo?coupon=LAUNCH25',
    publicShareUrl: null,
    createdBy: null,
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

describe('CourseCouponsPanel', () => {
  beforeEach(() => {
    fetchCourseCoupons.mockReset()
    fetchCouponSummary.mockReset()
    updateCourseCoupon.mockReset()
    fetchCouponSummary.mockResolvedValue({ rows: [], currency: 'usd' })
  })

  it('shows free-course empty state without create when price is 0', async () => {
    fetchCourseCoupons.mockResolvedValue([])
    render(<CourseCouponsPanel courseCode="DEMO" priceCents={0} priceCurrency="usd" />)
    expect(screen.getByText('course.settings.coupons.freeCourse')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'course.settings.coupons.new' })).toBeNull()
  })

  it('renders empty state with create for paid courses', async () => {
    fetchCourseCoupons.mockResolvedValue([])
    render(<CourseCouponsPanel courseCode="DEMO" priceCents={4000} priceCurrency="usd" />)
    await waitFor(() => {
      expect(screen.getByText('course.settings.coupons.emptyTitle')).toBeInTheDocument()
    })
    expect(
      screen.getAllByRole('button', { name: 'course.settings.coupons.new' }).length,
    ).toBeGreaterThan(0)
  })

  it('renders usage with accessible text', async () => {
    fetchCourseCoupons.mockResolvedValue([sampleCoupon()])
    render(<CourseCouponsPanel courseCode="DEMO" priceCents={4000} priceCurrency="usd" />)
    await waitFor(() => {
      expect(screen.getAllByText('LAUNCH25').length).toBeGreaterThan(0)
    })
    expect(screen.getAllByText('12 / 50').length).toBeGreaterThan(0)
    expect(screen.getAllByText('course.settings.coupons.usageOf').length).toBeGreaterThan(0)
  })

  it('shows unlimited usage accessible text', async () => {
    fetchCourseCoupons.mockResolvedValue([
      sampleCoupon({
        maxRedemptions: null,
        seats: { consumed: 3, reserved: 0, redeemed: 3, remaining: null },
      }),
    ])
    render(<CourseCouponsPanel courseCode="DEMO" priceCents={4000} priceCurrency="usd" />)
    await waitFor(() => {
      expect(screen.getAllByText('course.settings.coupons.usageUnlimited').length).toBeGreaterThan(0)
    })
  })

  it('pauses a coupon optimistically', async () => {
    const coupon = sampleCoupon()
    fetchCourseCoupons.mockResolvedValue([coupon])
    updateCourseCoupon.mockResolvedValue({ ...coupon, status: 'disabled' })
    const user = userEvent.setup()
    render(<CourseCouponsPanel courseCode="DEMO" priceCents={4000} priceCurrency="usd" />)
    await waitFor(() => expect(screen.getAllByText('LAUNCH25').length).toBeGreaterThan(0))
    await user.click(screen.getAllByRole('button', { name: 'pause-LAUNCH25' })[0]!)
    await waitFor(() => {
      expect(updateCourseCoupon).toHaveBeenCalledWith('DEMO', 'c1', { status: 'disabled' })
    })
  })
})
