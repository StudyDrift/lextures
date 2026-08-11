import { useEffect, useId, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Button,
  DatePicker,
  Dialog,
  Field,
  Input,
  SegmentedControl,
  Textarea,
} from '../../components/ui'
import {
  CourseCouponApiError,
  createCourseCoupon,
  generateCouponCode,
  normalizeCouponCode,
  previewCouponPrice,
  updateCourseCoupon,
  validateCouponDraft,
  type ClientCouponValidation,
  type CourseCoupon,
  type CouponDiscountType,
  type CreateCourseCouponBody,
  type UpdateCourseCouponBody,
} from '../../lib/course-coupons-api'
import {
  datetimeLocalValueToIso,
  detectBrowserTimeZone,
  isoToDatetimeLocalValue,
} from '../../lib/format'
import { formatMarketplacePrice, majorUnitsToPriceCents } from '../../lib/marketplace-price'
import { toastMutationError, toastSaveOk } from '../../lib/lms-toast'
import { emitCouponManagerTelemetry } from '../../lib/coupon-manager-telemetry'
import { useConfirm } from '../../components/use-confirm'

export type CourseCouponCreateDialogProps = {
  open: boolean
  onClose: () => void
  courseCode: string
  coursePriceCents: number
  courseCurrency: string
  /** When set, dialog is in edit mode (window / limits / note / status only). */
  editing?: CourseCoupon | null
  onCreated: (coupon: CourseCoupon) => void
  onUpdated: (coupon: CourseCoupon) => void
}

type FormState = {
  code: string
  discountType: CouponDiscountType
  percentOff: string
  amountOff: string
  startsAt: string
  endsAt: string
  maxRedemptions: string
  maxRedemptionsPerUser: string
  note: string
}

const emptyForm = (): FormState => ({
  code: '',
  discountType: 'percent',
  percentOff: '25',
  amountOff: '',
  startsAt: '',
  endsAt: '',
  maxRedemptions: '',
  maxRedemptionsPerUser: '1',
  note: '',
})

function formFromCoupon(c: CourseCoupon): FormState {
  return {
    code: c.code,
    discountType: c.discountType,
    percentOff: c.percentOff != null ? String(c.percentOff) : '',
    amountOff: '',
    startsAt: isoToDatetimeLocalValue(c.startsAt),
    endsAt: isoToDatetimeLocalValue(c.endsAt),
    maxRedemptions: c.maxRedemptions != null ? String(c.maxRedemptions) : '',
    maxRedemptionsPerUser: String(c.maxRedemptionsPerUser ?? 1),
    note: c.note ?? '',
  }
}

