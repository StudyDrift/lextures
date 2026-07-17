import { useCallback, useEffect, useId, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  fetchAdminBoardPolicies,
  fetchAdminBoardsOverview,
  patchAdminBoardPolicies,
  type BoardAdminOverview,
  type BoardAttribution,
  type BoardOrgPolicies,
} from '../../lib/boards-api'

function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  if (n < 1024 * 1024 * 1024) return `${(n / (1024 * 1024)).toFixed(1)} MB`
  return `${(n / (1024 * 1024 * 1024)).toFixed(2)} GB`
}

export default function BoardsGovernancePage() {
  const { t } = useTranslation('common')
  const titleId = useId()
  const [searchParams] = useSearchParams()
  const orgId = searchParams.get('orgId') ?? undefined
  const { ffVisualBoards, loading: featuresLoading } = usePlatformFeatures()

  const [policies, setPolicies] = useState<BoardOrgPolicies | null>(null)
  const [overview, setOverview] = useState<BoardAdminOverview | null>(null)
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [saved, setSaved] = useState(false)
  const [capDraft, setCapDraft] = useState('')

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const [pol, ov] = await Promise.all([
        fetchAdminBoardPolicies(orgId),
        fetchAdminBoardsOverview(orgId),
      ])
      setPolicies(pol)
      setOverview(ov)
      setCapDraft(pol.boardCapPerCourse != null ? String(pol.boardCapPerCourse) : '')
    } catch (e) {
      setError(e instanceof Error ? e.message : t('boards.admin.loadError'))
    } finally {
      setLoading(false)
    }
  }, [orgId, t])

  useEffect(() => {
    if (featuresLoading || !ffVisualBoards) return
    void load()
  }, [featuresLoading, ffVisualBoards, load])

  async function persist(patch: Parameters<typeof patchAdminBoardPolicies>[0]) {
    setSaving(true)
    setError(null)
    setSaved(false)
    try {
      const next = await patchAdminBoardPolicies(patch, orgId)
      setPolicies(next)
      setCapDraft(next.boardCapPerCourse != null ? String(next.boardCapPerCourse) : '')
      setSaved(true)
    } catch (e) {
      setError(e instanceof Error ? e.message : t('boards.admin.saveError'))
    } finally {
      setSaving(false)
    }
  }

  if (featuresLoading) {
    return (
      <main className="mx-auto max-w-3xl p-6">
        <p className="text-sm" role="status">
          {t('common.loading')}
        </p>
      </main>
    )
  }

  if (!ffVisualBoards) {
    return (
      <main className="mx-auto max-w-3xl p-6">
        <p className="text-sm text-slate-600 dark:text-neutral-400">{t('boards.admin.flagOff')}</p>
      </main>
    )
  }

  return (
    <main className="mx-auto max-w-3xl p-6">
      <h1 id={titleId} className="text-xl font-bold text-slate-900 dark:text-neutral-100">
        {t('boards.admin.title')}
      </h1>
      <p className="mt-1 text-sm text-slate-600 dark:text-neutral-400">{t('boards.admin.subtitle')}</p>

      {error ? (
        <p className="mt-4 text-sm text-red-600 dark:text-red-400" role="alert">
          {error}
        </p>
      ) : null}
      {saved ? (
        <p className="mt-4 text-sm text-green-700 dark:text-green-400" role="status">
          {t('boards.admin.saved')}
        </p>
      ) : null}

      {loading || !policies || !overview ? (
        <p className="mt-6 text-sm" role="status">
          {t('common.loading')}
        </p>
      ) : (
        <div className="mt-6 space-y-8">
          <section aria-labelledby={`${titleId}-overview`}>
            <h2 id={`${titleId}-overview`} className="text-base font-semibold text-slate-900 dark:text-neutral-100">
              {t('boards.admin.overviewTitle')}
            </h2>
            <dl className="mt-3 grid grid-cols-2 gap-3 sm:grid-cols-3">
              <div className="rounded-md border border-slate-200 p-3 dark:border-neutral-700">
                <dt className="text-xs text-slate-500">{t('boards.admin.boardCount')}</dt>
                <dd className="text-lg font-semibold">{overview.boardCount}</dd>
              </div>
              <div className="rounded-md border border-slate-200 p-3 dark:border-neutral-700">
                <dt className="text-xs text-slate-500">{t('boards.admin.activeBoards')}</dt>
                <dd className="text-lg font-semibold">{overview.activeBoardCount}</dd>
              </div>
              <div className="rounded-md border border-slate-200 p-3 dark:border-neutral-700">
                <dt className="text-xs text-slate-500">{t('boards.admin.coursesEnabled')}</dt>
                <dd className="text-lg font-semibold">{overview.coursesFeatureEnabled}</dd>
              </div>
              <div className="rounded-md border border-slate-200 p-3 dark:border-neutral-700">
                <dt className="text-xs text-slate-500">{t('boards.admin.coursesWithBoards')}</dt>
                <dd className="text-lg font-semibold">{overview.coursesWithBoards}</dd>
              </div>
              <div className="rounded-md border border-slate-200 p-3 dark:border-neutral-700 sm:col-span-2">
                <dt className="text-xs text-slate-500">{t('boards.admin.storage')}</dt>
                <dd className="text-lg font-semibold">{formatBytes(overview.storageBytes)}</dd>
              </div>
            </dl>
            <table className="mt-4 w-full text-start text-sm">
              <caption className="mb-2 text-start text-xs text-slate-500">
                {t('boards.admin.topContentTypes')}
              </caption>
              <thead>
                <tr className="border-b border-slate-200 text-slate-500 dark:border-neutral-700">
                  <th scope="col" className="py-1 font-medium">
                    {t('boards.admin.contentType')}
                  </th>
                  <th scope="col" className="py-1 font-medium">
                    {t('boards.admin.count')}
                  </th>
                </tr>
              </thead>
              <tbody>
                {overview.topContentTypes.length === 0 ? (
                  <tr>
                    <td colSpan={2} className="py-2 text-slate-500">
                      {t('boards.admin.noContentTypes')}
                    </td>
                  </tr>
                ) : (
                  overview.topContentTypes.map((row) => (
                    <tr key={row.contentType} className="border-b border-slate-100 dark:border-neutral-800">
                      <td className="py-1">{row.contentType}</td>
                      <td className="py-1">{row.count}</td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </section>

          <section aria-labelledby={`${titleId}-policies`}>
            <h2 id={`${titleId}-policies`} className="text-base font-semibold text-slate-900 dark:text-neutral-100">
              {t('boards.admin.policiesTitle')}
            </h2>
            <div className="mt-3 space-y-4">
              <label className="flex items-start gap-3 text-sm">
                <input
                  type="checkbox"
                  className="mt-1"
                  checked={policies.externalSharing}
                  disabled={saving}
                  onChange={(e) => void persist({ externalSharing: e.target.checked })}
                />
                <span>
                  <span className="font-medium text-slate-900 dark:text-neutral-100">
                    {t('boards.admin.externalSharing')}
                  </span>
                  <span className="mt-0.5 block text-slate-600 dark:text-neutral-400">
                    {t('boards.admin.externalSharingHint')}
                  </span>
                </span>
              </label>
              <label className="flex items-start gap-3 text-sm">
                <input
                  type="checkbox"
                  className="mt-1"
                  checked={policies.minorModerationFloor}
                  disabled={saving}
                  onChange={(e) => void persist({ minorModerationFloor: e.target.checked })}
                />
                <span>
                  <span className="font-medium text-slate-900 dark:text-neutral-100">
                    {t('boards.admin.minorFloor')}
                  </span>
                  <span className="mt-0.5 block text-slate-600 dark:text-neutral-400">
                    {t('boards.admin.minorFloorHint')}
                  </span>
                </span>
              </label>
              <label className="block text-sm">
                <span className="font-medium text-slate-900 dark:text-neutral-100">
                  {t('boards.admin.defaultAttribution')}
                </span>
                <select
                  className="mt-1 block w-full max-w-xs rounded-md border border-slate-300 bg-white px-3 py-2 dark:border-neutral-600 dark:bg-neutral-900"
                  value={policies.defaultAttribution}
                  disabled={saving}
                  onChange={(e) =>
                    void persist({ defaultAttribution: e.target.value as BoardAttribution })
                  }
                >
                  <option value="named">{t('boards.access.attribution.named')}</option>
                  <option value="anon_to_peers">{t('boards.access.attribution.anon_to_peers')}</option>
                  <option value="anonymous">{t('boards.access.attribution.anonymous')}</option>
                </select>
              </label>
              <div className="text-sm">
                <label htmlFor={`${titleId}-cap`} className="font-medium text-slate-900 dark:text-neutral-100">
                  {t('boards.admin.boardCap')}
                </label>
                <p className="mt-0.5 text-slate-600 dark:text-neutral-400">{t('boards.admin.boardCapHint')}</p>
                <div className="mt-2 flex flex-wrap items-center gap-2">
                  <input
                    id={`${titleId}-cap`}
                    type="number"
                    min={0}
                    className="w-32 rounded-md border border-slate-300 px-3 py-2 dark:border-neutral-600 dark:bg-neutral-900"
                    value={capDraft}
                    disabled={saving}
                    placeholder={t('boards.admin.unlimited')}
                    onChange={(e) => setCapDraft(e.target.value)}
                  />
                  <button
                    type="button"
                    className="rounded-md bg-slate-900 px-3 py-2 text-white disabled:opacity-50 dark:bg-neutral-100 dark:text-neutral-900"
                    disabled={saving}
                    onClick={() => {
                      const trimmed = capDraft.trim()
                      if (trimmed === '') {
                        void persist({ clearBoardCap: true })
                        return
                      }
                      const n = Number(trimmed)
                      if (!Number.isFinite(n) || n < 0) {
                        setError(t('boards.admin.capInvalid'))
                        return
                      }
                      void persist({ boardCapPerCourse: Math.floor(n) })
                    }}
                  >
                    {t('boards.admin.saveCap')}
                  </button>
                </div>
              </div>
            </div>
          </section>
        </div>
      )}
    </main>
  )
}
