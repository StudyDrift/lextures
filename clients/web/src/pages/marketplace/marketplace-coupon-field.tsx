/**
 * Learner coupon entry control (plan MKTC.5).
 * Disclosure + Field/Input + Apply/Remove; parent owns preview application.
 */

import { useEffect, useId, useRef, useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { Button, Disclosure, Field, Input } from '../../components/ui'
import {
  COUPON_INPUT_MAX_LEN,
  normalizeCouponInput,
  type CouponReason,
} from '../../lib/marketplace-coupon'
import type { MarketplaceCouponPreview } from '../../lib/marketplace-api'
import { formatMarketplacePrice } from '../../lib/marketplace-price'
import { emitLearnerCouponTelemetry } from '../../lib/marketplace-learner-coupon-telemetry'

export type CouponFieldStatus = 'idle' | 'checking' | 'applied' | 'rejected' | 'rate_limited'

export type MarketplaceCouponFieldProps = {
  slug: string
  /** When true, disclosure starts open (URL/pending code present). */
  defaultOpen?: boolean
  initialCode?: string
  status: CouponFieldStatus
  preview: MarketplaceCouponPreview | null
  errorReason?: CouponReason | null
  rateLimited?: boolean
  disabled?: boolean
  onApply: (code: string) => void | Promise<void>
  onRemove: () => void
  locale?: string
}

export function MarketplaceCouponField({
  slug: _slug,
  defaultOpen = false,
  initialCode = '',
  status,
  preview,
  errorReason,
  rateLimited = false,
  disabled = false,
  onApply,
  onRemove,
  locale,
}: MarketplaceCouponFieldProps) {
  const { t, i18n } = useTranslation('common')
  const [open, setOpen] = useState(defaultOpen || Boolean(initialCode) || status === 'applied')
  const [value, setValue] = useState(() => normalizeCouponInput(initialCode))
  const inputRef = useRef<HTMLInputElement>(null)
  const helpId = useId()
  const activeLocale = locale ?? i18n.language

  useEffect(() => {
    if (initialCode) {
      setValue(normalizeCouponInput(initialCode))
      setOpen(true)
    }
  }, [initialCode])

  useEffect(() => {
    if (status === 'applied' || status === 'rejected' || status === 'rate_limited') {
      setOpen(true)
    }
  }, [status])

  const checking = status === 'checking'
  const applied = status === 'applied' && preview?.applied
  const rejected = status === 'rejected' || status === 'rate_limited'
  const busy = checking || disabled

  let errorText: string | null = null
  if (status === 'rate_limited' || rateLimited) {
    // Cool-down (10 failed applies) uses a distinct reason; generic 429 uses rateLimited.
    if (errorReason === 'cooldown') {
      errorText = t('marketplace.coupon.cooldown')
    } else {
      errorText = t('marketplace.coupon.rateLimited')
    }
  } else if (rejected && errorReason) {
    errorText = t(`marketplace.coupon.reason.${errorReason}`, {
      defaultValue: t('marketplace.coupon.reason.not_found'),
    })
  }

  function handleOpenChange(next: boolean) {
    setOpen(next)
    if (next) {
      emitLearnerCouponTelemetry('coupon_field_opened')
    }
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    if (busy || applied) return
    const code = normalizeCouponInput(value)
    if (!code) return
    setValue(code)
    await onApply(code)
  }

  function handleRemove() {
    onRemove()
    setValue('')
    // Focus returns after parent clears state (FR-4).
    requestAnimationFrame(() => {
      inputRef.current?.focus()
    })
  }

  const savingsLine =
    applied && preview && preview.discountCents > 0
      ? t('marketplace.coupon.savings', {
          amount: formatMarketplacePrice(
            preview.discountCents,
            preview.currency,
            activeLocale,
            '',
          ),
        })
      : null

  const seatsLine =
    applied && preview?.seatsRemaining != null
      ? t('marketplace.coupon.seatsLeft', { count: preview.seatsRemaining })
      : null

  const endsLine =
    applied && preview?.endsAt
      ? t('marketplace.coupon.endsSoon', {
          date: new Date(preview.endsAt).toLocaleDateString(activeLocale, {
            dateStyle: 'medium',
          }),
        })
      : null

  return (
    <div data-testid="marketplace-coupon-field" className="w-full">
      <Disclosure
        title={t('marketplace.coupon.disclosure')}
        open={open}
        onOpenChange={handleOpenChange}
        className="border-border-default bg-surface-raised"
      >
        <form onSubmit={(e) => void handleSubmit(e)} className="flex flex-col gap-3">
          <Field
            label={t('marketplace.coupon.label')}
            description={t('marketplace.coupon.placeholder')}
            error={
              errorText ? (
                <span role="alert" data-testid="marketplace-coupon-error">
                  {errorText}
                </span>
              ) : undefined
            }
            busy={checking}
          >
            <div className="flex flex-col gap-2 sm:flex-row sm:items-start">
              <Input
                ref={inputRef}
                name="coupon"
                autoComplete="off"
                spellCheck={false}
                maxLength={COUPON_INPUT_MAX_LEN}
                value={value}
                disabled={busy || Boolean(applied)}
                onChange={(e) => setValue(normalizeCouponInput(e.target.value))}
                placeholder={t('marketplace.coupon.placeholder')}
                aria-describedby={helpId}
                data-testid="marketplace-coupon-input"
                className="font-mono uppercase tracking-wide"
              />
              {applied ? (
                <Button
                  type="button"
                  variant="secondary"
                  onClick={handleRemove}
                  disabled={disabled}
                  data-testid="marketplace-coupon-remove"
                >
                  {t('marketplace.coupon.remove')}
                </Button>
              ) : (
                <Button
                  type="submit"
                  variant="secondary"
                  loading={checking}
                  disabled={busy || !value || rateLimited}
                  data-testid="marketplace-coupon-apply"
                >
                  {checking ? t('marketplace.coupon.applying') : t('marketplace.coupon.apply')}
                </Button>
              )}
            </div>
          </Field>
          <p id={helpId} className="sr-only">
            {t('marketplace.coupon.placeholder')}
          </p>
          {applied && preview ? (
            <div className="space-y-1 text-sm text-success-fg" data-testid="marketplace-coupon-applied">
              {savingsLine ? <p>{savingsLine}</p> : null}
              {seatsLine ? <p className="text-fg-muted">{seatsLine}</p> : null}
              {endsLine ? <p className="text-fg-muted">{endsLine}</p> : null}
            </div>
          ) : null}
        </form>
      </Disclosure>
    </div>
  )
}
