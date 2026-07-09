import { type FormEvent, useCallback, useEffect, useId, useMemo, useState } from 'react'
import { Loader2, Save } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useConfirm } from '../../components/use-confirm'
import { CourseHeroImage } from '../../components/course-hero-image'
import { usePermissions } from '../../context/use-permissions'
import { courseItemCreatePermission } from '../../lib/courses-api'
import {
  fetchCourseCatalogListing,
  putCourseCatalogListing,
  type CourseCatalogListing,
} from '../../lib/course-catalog-listing-api'
import { toastMutationError, toastSaveOk } from '../../lib/lms-toast'
import {
  formatMarketplacePrice,
  majorUnitsToPriceCents,
  MARKETPLACE_CURRENCIES,
  priceCentsToMajorUnits,
  validateMarketplaceAmount,
} from '../../lib/marketplace-price'

type CourseMarketplaceSettingsSectionProps = {
  courseCode: string
  courseTitle: string
  heroImageUrl: string | null
}

export function CourseMarketplaceSettingsSection({
  courseCode,
  courseTitle,
  heroImageUrl,
}: CourseMarketplaceSettingsSectionProps) {
  const { t, i18n } = useTranslation('common')
  const { allows } = usePermissions()
  const { confirm, ConfirmDialogHost } = useConfirm()
  const feeHelpId = useId()
  const feeErrorId = useId()

  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [listing, setListing] = useState<CourseCatalogListing | null>(null)
  const [marketplaceListed, setMarketplaceListed] = useState(false)
  const [amount, setAmount] = useState('')
  const [currency, setCurrency] = useState('usd')
  const [amountError, setAmountError] = useState<string | null>(null)

  const canEdit = allows(courseItemCreatePermission(courseCode))
  const isDraft = listing?.publishState === 'draft'

  const reload = useCallback(async () => {
    setLoading(true)
    try {
      const data = await fetchCourseCatalogListing(courseCode)
      setListing(data)
      setMarketplaceListed(data.marketplaceListed)
      setAmount(priceCentsToMajorUnits(data.priceCents))
      setCurrency(data.priceCurrency || 'usd')
      setAmountError(null)
    } catch (e) {
      toastMutationError(e instanceof Error ? e.message : t('course.settings.marketplace.loadError'))
    } finally {
      setLoading(false)
    }
  }, [courseCode, t])

  useEffect(() => {
    void reload()
  }, [reload])

  const priceCentsDraft = useMemo(() => {
    if (!amount.trim()) return 0
    return majorUnitsToPriceCents(amount) ?? listing?.priceCents ?? 0
  }, [amount, listing?.priceCents])

  const previewPriceLabel = formatMarketplacePrice(
    priceCentsDraft,
    currency,
    i18n.language,
    t('course.settings.marketplace.free'),
  )

  const isDirty = useMemo(() => {
    if (!listing) return false
    const nextCents = amount.trim() ? (majorUnitsToPriceCents(amount) ?? listing.priceCents) : 0
    return (
      marketplaceListed !== listing.marketplaceListed ||
      nextCents !== listing.priceCents ||
      currency !== (listing.priceCurrency || 'usd')
    )
  }, [listing, marketplaceListed, amount, currency])

  async function persistListing() {
    if (!listing) return
    const nextCents = amount.trim() ? (majorUnitsToPriceCents(amount) ?? listing.priceCents) : 0
    const updated = await putCourseCatalogListing(
      courseCode,
      {
        marketplaceListed,
        priceCents: nextCents,
        priceCurrency: currency,
      },
      listing,
    )
    setListing(updated)
    setMarketplaceListed(updated.marketplaceListed)
    setAmount(priceCentsToMajorUnits(updated.priceCents))
    setCurrency(updated.priceCurrency || 'usd')
    toastSaveOk(t('course.settings.marketplace.saved'))
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    if (!listing || !canEdit) return

    const validationError = validateMarketplaceAmount(amount)
    if (validationError) {
      setAmountError(validationError)
      return
    }
    setAmountError(null)

    const nextCents = amount.trim() ? (majorUnitsToPriceCents(amount) ?? 0) : 0
    const priceChanged = nextCents !== listing.priceCents || currency !== (listing.priceCurrency || 'usd')
    if (priceChanged && listing.activePurchaseCount > 0) {
      const ok = await confirm({
        title: t('course.settings.marketplace.priceChangeTitle'),
        description: t('course.settings.marketplace.priceChangeWarning'),
        confirmLabel: t('course.settings.marketplace.priceChangeConfirm'),
      })
      if (!ok) return
    }

    setSaving(true)
    try {
      await persistListing()
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : t('course.settings.marketplace.saveError'))
      void reload()
    } finally {
      setSaving(false)
    }
  }

  if (loading) {
    return (
      <section
        className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm shadow-slate-900/5 dark:border-neutral-800 dark:bg-neutral-900"
        aria-busy="true"
      >
        <p className="flex items-center gap-2 text-sm text-slate-600 dark:text-neutral-300">
          <Loader2 className="h-4 w-4 motion-safe:animate-spin" aria-hidden />
          {t('course.settings.marketplace.loading')}
        </p>
      </section>
    )
  }

  if (!listing) {
    return null
  }

  return (
    <>
      <form
        onSubmit={(e) => void onSubmit(e)}
        className="space-y-6 rounded-2xl border border-slate-200 bg-white p-5 shadow-sm shadow-slate-900/5 dark:border-neutral-800 dark:bg-neutral-900"
      >
        <div>
          <h2 className="text-sm font-semibold text-slate-900 dark:text-neutral-50">
            {t('course.settings.marketplace.title')}
          </h2>
          <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">
            {t('course.settings.marketplace.description')}
          </p>
        </div>

        <div className="border-t border-slate-100 pt-4 dark:border-neutral-800">
          <div className="flex flex-wrap items-start justify-between gap-4">
            <div className="min-w-0 flex-1">
              <p className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
                {t('course.settings.marketplace.listToggle')}
              </p>
              {isDraft ? (
                <p className="mt-1 text-sm text-amber-800 dark:text-amber-200" id={feeHelpId}>
                  {t('course.settings.marketplace.publishFirst')}
                </p>
              ) : (
                <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">
                  {t('course.settings.marketplace.listHelp')}
                </p>
              )}
            </div>
            <button
              type="button"
              role="switch"
              aria-checked={marketplaceListed}
              aria-label={t('course.settings.marketplace.listToggle')}
              aria-describedby={isDraft ? feeHelpId : undefined}
              disabled={!canEdit || saving || isDraft}
              onClick={() => setMarketplaceListed((v) => !v)}
              className={`relative mt-0.5 inline-flex h-7 w-12 shrink-0 rounded-full border-2 border-transparent transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500 focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 ${
                marketplaceListed ? 'bg-indigo-600' : 'bg-slate-200 dark:bg-neutral-700'
              }`}
            >
              <span
                className={`pointer-events-none inline-block h-6 w-6 transform rounded-full bg-white shadow ring-0 transition-transform ${
                  marketplaceListed ? 'translate-x-5' : 'translate-x-0.5'
                }`}
              />
            </button>
          </div>
        </div>

        <div className="grid gap-4 sm:grid-cols-[minmax(0,1fr)_10rem]">
          <label className="block">
            <span className="mb-1.5 block text-sm font-medium text-slate-700 dark:text-neutral-300">
              {t('course.settings.marketplace.fee')}
            </span>
            <input
              type="text"
              inputMode="decimal"
              value={amount}
              onChange={(e) => {
                setAmount(e.target.value)
                setAmountError(null)
              }}
              disabled={!canEdit || saving}
              placeholder={t('course.settings.marketplace.free')}
              aria-describedby={amountError ? `${feeHelpId} ${feeErrorId}` : feeHelpId}
              aria-invalid={amountError ? true : undefined}
              className="w-full rounded-xl border border-slate-200 bg-white px-3 py-2.5 text-sm text-slate-900 outline-none ring-indigo-500/20 placeholder:text-slate-400 focus:border-indigo-400 focus:ring-2 dark:border-neutral-700 dark:bg-neutral-950 dark:text-neutral-50"
            />
            <p id={feeHelpId} className="mt-1.5 text-xs text-slate-500 dark:text-neutral-400">
              {t('course.settings.marketplace.feeHelp')}
            </p>
            {amountError ? (
              <p id={feeErrorId} className="mt-1 text-sm text-rose-700 dark:text-rose-400" role="alert">
                {amountError}
              </p>
            ) : null}
          </label>
          <label className="block">
            <span className="mb-1.5 block text-sm font-medium text-slate-700 dark:text-neutral-300">
              {t('course.settings.marketplace.currency')}
            </span>
            <select
              value={currency}
              onChange={(e) => setCurrency(e.target.value)}
              disabled={!canEdit || saving}
              className="w-full rounded-xl border border-slate-200 bg-white px-2 py-2.5 text-sm text-slate-900 outline-none ring-indigo-500/20 focus:border-indigo-400 focus:ring-2 dark:border-neutral-700 dark:bg-neutral-950 dark:text-neutral-50"
            >
              {MARKETPLACE_CURRENCIES.map((c) => (
                <option key={c.code} value={c.code}>
                  {c.label}
                </option>
              ))}
            </select>
          </label>
        </div>

        <div>
          <h3 className="text-sm font-semibold text-slate-900 dark:text-neutral-50">
            {t('course.settings.marketplace.preview')}
          </h3>
          <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">
            {t('course.settings.marketplace.previewHelp')}
          </p>
          <article
            className="relative mt-4 flex max-w-sm flex-col overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm dark:border-neutral-800 dark:bg-neutral-950"
            aria-label={t('course.settings.marketplace.preview')}
          >
            {heroImageUrl ? (
              <CourseHeroImage
                src={heroImageUrl}
                alt=""
                className="h-32 w-full object-cover"
                loading="lazy"
              />
            ) : (
              <div className="h-32 w-full bg-gradient-to-br from-indigo-100 to-sky-100 dark:from-indigo-950 dark:to-sky-950" />
            )}
            <div className="flex flex-1 flex-col gap-2 p-4">
              <div className="flex flex-wrap items-center gap-2 text-xs text-slate-500 dark:text-neutral-400">
                {listing.category ? <span>{listing.category}</span> : null}
                {listing.difficultyLevel ? (
                  <span className="rounded-full bg-slate-100 px-2 py-0.5 capitalize dark:bg-neutral-800">
                    {listing.difficultyLevel}
                  </span>
                ) : null}
              </div>
              <h4 className="text-base font-semibold text-slate-900 dark:text-neutral-100">{courseTitle}</h4>
              <p className="mt-auto text-sm font-semibold text-slate-900 dark:text-neutral-100" aria-live="polite">
                {previewPriceLabel}
              </p>
            </div>
          </article>
        </div>

        {canEdit ? (
          <div className="flex flex-wrap items-center gap-3 border-t border-slate-100 pt-4 dark:border-neutral-800">
            <button
              type="submit"
              disabled={saving || !isDirty}
              className="inline-flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-semibold text-white hover:bg-indigo-500 disabled:cursor-not-allowed disabled:opacity-50"
            >
              {saving ? (
                <Loader2 className="h-4 w-4 motion-safe:animate-spin" aria-hidden />
              ) : (
                <Save className="h-4 w-4" aria-hidden />
              )}
              {saving ? t('course.settings.marketplace.saving') : t('course.settings.marketplace.save')}
            </button>
          </div>
        ) : null}
      </form>
      {ConfirmDialogHost}
    </>
  )
}
