import { useEffect, useMemo, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { Award, PartyPopper } from 'lucide-react'
import { Link } from 'react-router-dom'
import { CredentialShareActions } from '../credentials/credential-share-actions'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { useIntroCourseProgress } from '../../hooks/use-intro-course-progress'
import {
  markIntroCelebrationSeen,
  shouldShowIntroCelebration,
} from '../../lib/intro-course-api'
import type { IssuedCredentialSummary } from '../../lib/credentials-api'
import { recordIntroCourseCelebrationView } from '../../lib/intro-course-observability'

function prefersReducedMotion(): boolean {
  if (typeof window === 'undefined') return false
  return (
    document.documentElement.classList.contains('reduced-motion') ||
    window.matchMedia('(prefers-reduced-motion: reduce)').matches
  )
}

export function IntroCompletionCelebration() {
  const { t } = useTranslation('introCourse')
  const { ffCompletionCredentials, ffHighContrastReducedMotion } = usePlatformFeatures()
  const { progress, loading, refresh } = useIntroCourseProgress()
  const closeButtonRef = useRef<HTMLButtonElement>(null)

  const visible = !loading && shouldShowIntroCelebration(progress)
  const reducedMotion = ffHighContrastReducedMotion || prefersReducedMotion()

  useEffect(() => {
    if (visible) {
      recordIntroCourseCelebrationView()
      closeButtonRef.current?.focus()
    }
  }, [visible])

  const credential = useMemo((): IssuedCredentialSummary | null => {
    if (!progress?.credentialId || !ffCompletionCredentials) return null
    const origin = typeof window !== 'undefined' ? window.location.origin : ''
    return {
      id: progress.credentialId,
      title: t('introCourse.celebration.credentialTitle'),
      sourceType: 'course',
      sourceId: progress.courseCode ?? 'C-WLCOME',
      issuedAt: progress.completedAt ?? new Date().toISOString(),
      verificationUrl: `${origin}/verify/${progress.credentialId}`,
      revoked: false,
    }
  }, [progress, ffCompletionCredentials, t])

  if (!visible) return null

  const handleClose = () => {
    void markIntroCelebrationSeen()
      .then(() => refresh())
      .catch(() => refresh())
  }

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-label={t('introCourse.celebration.ariaLabel')}
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/70 p-4"
    >
      {!reducedMotion ? (
        <div
          aria-hidden
          className="pointer-events-none absolute inset-0 overflow-hidden motion-safe:animate-pulse"
        >
          <div className="absolute left-1/4 top-1/4 h-2 w-2 rounded-full bg-amber-300 opacity-80" />
          <div className="absolute right-1/3 top-1/3 h-2 w-2 rounded-full bg-sky-300 opacity-80" />
          <div className="absolute bottom-1/4 left-1/3 h-2 w-2 rounded-full bg-emerald-300 opacity-80" />
        </div>
      ) : null}
      <div className="relative w-full max-w-md rounded-2xl bg-white p-8 text-center shadow-xl dark:bg-neutral-900">
        <PartyPopper className="mx-auto h-12 w-12 text-emerald-500" aria-hidden />
        <h2 className="mt-4 text-xl font-bold text-slate-900 dark:text-white">
          {t('introCourse.celebration.title')}
        </h2>
        <p className="mt-2 text-sm text-slate-600 dark:text-neutral-300">
          {credential
            ? t('introCourse.celebration.bodyWithCredential')
            : t('introCourse.celebration.body')}
        </p>
        {credential ? (
          <div className="mt-6 text-left">
            <div className="mb-3 flex items-center gap-2 text-sm font-medium text-slate-800 dark:text-neutral-100">
              <Award className="h-4 w-4 text-emerald-600" aria-hidden />
              <span>{t('introCourse.celebration.badgeLabel')}</span>
            </div>
            <CredentialShareActions credential={credential} layout="stack" />
          </div>
        ) : (
          <Link
            to="/me/credentials"
            className="mt-4 inline-flex text-sm font-medium text-sky-700 hover:underline dark:text-sky-300"
          >
            {t('introCourse.celebration.credentialsLink')}
          </Link>
        )}
        <button
          ref={closeButtonRef}
          type="button"
          onClick={handleClose}
          className="mt-6 w-full rounded-lg px-4 py-2 text-sm font-medium text-slate-600 hover:bg-slate-100 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-sky-600 dark:text-neutral-300 dark:hover:bg-neutral-800"
        >
          {t('introCourse.celebration.close')}
        </button>
      </div>
    </div>
  )
}