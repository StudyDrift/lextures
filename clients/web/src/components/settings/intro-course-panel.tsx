import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { Loader2, RefreshCw, Users } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { formatDateTime } from '../../lib/format'
import { toastMutationError, toastSaveOk } from '../../lib/lms-toast'
import {
  fetchIntroCourseAdminAnalytics,
  fetchIntroCourseAdminStatus,
  resyncIntroCourse,
  startIntroCourseBackfill,
  type IntroCourseAdminAnalytics,
  type IntroCourseAdminStatus,
} from '../../lib/intro-course-admin-api'
import { useConfirm } from '../use-confirm'
import { SettingsSection } from './settings-section'
import { FeatureToggleRow } from './feature-toggle-row'
import { PLATFORM_FEATURE_DEFINITIONS } from './platform-feature-definitions'
import { authorizedFetch } from '../../lib/api'
import { readApiErrorMessage } from '../../lib/errors'

function percentLabel(rate: number): string {
  return `${Math.round(rate * 100)}%`
}

function FunnelTable({ analytics }: { analytics: IntroCourseAdminAnalytics }) {
  const { t } = useTranslation('introCourse')
  if (analytics.perModuleFunnel.length === 0) {
    return (
      <p className="text-sm text-slate-500 dark:text-neutral-400">
        {t('introCourse.admin.analytics.empty')}
      </p>
    )
  }
  return (
    <div className="overflow-x-auto">
      <table className="min-w-full text-start text-sm">
        <caption className="sr-only">{t('introCourse.admin.analytics.funnelCaption')}</caption>
        <thead>
          <tr className="border-b border-slate-200 text-slate-500 dark:border-neutral-700 dark:text-neutral-400">
            <th scope="col" className="py-2 pr-4 font-medium">
              {t('introCourse.admin.analytics.module')}
            </th>
            <th scope="col" className="py-2 pr-4 font-medium">
              {t('introCourse.admin.analytics.attempted')}
            </th>
            <th scope="col" className="py-2 font-medium">
              {t('introCourse.admin.analytics.rate')}
            </th>
          </tr>
        </thead>
        <tbody>
          {analytics.perModuleFunnel.map((row) => (
            <tr
              key={row.moduleSlug}
              className="border-b border-slate-100 dark:border-neutral-800"
            >
              <td className="py-2 pr-4 text-slate-900 dark:text-neutral-100">{row.moduleTitle}</td>
              <td className="py-2 pr-4 tabular-nums text-slate-700 dark:text-neutral-300">
                {row.quizAttempted}
              </td>
              <td className="py-2 tabular-nums text-slate-700 dark:text-neutral-300">
                {percentLabel(row.attemptRate)}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

export function IntroCoursePanel() {
  const { t } = useTranslation('introCourse')
  const { introCourseEnabled, loading: featuresLoading, refresh: refreshFeatures } = usePlatformFeatures()
  const { confirm, ConfirmDialogHost, setConfirmBusy } = useConfirm()
  const [status, setStatus] = useState<IntroCourseAdminStatus | null>(null)
  const [analytics, setAnalytics] = useState<IntroCourseAdminAnalytics | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [flagBusy, setFlagBusy] = useState(false)

  const introFeature = useMemo(
    () => PLATFORM_FEATURE_DEFINITIONS.find((f) => f.key === 'introCourseEnabled'),
    [],
  )

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const [st, an] = await Promise.all([
        fetchIntroCourseAdminStatus(),
        fetchIntroCourseAdminAnalytics(),
      ])
      setStatus(st)
      setAnalytics(an)
    } catch {
      setError(t('introCourse.admin.error.load'))
    } finally {
      setLoading(false)
    }
  }, [t])

  useEffect(() => {
    void load()
  }, [load])

  async function toggleFlag(enabled: boolean) {
    setFlagBusy(true)
    try {
      const res = await authorizedFetch('/api/v1/settings/platform', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ introCourseEnabled: enabled, updateMask: ['introCourseEnabled'] }),
      })
      const raw: unknown = await res.json().catch(() => ({}))
      if (!res.ok) {
        throw new Error(readApiErrorMessage(raw))
      }
      await refreshFeatures()
      await load()
      toastSaveOk()
    } catch (e) {
      toastMutationError(e instanceof Error ? e.message : t('introCourse.admin.error.load'))
    } finally {
      setFlagBusy(false)
    }
  }

  async function runResync() {
    const ok = await confirm({
      title: t('introCourse.admin.resyncConfirm.title'),
      description: t('introCourse.admin.resyncConfirm.body'),
      confirmLabel: t('introCourse.admin.actions.resync'),
    })
    if (!ok) return
    setBusy(true)
    setConfirmBusy(true)
    try {
      await resyncIntroCourse()
      await load()
      toastSaveOk(t('introCourse.admin.toast.resync'))
    } catch (e) {
      toastMutationError(e instanceof Error ? e.message : t('introCourse.admin.error.load'))
    } finally {
      setBusy(false)
      setConfirmBusy(false)
    }
  }

  async function runBackfill() {
    const ok = await confirm({
      title: t('introCourse.admin.backfillConfirm.title'),
      description: t('introCourse.admin.backfillConfirm.body'),
      confirmLabel: t('introCourse.admin.actions.backfill'),
    })
    if (!ok) return
    setBusy(true)
    setConfirmBusy(true)
    try {
      await startIntroCourseBackfill()
      await load()
      toastSaveOk(t('introCourse.admin.toast.backfill'))
    } catch (e) {
      toastMutationError(e instanceof Error ? e.message : t('introCourse.admin.error.load'))
    } finally {
      setBusy(false)
      setConfirmBusy(false)
    }
  }

  if (loading || featuresLoading) {
    return (
      <p className="mt-6 flex items-center gap-2 text-sm text-slate-500 dark:text-neutral-400">
        <Loader2 className="h-4 w-4 motion-safe:animate-spin" aria-hidden />
        {t('introCourse.admin.loading')}
      </p>
    )
  }

  if (error) {
    return (
      <p role="alert" className="mt-6 text-sm text-red-600 dark:text-red-400">
        {error}
      </p>
    )
  }

  const dropOffTitle =
    analytics?.perModuleFunnel.find((m) => m.moduleSlug === analytics.dropOffModuleSlug)?.moduleTitle ??
    analytics?.dropOffModuleSlug

  return (
    <div className="mt-6 space-y-6">
      {ConfirmDialogHost}

      <p className="text-sm text-slate-500 dark:text-neutral-400">{t('introCourse.admin.auditNote')}</p>

      {introFeature ? (
        <SettingsSection
          id="intro-course-flag"
          title={t('introCourse.admin.flag.title')}
          description={t('introCourse.admin.flag.description')}
        >
          <FeatureToggleRow
            label={introFeature.label}
            description={introFeature.description}
            enabled={introCourseEnabled ?? true}
            disabled={flagBusy}
            onToggle={() => void toggleFlag(!(introCourseEnabled ?? true))}
          />
        </SettingsSection>
      ) : null}

      <SettingsSection
        id="intro-course-status"
        title={t('introCourse.admin.status.title')}
        description={t('introCourse.admin.status.description')}
      >
        <dl className="grid gap-3 text-sm sm:grid-cols-2">
          <div>
            <dt className="text-slate-500 dark:text-neutral-500">{t('introCourse.admin.status.version')}</dt>
            <dd className="font-mono text-slate-900 dark:text-neutral-100">{status?.contentVersion ?? '—'}</dd>
          </div>
          <div>
            <dt className="text-slate-500 dark:text-neutral-500">{t('introCourse.admin.status.modules')}</dt>
            <dd className="text-slate-900 dark:text-neutral-100">{status?.moduleCount ?? 0}</dd>
          </div>
          <div>
            <dt className="text-slate-500 dark:text-neutral-500">{t('introCourse.admin.status.lastSync')}</dt>
            <dd className="text-slate-900 dark:text-neutral-100">
              {status?.lastSyncedAt ? formatDateTime(status.lastSyncedAt) : t('introCourse.admin.status.never')}
              {status?.lastSyncResult ? ` (${status.lastSyncResult})` : ''}
            </dd>
          </div>
          <div>
            <dt className="text-slate-500 dark:text-neutral-500">{t('introCourse.admin.status.validation')}</dt>
            <dd className="text-slate-900 dark:text-neutral-100">
              {status?.lastValidationResult ?? t('introCourse.admin.status.unknown')}
            </dd>
          </div>
          <div className="sm:col-span-2">
            <dt className="text-slate-500 dark:text-neutral-500">{t('introCourse.admin.status.locales')}</dt>
            <dd className="text-slate-900 dark:text-neutral-100">
              {(status?.availableLocales ?? []).map((loc) => {
                const cov = status?.localeCoverage?.[loc]
                const pct = cov != null ? ` ${Math.round(cov * 100)}%` : ''
                return `${loc}${pct}`
              }).join(' · ') || 'en'}
            </dd>
          </div>
        </dl>

        {status?.coursePresent && status.courseId ? (
          <p className="mt-4 text-sm">
            <Link
              to={`/courses/${encodeURIComponent(status.courseCode)}`}
              className="font-medium text-indigo-600 hover:text-indigo-800 dark:text-indigo-400 dark:hover:text-indigo-300"
            >
              {t('introCourse.admin.status.openCourse')}
            </Link>
          </p>
        ) : null}

        <div className="mt-4 flex flex-wrap gap-3">
          <button
            type="button"
            disabled={busy}
            onClick={() => void runResync()}
            className="inline-flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-60 dark:bg-indigo-500 dark:hover:bg-indigo-600"
          >
            <RefreshCw className="h-4 w-4" aria-hidden />
            {t('introCourse.admin.actions.resync')}
          </button>
          <button
            type="button"
            disabled={busy || !introCourseEnabled}
            onClick={() => void runBackfill()}
            className="inline-flex items-center gap-2 rounded-lg border border-slate-200 bg-white px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 disabled:opacity-60 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-200 dark:hover:bg-neutral-800"
          >
            <Users className="h-4 w-4" aria-hidden />
            {t('introCourse.admin.actions.backfill')}
          </button>
        </div>

        {status?.backfill ? (
          <p className="mt-3 text-sm text-slate-600 dark:text-neutral-400">
            {t('introCourse.admin.backfill.remaining', { count: status.backfill.remaining })}
            {status.backfill.completedAt
              ? ` · ${t('introCourse.admin.backfill.completed')}`
              : status.backfill.startedAt
                ? ` · ${t('introCourse.admin.backfill.inProgress')}`
                : ''}
          </p>
        ) : null}
      </SettingsSection>

      <SettingsSection
        id="intro-course-analytics"
        title={t('introCourse.admin.analytics.title')}
        description={t('introCourse.admin.analytics.description')}
      >
        {analytics ? (
          <div className="space-y-4">
            <dl className="grid gap-3 text-sm sm:grid-cols-3">
              <div>
                <dt className="text-slate-500 dark:text-neutral-500">{t('introCourse.admin.analytics.enrolled')}</dt>
                <dd className="text-lg font-semibold tabular-nums text-slate-900 dark:text-neutral-100">
                  {analytics.enrolled}
                </dd>
              </div>
              <div>
                <dt className="text-slate-500 dark:text-neutral-500">{t('introCourse.admin.analytics.completed')}</dt>
                <dd className="text-lg font-semibold tabular-nums text-slate-900 dark:text-neutral-100">
                  {analytics.completed}
                </dd>
              </div>
              <div>
                <dt className="text-slate-500 dark:text-neutral-500">{t('introCourse.admin.analytics.rate')}</dt>
                <dd className="text-lg font-semibold tabular-nums text-slate-900 dark:text-neutral-100">
                  {percentLabel(analytics.completionRate)}
                </dd>
              </div>
            </dl>
            {analytics.avgTimeToCompleteHours != null ? (
              <p className="text-sm text-slate-600 dark:text-neutral-400">
                {t('introCourse.admin.analytics.avgHours', {
                  hours: analytics.avgTimeToCompleteHours.toFixed(1),
                })}
              </p>
            ) : null}
            {dropOffTitle ? (
              <p className="text-sm text-slate-600 dark:text-neutral-400">
                {t('introCourse.admin.analytics.dropOff', { module: dropOffTitle })}
              </p>
            ) : null}
            <FunnelTable analytics={analytics} />
          </div>
        ) : null}
      </SettingsSection>
    </div>
  )
}