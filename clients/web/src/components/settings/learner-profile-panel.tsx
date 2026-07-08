import { useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { Download, Pause, Play, RotateCcw } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  downloadLearnerProfileExport,
  fetchLearnerProfile,
  pauseLearnerProfile,
  resetLearnerProfile,
  resumeLearnerProfile,
  sortFacetsByPriority,
  type LearnerProfile,
} from '../../lib/learner-profile-api'
import { recordLearnerProfilePageView } from '../../lib/learner-profile-observability'
import { useConfirm } from '../use-confirm'
import { SettingsSection } from './settings-section'
import { FacetSection } from './learner-profile/facet-section'

function ProfileSkeleton() {
  return (
    <div className="space-y-4" aria-hidden>
      {[0, 1, 2].map((i) => (
        <div
          key={i}
          className="h-28 motion-safe:animate-pulse rounded-2xl border border-slate-200 bg-slate-100 dark:border-neutral-700 dark:bg-neutral-800"
        />
      ))}
    </div>
  )
}

export function LearnerProfilePanel() {
  const { t } = useTranslation('learnerProfile')
  const { gdprModuleEnabled } = usePlatformFeatures()
  const { confirm, ConfirmDialogHost, setConfirmBusy } = useConfirm()
  const [profile, setProfile] = useState<LearnerProfile | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [controlBusy, setControlBusy] = useState(false)
  const [controlError, setControlError] = useState<string | null>(null)

  const loadProfile = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const data = await fetchLearnerProfile()
      setProfile(data)
    } catch {
      setError(t('learnerProfile.error.load'))
    } finally {
      setLoading(false)
    }
  }, [t])

  useEffect(() => {
    recordLearnerProfilePageView()
    void loadProfile()
  }, [loadProfile])

  const facets = profile ? sortFacetsByPriority(profile.facets) : []
  const showEmpty =
    profile &&
    (profile.status === 'insufficient_data' || facets.every((f) => f.state === 'insufficient_data'))
  const isPaused = profile?.status === 'paused'

  async function runControl(action: () => Promise<void>) {
    setControlError(null)
    setControlBusy(true)
    setConfirmBusy(true)
    try {
      await action()
      await loadProfile()
    } catch {
      setControlError(t('learnerProfile.manage.error'))
    } finally {
      setControlBusy(false)
      setConfirmBusy(false)
    }
  }

  async function handleDownload() {
    await runControl(async () => {
      await downloadLearnerProfileExport()
    })
  }

  async function handlePauseOrResume() {
    if (isPaused) {
      const ok = await confirm({
        title: t('learnerProfile.manage.resumeConfirm.title'),
        description: t('learnerProfile.manage.resumeConfirm.body'),
        confirmLabel: t('learnerProfile.manage.resume'),
      })
      if (!ok) return
      await runControl(async () => {
        await resumeLearnerProfile()
      })
      return
    }
    const ok = await confirm({
      title: t('learnerProfile.manage.pauseConfirm.title'),
      description: t('learnerProfile.manage.pauseConfirm.body'),
      confirmLabel: t('learnerProfile.manage.pause'),
    })
    if (!ok) return
    await runControl(async () => {
      await pauseLearnerProfile()
    })
  }

  async function handleReset() {
    const ok = await confirm({
      title: t('learnerProfile.manage.resetConfirm.title'),
      description: t('learnerProfile.manage.resetConfirm.body'),
      confirmLabel: t('learnerProfile.manage.reset'),
      variant: 'danger',
      requireTypedPhrase: t('learnerProfile.manage.resetConfirm.phrase'),
    })
    if (!ok) return
    await runControl(async () => {
      await resetLearnerProfile()
    })
  }

  return (
    <div className="space-y-6">
      {ConfirmDialogHost}

      <div>
        <h2 className="text-base font-semibold text-slate-900 dark:text-neutral-100">
          {t('learnerProfile.title')}
        </h2>
        <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">
          {t('learnerProfile.description')}
        </p>
      </div>

      <SettingsSection
        id="learner-profile-how-it-works"
        title={t('learnerProfile.howItWorks.title')}
        description={t('learnerProfile.howItWorks.body')}
      >
        <p className="text-sm text-slate-600 dark:text-neutral-300">
          {t('learnerProfile.howItWorks.art22')}
        </p>
      </SettingsSection>

      {gdprModuleEnabled ? (
        <p className="text-sm text-slate-600 dark:text-neutral-300">
          {t('learnerProfile.privacy.notice')}{' '}
          <Link
            to="/privacy-centre"
            className="font-medium text-indigo-600 hover:text-indigo-500 dark:text-indigo-300"
          >
            {t('learnerProfile.privacy.link')}
          </Link>
        </p>
      ) : null}

      {loading ? <ProfileSkeleton /> : null}

      {error ? (
        <p
          className="rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-800 dark:border-rose-900/50 dark:bg-rose-950/40 dark:text-rose-200"
          role="alert"
        >
          {error}
        </p>
      ) : null}

      {controlError ? (
        <p
          className="rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-800 dark:border-rose-900/50 dark:bg-rose-950/40 dark:text-rose-200"
          role="alert"
        >
          {controlError}
        </p>
      ) : null}

      {!loading && !error && isPaused ? (
        <div
          className="rounded-2xl border border-amber-200 bg-amber-50 px-5 py-4 dark:border-amber-900/50 dark:bg-amber-950/30"
          role="status"
        >
          <h3 className="text-sm font-semibold text-amber-950 dark:text-amber-100">
            {t('learnerProfile.paused.title')}
          </h3>
          <p className="mt-1 text-sm text-amber-900 dark:text-amber-200">
            {t('learnerProfile.paused.body')}
          </p>
        </div>
      ) : null}

      {!loading && !error && showEmpty ? (
        <div
          className="rounded-2xl border border-slate-200 bg-slate-50 px-5 py-8 text-center dark:border-neutral-700 dark:bg-neutral-800/50"
          role="status"
        >
          <h3 className="text-base font-semibold text-slate-900 dark:text-neutral-100">
            {t('learnerProfile.empty.title')}
          </h3>
          <p className="mx-auto mt-2 max-w-md text-sm text-slate-600 dark:text-neutral-300">
            {t('learnerProfile.empty.body')}
          </p>
        </div>
      ) : null}

      {!loading && !error && facets.length > 0 ? (
        <div className="space-y-4">
          {facets.map((facet) => (
            <FacetSection key={facet.facetKey} facet={facet} />
          ))}
        </div>
      ) : null}

      <SettingsSection
        id="learner-profile-manage"
        title={t('learnerProfile.manage.title')}
        description={t('learnerProfile.manage.description')}
      >
        <div className="flex flex-wrap gap-3">
          <button
            type="button"
            disabled={loading || controlBusy}
            onClick={() => void handleDownload()}
            className="inline-flex items-center gap-2 rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 disabled:opacity-60 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-200 dark:hover:bg-neutral-700"
          >
            <Download className="h-4 w-4" aria-hidden />
            {controlBusy ? t('learnerProfile.manage.exporting') : t('learnerProfile.manage.download')}
          </button>
          <button
            type="button"
            disabled={loading || controlBusy}
            onClick={() => void handlePauseOrResume()}
            className="inline-flex items-center gap-2 rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 disabled:opacity-60 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-200 dark:hover:bg-neutral-700"
          >
            {isPaused ? (
              <Play className="h-4 w-4" aria-hidden />
            ) : (
              <Pause className="h-4 w-4" aria-hidden />
            )}
            {isPaused ? t('learnerProfile.manage.resume') : t('learnerProfile.manage.pause')}
          </button>
          <button
            type="button"
            disabled={loading || controlBusy}
            onClick={() => void handleReset()}
            className="inline-flex items-center gap-2 rounded-xl border border-rose-200 bg-white px-3 py-2 text-sm font-medium text-rose-800 hover:bg-rose-50 disabled:opacity-60 dark:border-rose-900/50 dark:bg-neutral-800 dark:text-rose-200 dark:hover:bg-rose-950/40"
          >
            <RotateCcw className="h-4 w-4" aria-hidden />
            {t('learnerProfile.manage.reset')}
          </button>
        </div>
      </SettingsSection>
    </div>
  )
}