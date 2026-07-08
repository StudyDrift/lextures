import { useEffect } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { ArrowRight, BookOpen, CheckCircle2, Sparkles } from 'lucide-react'
import { useIntroCourseProgress } from '../../hooks/use-intro-course-progress'
import {
  introCourseCardState,
  introCourseFallbackHref,
  type IntroCourseProgress,
} from '../../lib/intro-course-api'
import { recordIntroCourseCardView, recordIntroCourseCtaClick } from '../../lib/intro-course-observability'
import { IntroCourseProgressBar } from './intro-course-progress-bar'

function IntroCourseCardSkeleton() {
  return (
    <section
      aria-busy="true"
      aria-label="Loading intro course"
      className="rounded-2xl border border-sky-100 bg-gradient-to-br from-sky-50/90 to-white p-5 shadow-sm dark:border-sky-900/40 dark:from-sky-950/30 dark:to-neutral-900"
    >
      <div className="h-4 w-24 animate-pulse rounded bg-sky-100 dark:bg-sky-900/40" />
      <div className="mt-3 h-6 w-2/3 max-w-sm animate-pulse rounded bg-slate-100 dark:bg-neutral-800" />
      <div className="mt-4 h-2 w-full animate-pulse rounded-full bg-slate-100 dark:bg-neutral-800" />
      <div className="mt-4 h-10 w-36 animate-pulse rounded-xl bg-sky-100 dark:bg-sky-900/40" />
    </section>
  )
}

function IntroCourseCardContent({ progress }: { progress: IntroCourseProgress }) {
  const { t } = useTranslation('introCourse')
  const state = introCourseCardState(progress, false, false)
  const courseCode = progress.courseCode ?? introCourseFallbackHref().split('/').pop() ?? 'C-WLCOME'
  const ctaHref = progress.nextItem?.route ?? introCourseFallbackHref(courseCode)

  if (state === 'completed') {
    return (
      <section aria-label={t('introCourse.card.completedAria')}>
        <article className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-slate-200 bg-slate-50/80 px-4 py-3 dark:border-neutral-700 dark:bg-neutral-900/50">
          <div className="flex min-w-0 items-center gap-2 text-sm text-slate-700 dark:text-neutral-200">
            <CheckCircle2 className="h-4 w-4 shrink-0 text-emerald-600 dark:text-emerald-400" aria-hidden />
            <span>{t('introCourse.card.completedLabel')}</span>
          </div>
          <Link
            to={introCourseFallbackHref(courseCode)}
            onClick={() => recordIntroCourseCtaClick()}
            className="inline-flex items-center gap-1 text-sm font-medium text-sky-700 hover:underline dark:text-sky-300"
          >
            {t('introCourse.card.revisit')}
            <ArrowRight className="h-3.5 w-3.5" aria-hidden />
          </Link>
        </article>
      </section>
    )
  }

  const isNotStarted = state === 'not-started'

  return (
    <section aria-label={t('introCourse.card.ariaLabel')}>
      <article className="rounded-2xl border border-sky-100 bg-gradient-to-br from-sky-50/90 to-white p-5 shadow-sm dark:border-sky-900/40 dark:from-sky-950/30 dark:to-neutral-900">
        <div className="flex flex-wrap items-center gap-2 text-xs font-medium text-sky-800 dark:text-sky-200">
          <Sparkles className="h-4 w-4 shrink-0" aria-hidden />
          <span>{isNotStarted ? t('introCourse.card.startHere') : t('introCourse.card.continueOnboarding')}</span>
        </div>
        <h2 className="mt-2 text-lg font-semibold tracking-tight text-slate-900 dark:text-neutral-50">
          {t('introCourse.card.title')}
        </h2>
        {progress.nextItem?.title && !isNotStarted ? (
          <p className="mt-1 text-sm text-slate-600 dark:text-neutral-400">
            {t('introCourse.card.nextUp', { title: progress.nextItem.title })}
          </p>
        ) : (
          <p className="mt-1 text-sm text-slate-600 dark:text-neutral-400">
            {t('introCourse.card.subtitle')}
          </p>
        )}
        <div className="mt-4">
          <IntroCourseProgressBar
            percent={progress.percent}
            modulesComplete={progress.modulesComplete}
            modulesTotal={progress.modulesTotal}
          />
        </div>
        <Link
          to={ctaHref}
          onClick={() => recordIntroCourseCtaClick()}
          className="mt-4 inline-flex items-center gap-2 rounded-xl bg-sky-600 px-4 py-2.5 text-sm font-semibold text-white shadow-sm transition-[background-color,color,border-color] hover:bg-sky-500"
        >
          {isNotStarted ? t('introCourse.card.ctaStart') : t('introCourse.card.ctaContinue')}
          <ArrowRight className="h-4 w-4" aria-hidden />
        </Link>
      </article>
    </section>
  )
}

function IntroCourseCardError() {
  const { t } = useTranslation('introCourse')
  return (
    <section aria-label={t('introCourse.card.ariaLabel')}>
      <article className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-neutral-700 dark:bg-neutral-900">
        <div className="flex flex-wrap items-center gap-2 text-xs font-medium text-slate-600 dark:text-neutral-300">
          <BookOpen className="h-4 w-4 shrink-0" aria-hidden />
          <span>{t('introCourse.card.fallbackLabel')}</span>
        </div>
        <Link
          to={introCourseFallbackHref()}
          onClick={() => recordIntroCourseCtaClick()}
          className="mt-3 inline-flex items-center gap-2 text-sm font-semibold text-sky-700 hover:underline dark:text-sky-300"
        >
          {t('introCourse.card.fallbackLink')}
          <ArrowRight className="h-4 w-4" aria-hidden />
        </Link>
      </article>
    </section>
  )
}

export function IntroCourseCard() {
  const { progress, loading, error } = useIntroCourseProgress()
  const state = introCourseCardState(progress, loading, error)

  useEffect(() => {
    if (state !== 'hidden' && state !== 'loading') {
      recordIntroCourseCardView()
    }
  }, [state])

  if (state === 'hidden') return null
  if (state === 'loading') return <IntroCourseCardSkeleton />
  if (state === 'error') return <IntroCourseCardError />
  if (!progress) return null
  return <IntroCourseCardContent progress={progress} />
}