import { useEffect, useId, useMemo, useState } from 'react'
import { ArrowLeft, ArrowRight } from 'lucide-react'
import { MarketingPageShell } from '../components/marketing-page-shell'
import { WindLines } from '../components/home/wind-lines'
import {
  CALCULATOR_DEFAULT_USERS,
  CALCULATOR_MAX_USERS,
  CALCULATOR_MIN_USERS,
  CALCULATOR_STEP,
  PRICING_TIERS,
  estimatedTotal,
  formatUsd,
  formatUsers,
  pricePerStudent,
  tierLabelForUsers,
} from '../lib/institution-pricing'

/** Slider fill percentage for a given user count (linear across min–max). */
function sliderPercent(users: number): number {
  const span = CALCULATOR_MAX_USERS - CALCULATOR_MIN_USERS
  if (span <= 0) return 0
  return Math.min(100, Math.max(0, ((users - CALCULATOR_MIN_USERS) / span) * 100))
}

export function PricingCalculatorPage() {
  const [users, setUsers] = useState(CALCULATOR_DEFAULT_USERS)
  const sliderId = useId()
  const rateId = useId()

  useEffect(() => {
    document.title = 'Pricing calculator — Lextures'
  }, [])

  const rate = useMemo(() => pricePerStudent(users), [users])
  const total = useMemo(() => estimatedTotal(users), [users])
  const tierLabel = useMemo(() => tierLabelForUsers(users), [users])
  const fill = sliderPercent(users)

  return (
    <MarketingPageShell>
      <section className="relative overflow-hidden">
        <WindLines variant="hero" />
        <div className="relative z-[2] mx-auto max-w-[880px] px-5 py-14 md:px-10 md:py-16 xl:px-14">
          <a
            href="/pricing"
            className="inline-flex items-center gap-1.5 text-[14px] font-medium no-underline"
            style={{ color: 'var(--text-soft)' }}
          >
            <ArrowLeft className="h-4 w-4" aria-hidden />
            Back to pricing
          </a>

          <span
            className="mt-8 inline-flex items-center rounded-full px-3.5 py-[7px] text-[13px] font-semibold uppercase tracking-[0.04em]"
            style={{ color: '#4fa894', backgroundColor: 'rgba(106,197,176,0.14)' }}
          >
            University or district
          </span>
          <h1
            className="font-display mt-5 max-w-[720px] font-semibold leading-[1.05] tracking-[-0.02em] text-balance"
            style={{ color: '#22333b', fontSize: 'clamp(32px,4.4vw,48px)' }}
          >
            Pricing calculator
          </h1>
          <p className="mt-4 max-w-[640px] text-[17px] leading-relaxed" style={{ color: '#4a5b5d' }}>
            Estimate hosted university or district pricing by enrollment size. Bulk discounts apply
            automatically as your student count grows.
          </p>
        </div>
        <div
          className="h-px"
          style={{ background: 'linear-gradient(90deg, transparent, rgba(38,58,60,0.1), transparent)' }}
        />
      </section>

      <section className="py-12 md:py-16">
        <div className="mx-auto max-w-[880px] px-5 md:px-10 xl:px-14">
          <div
            className="rounded-[20px] p-8 md:p-10"
            style={{
              backgroundColor: 'var(--panel)',
              border: '1px solid rgba(38,58,60,0.08)',
              boxShadow: '0 14px 34px rgba(34,51,59,0.07)',
            }}
          >
            <div className="flex flex-col gap-8 md:flex-row md:items-end md:justify-between">
              <div>
                <p
                  className="text-[13px] font-semibold uppercase tracking-[0.08em]"
                  style={{ color: 'var(--coral)' }}
                >
                  Number of students
                </p>
                <p
                  className="font-display mt-2 text-[clamp(40px,6vw,56px)] font-semibold leading-none tabular-nums"
                  style={{ color: 'var(--ink)' }}
                  aria-live="polite"
                >
                  {formatUsers(users)}
                </p>
                <p className="mt-2 text-[14px]" style={{ color: 'var(--text-soft)' }}>
                  {tierLabel}
                </p>
              </div>

              <div className="text-left md:text-right">
                <p
                  id={rateId}
                  className="text-[13px] font-semibold uppercase tracking-[0.08em]"
                  style={{ color: 'var(--muted)' }}
                >
                  Price per student
                </p>
                <p
                  className="font-display mt-2 text-[clamp(32px,4vw,44px)] font-semibold leading-none tabular-nums"
                  style={{ color: 'var(--teal-deep)' }}
                  aria-live="polite"
                >
                  {formatUsd(rate, { forceCents: true })}
                </p>
                <p className="mt-2 text-[14px]" style={{ color: 'var(--text-soft)' }}>
                  per student / year
                </p>
              </div>
            </div>

            <div className="mt-10">
              <label htmlFor={sliderId} className="sr-only">
                Number of students, from {formatUsers(CALCULATOR_MIN_USERS)} to{' '}
                {formatUsers(CALCULATOR_MAX_USERS)}
              </label>
              <input
                id={sliderId}
                type="range"
                min={CALCULATOR_MIN_USERS}
                max={CALCULATOR_MAX_USERS}
                step={CALCULATOR_STEP}
                value={users}
                onChange={e => setUsers(Number(e.target.value))}
                className="pricing-slider w-full"
                style={{
                  // CSS custom property drives the filled track via index.css
                  ['--slider-fill' as string]: `${fill}%`,
                }}
                aria-valuemin={CALCULATOR_MIN_USERS}
                aria-valuemax={CALCULATOR_MAX_USERS}
                aria-valuenow={users}
                aria-valuetext={`${formatUsers(users)} students, ${formatUsd(rate, { forceCents: true })} per student`}
                aria-describedby={rateId}
              />
              <div
                className="mt-3 flex justify-between text-[12px] font-medium tabular-nums"
                style={{ color: 'var(--muted)' }}
              >
                <span>{formatUsers(CALCULATOR_MIN_USERS)}</span>
                <span>15k</span>
                <span>25k</span>
                <span>50k</span>
                <span>{formatUsers(CALCULATOR_MAX_USERS)}</span>
              </div>
            </div>

            <div
              className="mt-10 flex flex-col gap-4 rounded-[14px] px-6 py-5 sm:flex-row sm:items-center sm:justify-between"
              style={{ backgroundColor: 'var(--teal-tint)' }}
            >
              <div>
                <p
                  className="text-[13px] font-semibold uppercase tracking-[0.08em]"
                  style={{ color: 'var(--teal-deep)' }}
                >
                  Estimated annual total
                </p>
                <p
                  className="font-display mt-1 text-[clamp(28px,3.5vw,36px)] font-semibold tabular-nums"
                  style={{ color: 'var(--ink)' }}
                  aria-live="polite"
                >
                  {formatUsd(total)}
                </p>
              </div>
              <p className="max-w-[280px] text-[13px] leading-relaxed" style={{ color: 'var(--text-soft)' }}>
                Estimate only. Final quotes may adjust for hosting model, SSO, and support level.
              </p>
            </div>
          </div>

          <div className="mt-10">
            <h2
              className="font-display text-[clamp(22px,2.5vw,28px)] font-semibold"
              style={{ color: 'var(--ink)' }}
            >
              Bulk discount tiers
            </h2>
            <div className="mt-5 overflow-hidden rounded-[14px]" style={{ border: '1px solid var(--line)' }}>
              <table className="w-full text-left text-[14px]">
                <thead>
                  <tr style={{ backgroundColor: 'var(--panel-sunken)' }}>
                    <th className="px-5 py-3 font-semibold" style={{ color: 'var(--ink-nav)' }}>
                      Students
                    </th>
                    <th className="px-5 py-3 font-semibold" style={{ color: 'var(--ink-nav)' }}>
                      Per student / year
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {PRICING_TIERS.map(tier => {
                    const active = pricePerStudent(users) === tier.pricePerStudent
                    return (
                      <tr
                        key={tier.label}
                        style={{
                          backgroundColor: active ? 'rgba(106,197,176,0.12)' : 'var(--panel)',
                          borderTop: '1px solid var(--line)',
                        }}
                      >
                        <td className="px-5 py-3.5" style={{ color: 'var(--text)' }}>
                          {tier.label}
                          {active && (
                            <span
                              className="ml-2 rounded-full px-2 py-0.5 text-[11px] font-semibold uppercase tracking-[0.06em]"
                              style={{ backgroundColor: 'rgba(106,197,176,0.2)', color: '#4fa894' }}
                            >
                              Current
                            </span>
                          )}
                        </td>
                        <td
                          className="px-5 py-3.5 font-semibold tabular-nums"
                          style={{ color: 'var(--ink)' }}
                        >
                          {formatUsd(tier.pricePerStudent, { forceCents: true })}
                        </td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>
          </div>

          <div className="mt-12 flex flex-col items-start gap-3.5 sm:flex-row sm:items-center">
            <a href="/request-information" className="btn-primary gap-2">
              Request information
              <ArrowRight className="h-4 w-4" aria-hidden />
            </a>
            <a href="/pricing" className="btn-secondary">
              Compare all plans
            </a>
          </div>
        </div>
      </section>
    </MarketingPageShell>
  )
}
