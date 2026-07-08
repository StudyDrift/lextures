import { useEffect, useRef } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { ArrowRight, X } from 'lucide-react'
import { useIntroCourseProgress } from '../../hooks/use-intro-course-progress'
import {
  dismissIntroWelcomeBanner,
  introCourseFallbackHref,
  shouldShowIntroWelcomeBanner,
} from '../../lib/intro-course-api'
import { recordIntroCourseBannerDismiss, recordIntroCourseCtaClick } from '../../lib/intro-course-observability'

export function IntroWelcomeBanner() {
  const { t } = useTranslation('introCourse')
  const { progress, loading, refresh } = useIntroCourseProgress()
  const dismissButtonRef = useRef<HTMLButtonElement>(null)

  const visible = !loading && shouldShowIntroWelcomeBanner(progress)

  useEffect(() => {
    if (visible) {
      dismissButtonRef.current?.focus()
    }
  }, [visible])

  if (!visible || !progress) return null

  const ctaHref = progress.nextItem?.route ?? introCourseFallbackHref(progress.courseCode)

  const handleDismiss = () => {
    recordIntroCourseBannerDismiss()
    void dismissIntroWelcomeBanner()
      .then(() => refresh())
      .catch(() => refresh())
  }

  return (
    <div
      role="region"
      aria-label={t('introCourse.banner.ariaLabel')}
      className="rounded-2xl border border-amber-200 bg-gradient-to-r from-amber-50 to-white p-4 shadow-sm dark:border-amber-900/50 dark:from-amber-950/40 dark:to-neutral-900"
    >
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0 flex-1">
          <p className="text-sm font-semibold text-slate-900 dark:text-neutral-50">
            {t('introCourse.banner.title')}
          </p>
          <p className="mt-1 text-sm text-slate-600 dark:text-neutral-300">
            {t('introCourse.banner.body')}
          </p>
          <Link
            to={ctaHref}
            onClick={() => recordIntroCourseCtaClick()}
            className="mt-3 inline-flex items-center gap-2 rounded-xl bg-amber-700 px-4 py-2 text-sm font-semibold text-white shadow-sm transition-[background-color,color,border-color] hover:bg-amber-600"
          >
            {t('introCourse.banner.cta')}
            <ArrowRight className="h-4 w-4" aria-hidden />
          </Link>
        </div>
        <button
          ref={dismissButtonRef}
          type="button"
          onClick={handleDismiss}
          aria-label={t('introCourse.banner.dismiss')}
          className="inline-flex shrink-0 items-center justify-center rounded-lg p-2 text-slate-500 transition-[background-color,color,border-color] hover:bg-slate-100 hover:text-slate-700 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-amber-600 dark:text-neutral-400 dark:hover:bg-neutral-800 dark:hover:text-neutral-100"
        >
          <X className="h-4 w-4" aria-hidden />
        </button>
      </div>
    </div>
  )
}