export function CourseCouponCreateDialog({
  open,
  onClose,
  courseCode,
  coursePriceCents,
  courseCurrency,
  editing = null,
  onCreated,
  onUpdated,
}: CourseCouponCreateDialogProps) {
  const { t, i18n } = useTranslation('common')
  const { confirm, ConfirmDialogHost } = useConfirm()
  const tz = detectBrowserTimeZone()
  const [form, setForm] = useState<FormState>(emptyForm)
  const [fieldErrors, setFieldErrors] = useState<ClientCouponValidation>({})
  const [submitting, setSubmitting] = useState(false)
  const formId = useId()
  const isEdit = Boolean(editing)

  useEffect(() => {
    if (!open) return
    setFieldErrors({})
    setForm(editing ? formFromCoupon(editing) : emptyForm())
  }, [open, editing])

  const amountOffCents = useMemo(() => {
    if (form.discountType !== 'fixed') return null
    return majorUnitsToPriceCents(form.amountOff, courseCurrency)
  }, [form.discountType, form.amountOff, courseCurrency])

  const preview = useMemo(() => {
    if (isEdit && editing) {
      return previewCouponPrice(coursePriceCents, courseCurrency, {
        discountType: editing.discountType,
        percentOff: editing.percentOff,
        amountOffCents: editing.amountOffCents,
      })
    }
    return previewCouponPrice(coursePriceCents, courseCurrency, {
      discountType: form.discountType,
      percentOff: Number.parseFloat(form.percentOff) || 0,
      amountOffCents: amountOffCents ?? 0,
    })
  }, [isEdit, editing, coursePriceCents, courseCurrency, form, amountOffCents])

  const previewLabel = t('course.settings.coupons.preview', {
    now: formatMarketplacePrice(
      preview.chargedCents,
      courseCurrency,
      i18n.language,
      t('course.settings.marketplace.free'),
    ),
    before: formatMarketplacePrice(
      coursePriceCents,
      courseCurrency,
      i18n.language,
      t('course.settings.marketplace.free'),
    ),
  })

  function setField<K extends keyof FormState>(key: K, value: FormState[K]) {
    setForm((prev) => ({ ...prev, [key]: value }))
    setFieldErrors((prev) => {
      if (!prev[key as keyof ClientCouponValidation]) return prev
      const next = { ...prev }
      delete next[key as keyof ClientCouponValidation]
      return next
    })
  }

  function clientValidate(): boolean {
    if (isEdit) {
      const errors: ClientCouponValidation = {}
      if (form.startsAt && form.endsAt) {
        const s = new Date(form.startsAt).getTime()
        const e = new Date(form.endsAt).getTime()
        if (Number.isFinite(s) && Number.isFinite(e) && e <= s) {
          errors.endsAt = t('course.settings.coupons.error.endBeforeStart')
        }
      }
      if (form.maxRedemptions.trim()) {
        const n = Number.parseInt(form.maxRedemptions, 10)
        if (!Number.isFinite(n) || n <= 0) {
          errors.maxRedemptions = t('course.settings.coupons.error.maxRedemptionsPositive')
        }
      }
      if (form.maxRedemptionsPerUser.trim()) {
        const n = Number.parseInt(form.maxRedemptionsPerUser, 10)
        if (!Number.isFinite(n) || n < 1 || n > 100) {
          errors.maxRedemptionsPerUser = t('course.settings.coupons.error.perUserRange')
        }
      }
      setFieldErrors(errors)
      return Object.keys(errors).length === 0
    }

    const errors = validateCouponDraft({
      code: form.code,
      discountType: form.discountType,
      percentOff: form.percentOff,
      amountOffMajor: form.amountOff,
      amountOffCents,
      coursePriceCents,
      startsAtLocal: form.startsAt,
      endsAtLocal: form.endsAt,
      maxRedemptions: form.maxRedemptions,
      maxRedemptionsPerUser: form.maxRedemptionsPerUser,
      messages: {
        codeShape: t('course.settings.coupons.error.codeShape'),
        percentRange: t('course.settings.coupons.error.percentRange'),
        amountPositive: t('course.settings.coupons.error.amountPositive'),
        amountExceedsPrice: t('course.settings.coupons.error.amountExceedsPrice'),
        endBeforeStart: t('course.settings.coupons.error.endBeforeStart'),
        maxRedemptionsPositive: t('course.settings.coupons.error.maxRedemptionsPositive'),
        perUserRange: t('course.settings.coupons.error.perUserRange'),
      },
    })
    setFieldErrors(errors)
    return Object.keys(errors).length === 0
  }

  async function onSubmit() {
    if (!clientValidate() || submitting) return

    if (!isEdit && preview.free) {
      const ok = await confirm({
        title: t('course.settings.coupons.previewFreeConfirmTitle'),
        description: t('course.settings.coupons.previewFree'),
        confirmLabel: t('course.settings.coupons.createAnyway'),
        variant: 'default',
      })
      if (!ok) return
    }

    setSubmitting(true)
    try {
      if (isEdit && editing) {
        const body: UpdateCourseCouponBody = {
          startsAt: form.startsAt ? datetimeLocalValueToIso(form.startsAt) : null,
          endsAt: form.endsAt ? datetimeLocalValueToIso(form.endsAt) : null,
          maxRedemptions: form.maxRedemptions.trim()
            ? Number.parseInt(form.maxRedemptions, 10)
            : null,
          maxRedemptionsPerUser: form.maxRedemptionsPerUser.trim()
            ? Number.parseInt(form.maxRedemptionsPerUser, 10)
            : 1,
          note: form.note.trim() || null,
        }
        const updated = await updateCourseCoupon(courseCode, editing.id, body)
        onUpdated(updated)
        toastSaveOk(t('course.settings.coupons.updated'))
        onClose()
      } else {
        const body: CreateCourseCouponBody = {
          code: normalizeCouponCode(form.code),
          discountType: form.discountType,
          startsAt: form.startsAt ? datetimeLocalValueToIso(form.startsAt) : null,
          endsAt: form.endsAt ? datetimeLocalValueToIso(form.endsAt) : null,
          maxRedemptions: form.maxRedemptions.trim()
            ? Number.parseInt(form.maxRedemptions, 10)
            : null,
          maxRedemptionsPerUser: form.maxRedemptionsPerUser.trim()
            ? Number.parseInt(form.maxRedemptionsPerUser, 10)
            : 1,
          note: form.note.trim() || null,
        }
        if (form.discountType === 'percent') {
          body.percentOff = Number.parseFloat(form.percentOff)
        } else {
          body.amountOffCents = amountOffCents ?? 0
          body.currency = courseCurrency
        }
        const created = await createCourseCoupon(courseCode, body)
        emitCouponManagerTelemetry('coupon_created', { discountType: form.discountType })
        onCreated(created)
        toastSaveOk(t('course.settings.coupons.created'))
        const warnings = (created.warnings ?? []) as string[]
        if (warnings.includes('low_entropy')) {
          toastSaveOk(t('course.settings.coupons.lowEntropyWarning'))
        }
        onClose()
      }
    } catch (e) {
      if (e instanceof CourseCouponApiError) {
        const next: ClientCouponValidation = { ...e.fields }
        if (e.status === 409) {
          next.code = e.fields.code || t('course.settings.coupons.error.duplicate')
        }
        if (Object.keys(next).length > 0) {
          setFieldErrors(next)
        } else {
          toastMutationError(e.message || t('course.settings.coupons.error.save'))
        }
      } else {
        toastMutationError(e instanceof Error ? e.message : t('course.settings.coupons.error.save'))
      }
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <>
      <Dialog
        open={open}
        onClose={() => {
          if (!submitting) onClose()
        }}
        title={isEdit ? t('course.settings.coupons.editTitle', { code: editing?.code }) : t('course.settings.coupons.new')}
        description={
          isEdit
            ? t('course.settings.coupons.editDescription')
            : t('course.settings.coupons.createDescription')
        }
        closeLabel={t('dialogs.close')}
        size="lg"
        footer={
          <div className="flex flex-wrap justify-end gap-2">
            <Button type="button" variant="secondary" disabled={submitting} onClick={onClose}>
              {t('dialogs.cancel')}
            </Button>
            <Button
              type="submit"
              form={formId}
              variant="primary"
              loading={submitting}
              disabled={submitting}
            >
              {isEdit ? t('course.settings.coupons.save') : t('course.settings.coupons.create')}
            </Button>
          </div>
        }
      >
        <form
          id={formId}
          className="space-y-4"
          onSubmit={(e) => {
            e.preventDefault()
            void onSubmit()
          }}
        >
          <Field
            label={t('course.settings.coupons.code')}
            description={t('course.settings.coupons.codeHelp')}
            error={fieldErrors.code}
            required={!isEdit}
          >
            <div className="flex flex-wrap gap-2">
              <Input
                value={form.code}
                onChange={(e) => setField('code', normalizeCouponCode(e.target.value))}
                disabled={isEdit || submitting}
                autoComplete="off"
                spellCheck={false}
                className="min-w-0 flex-1 font-mono uppercase tracking-wide"
                maxLength={32}
              />
              {!isEdit ? (
                <Button
                  type="button"
                  variant="secondary"
                  disabled={submitting}
                  onClick={() => setField('code', generateCouponCode(8))}
                >
                  {t('course.settings.coupons.generate')}
                </Button>
              ) : null}
            </div>
          </Field>

          {!isEdit ? (
            <>
              <SegmentedControl
                label={t('course.settings.coupons.discount')}
                value={form.discountType}
                onChange={(v) => setField('discountType', v)}
                options={[
                  { value: 'percent', label: t('course.settings.coupons.discountPercent') },
                  { value: 'fixed', label: t('course.settings.coupons.discountFixed') },
                ]}
              />

              {form.discountType === 'percent' ? (
                <Field
                  label={t('course.settings.coupons.percent')}
                  error={fieldErrors.percentOff}
                  required
                >
                  <Input
                    type="number"
                    inputMode="decimal"
                    min={0.01}
                    max={100}
                    step="any"
                    value={form.percentOff}
                    onChange={(e) => setField('percentOff', e.target.value)}
                    disabled={submitting}
                  />
                </Field>
              ) : (
                <Field
                  label={t('course.settings.coupons.amount', {
                    currency: courseCurrency.toUpperCase(),
                  })}
                  error={fieldErrors.amountOff}
                  required
                >
                  <Input
                    type="text"
                    inputMode="decimal"
                    value={form.amountOff}
                    onChange={(e) => setField('amountOff', e.target.value)}
                    disabled={submitting}
                  />
                </Field>
              )}
            </>
          ) : null}

          <div className="grid gap-4 sm:grid-cols-2">
            <Field
              label={t('course.settings.coupons.startsAt')}
              description={t('course.settings.coupons.timezoneHint', { tz })}
            >
              <DatePicker
                type="datetime-local"
                value={form.startsAt}
                onChange={(e) => setField('startsAt', e.target.value)}
                disabled={submitting}
              />
            </Field>
            <Field
              label={t('course.settings.coupons.endsAt')}
              error={fieldErrors.endsAt}
              description={t('course.settings.coupons.timezoneHint', { tz })}
            >
              <DatePicker
                type="datetime-local"
                value={form.endsAt}
                onChange={(e) => setField('endsAt', e.target.value)}
                disabled={submitting}
              />
            </Field>
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <Field
              label={t('course.settings.coupons.maxRedemptions')}
              description={t('course.settings.coupons.maxRedemptionsHelp')}
              error={fieldErrors.maxRedemptions}
            >
              <Input
                type="number"
                inputMode="numeric"
                min={1}
                value={form.maxRedemptions}
                onChange={(e) => setField('maxRedemptions', e.target.value)}
                disabled={submitting}
                placeholder={t('course.settings.coupons.unlimited')}
              />
            </Field>
            <Field
              label={t('course.settings.coupons.perLearner')}
              description={t('course.settings.coupons.perLearnerHelp')}
              error={fieldErrors.maxRedemptionsPerUser}
            >
              <Input
                type="number"
                inputMode="numeric"
                min={1}
                max={100}
                value={form.maxRedemptionsPerUser}
                onChange={(e) => setField('maxRedemptionsPerUser', e.target.value)}
                disabled={submitting}
              />
            </Field>
          </div>

          <Field label={t('course.settings.coupons.note')} description={t('course.settings.coupons.noteHelp')}>
            <Textarea
              value={form.note}
              onChange={(e) => setField('note', e.target.value)}
              disabled={submitting}
              rows={2}
              maxLength={500}
            />
          </Field>

          <div
            className="rounded-xl border border-border-default bg-surface-sunken px-4 py-3 text-sm"
            aria-live="polite"
          >
            <p className="font-medium text-fg-default">{previewLabel}</p>
            {preview.free ? (
              <p className="mt-1 text-warning-fg" role="status">
                {t('course.settings.coupons.previewFree')}
              </p>
            ) : null}
          </div>
        </form>
      </Dialog>
      {ConfirmDialogHost}
    </>
  )
}
