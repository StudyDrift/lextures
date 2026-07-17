import { useCallback, useEffect, useId, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  fetchAdminIQAnalytics,
  fetchAdminIQLiveGames,
  fetchAdminIQReviewQueue,
  fetchAdminIQSettings,
  patchAdminIQSettings,
  postAdminIQBulkArchiveKits,
  postAdminIQForceEnd,
  postAdminIQReviewAction,
  type InteractiveQuizAnalytics,
  type InteractiveQuizLiveGame,
  type InteractiveQuizPlatformSettings,
  type InteractiveQuizReviewItem,
  type IQGuestJoinPolicy,
} from '../../lib/live-quiz-api'

function formatCents(cents: number): string {
  return `$${(cents / 100).toFixed(2)}`
}

export default function LiveQuizzesGovernancePage() {
  const { t } = useTranslation('common')
  const titleId = useId()
  const [searchParams] = useSearchParams()
  const orgId = searchParams.get('orgId') ?? undefined
  const [settings, setSettings] = useState<InteractiveQuizPlatformSettings | null>(null)
  const [analytics, setAnalytics] = useState<InteractiveQuizAnalytics | null>(null)
  const [queue, setQueue] = useState<InteractiveQuizReviewItem[]>([])
  const [pendingCount, setPendingCount] = useState(0)
  const [games, setGames] = useState<InteractiveQuizLiveGame[]>([])
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [saved, setSaved] = useState(false)
  const [rejectReasons, setRejectReasons] = useState<Record<string, string>>({})
  const [concurrentDraft, setConcurrentDraft] = useState('')
  const [kitsDraft, setKitsDraft] = useState('')

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const [s, a, q, g] = await Promise.all([
        fetchAdminIQSettings(),
        fetchAdminIQAnalytics(orgId),
        fetchAdminIQReviewQueue('pending'),
        fetchAdminIQLiveGames(orgId),
      ])
      setSettings(s)
      setAnalytics(a)
      setQueue(q.items)
      setPendingCount(q.pendingCount)
      setGames(g.games)
      setConcurrentDraft(s.maxConcurrentGames != null ? String(s.maxConcurrentGames) : '')
      setKitsDraft(s.maxKitsPerCourse != null ? String(s.maxKitsPerCourse) : '')
    } catch (e) {
      setError(e instanceof Error ? e.message : t('admin.liveQuiz.loadError'))
    } finally {
      setLoading(false)
    }
  }, [orgId, t])

  useEffect(() => {
    void load()
  }, [load])

  async function persist(patch: Parameters<typeof patchAdminIQSettings>[0]) {
    setSaving(true)
    setError(null)
    setSaved(false)
    try {
      const next = await patchAdminIQSettings(patch)
      setSettings(next)
      setConcurrentDraft(next.maxConcurrentGames != null ? String(next.maxConcurrentGames) : '')
      setKitsDraft(next.maxKitsPerCourse != null ? String(next.maxKitsPerCourse) : '')
      setSaved(true)
    } catch (e) {
      setError(e instanceof Error ? e.message : t('admin.liveQuiz.saveError'))
    } finally {
      setSaving(false)
    }
  }

  async function saveCaps() {
    const patch: Parameters<typeof patchAdminIQSettings>[0] = {}
    if (concurrentDraft.trim() === '') {
      patch.clearMaxConcurrentGames = true
    } else {
      const n = Number(concurrentDraft)
      if (!Number.isFinite(n) || n < 0) {
        setError(t('admin.liveQuiz.capInvalid'))
        return
      }
      patch.maxConcurrentGames = n
    }
    if (kitsDraft.trim() === '') {
      patch.clearMaxKitsPerCourse = true
    } else {
      const n = Number(kitsDraft)
      if (!Number.isFinite(n) || n < 0) {
        setError(t('admin.liveQuiz.capInvalid'))
        return
      }
      patch.maxKitsPerCourse = n
    }
    await persist(patch)
  }

  if (loading && !settings) {
    return (
      <main className="mx-auto max-w-4xl p-6">
        <p className="text-sm" role="status">
          {t('common.loading')}
        </p>
      </main>
    )
  }

  return (
    <main className="mx-auto max-w-4xl p-6">
      <h1 id={titleId} className="text-xl font-bold text-slate-900 dark:text-neutral-100">
        {t('admin.liveQuiz.title')}
      </h1>
      <p className="mt-1 text-sm text-slate-600 dark:text-neutral-400">{t('admin.liveQuiz.subtitle')}</p>

      {error ? (
        <p className="mt-4 text-sm text-red-600 dark:text-red-400" role="alert">
          {error}
        </p>
      ) : null}
      {saved ? (
        <p className="mt-4 text-sm text-green-700 dark:text-green-400" role="status">
          {t('admin.liveQuiz.saved')}
        </p>
      ) : null}

      {loading || !settings || !analytics ? (
        <p className="mt-6 text-sm" role="status">
          {t('common.loading')}
        </p>
      ) : (
        <div className="mt-6 space-y-10">
          <section aria-labelledby={`${titleId}-analytics`}>
            <h2 id={`${titleId}-analytics`} className="text-base font-semibold">
              {t('admin.liveQuiz.analyticsTitle')}
            </h2>
            {analytics.liveGamesNow >= (settings.maxConcurrentGames ?? Number.POSITIVE_INFINITY) &&
            settings.maxConcurrentGames != null ? (
              <p className="mt-2 text-sm text-amber-700 dark:text-amber-400" role="status">
                {t('admin.liveQuiz.quotaBreach')}
              </p>
            ) : null}
            <dl className="mt-3 grid grid-cols-2 gap-3 sm:grid-cols-3">
              <Stat label={t('admin.liveQuiz.games')} value={analytics.games} />
              <Stat label={t('admin.liveQuiz.liveNow')} value={analytics.liveGamesNow} />
              <Stat label={t('admin.liveQuiz.hosts')} value={analytics.uniqueHosts} />
              <Stat label={t('admin.liveQuiz.players')} value={analytics.uniquePlayers} />
              <Stat label={t('admin.liveQuiz.answers')} value={analytics.answersSubmitted} />
              <Stat
                label={t('admin.liveQuiz.avgParticipation')}
                value={analytics.avgParticipation.toFixed(1)}
              />
              <Stat label={t('admin.liveQuiz.guests')} value={analytics.guestPlayers} />
              <Stat label={t('admin.liveQuiz.enrolled')} value={analytics.enrolledPlayers} />
              <Stat label={t('admin.liveQuiz.aiCost')} value={formatCents(analytics.aiCostCents)} />
              <Stat label={t('admin.liveQuiz.coursesUsing')} value={analytics.coursesUsing} />
              <Stat label={t('admin.liveQuiz.pendingReviews')} value={analytics.pendingReviewCount} />
            </dl>
            <table className="mt-4 w-full text-start text-sm">
              <caption className="mb-2 text-start text-xs text-slate-500">
                {t('admin.liveQuiz.gamesByMode')}
              </caption>
              <thead>
                <tr className="border-b border-slate-200 text-slate-500 dark:border-neutral-700">
                  <th scope="col" className="py-1 font-medium">
                    {t('admin.liveQuiz.mode')}
                  </th>
                  <th scope="col" className="py-1 font-medium">
                    {t('admin.liveQuiz.count')}
                  </th>
                </tr>
              </thead>
              <tbody>
                {Object.keys(analytics.gamesByMode).length === 0 ? (
                  <tr>
                    <td colSpan={2} className="py-2 text-slate-500">
                      {t('admin.liveQuiz.noData')}
                    </td>
                  </tr>
                ) : (
                  Object.entries(analytics.gamesByMode).map(([mode, count]) => (
                    <tr key={mode} className="border-b border-slate-100 dark:border-neutral-800">
                      <td className="py-1">{mode}</td>
                      <td className="py-1">{count}</td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </section>

          <section aria-labelledby={`${titleId}-settings`}>
            <h2 id={`${titleId}-settings`} className="text-base font-semibold">
              {t('admin.liveQuiz.settingsTitle')}
            </h2>
            <div className="mt-3 space-y-4">
              <label className="block text-sm">
                <span className="font-medium">{t('admin.liveQuiz.guestPolicy')}</span>
                <select
                  className="mt-1 block w-full max-w-xs rounded border border-slate-300 bg-white px-2 py-1 dark:border-neutral-600 dark:bg-neutral-900"
                  value={settings.guestJoinPolicy}
                  disabled={saving}
                  onChange={(e) =>
                    void persist({ guestJoinPolicy: e.target.value as IQGuestJoinPolicy })
                  }
                >
                  <option value="disabled">{t('admin.liveQuiz.guestDisabled')}</option>
                  <option value="teacher_mediated">{t('admin.liveQuiz.guestTeacher')}</option>
                  <option value="open">{t('admin.liveQuiz.guestOpen')}</option>
                </select>
              </label>
              <label className="block text-sm">
                <span className="font-medium">{t('admin.liveQuiz.maxPlayers')}</span>
                <input
                  type="number"
                  min={1}
                  className="mt-1 block w-40 rounded border border-slate-300 px-2 py-1 dark:border-neutral-600 dark:bg-neutral-900"
                  defaultValue={settings.maxPlayersPerGame}
                  disabled={saving}
                  onBlur={(e) => {
                    const n = Number(e.target.value)
                    if (Number.isFinite(n) && n > 0 && n !== settings.maxPlayersPerGame) {
                      void persist({ maxPlayersPerGame: n })
                    }
                  }}
                />
              </label>
              <label className="block text-sm">
                <span className="font-medium">{t('admin.liveQuiz.retentionDays')}</span>
                <input
                  type="number"
                  min={1}
                  className="mt-1 block w-40 rounded border border-slate-300 px-2 py-1 dark:border-neutral-600 dark:bg-neutral-900"
                  defaultValue={settings.retentionDays}
                  disabled={saving}
                  onBlur={(e) => {
                    const n = Number(e.target.value)
                    if (Number.isFinite(n) && n > 0 && n !== settings.retentionDays) {
                      void persist({ retentionDays: n })
                    }
                  }}
                />
              </label>
              <label className="flex items-start gap-3 text-sm">
                <input
                  type="checkbox"
                  className="mt-1"
                  checked={settings.aiGenerationEnabled}
                  disabled={saving}
                  onChange={(e) => void persist({ aiGenerationEnabled: e.target.checked })}
                />
                <span>
                  <span className="font-medium">{t('admin.liveQuiz.aiEnabled')}</span>
                  <span className="mt-0.5 block text-slate-600 dark:text-neutral-400">
                    {t('admin.liveQuiz.aiEnabledHint')}
                  </span>
                </span>
              </label>
              <div className="flex flex-wrap items-end gap-3">
                <label className="block text-sm">
                  <span className="font-medium">{t('admin.liveQuiz.maxConcurrent')}</span>
                  <input
                    type="text"
                    inputMode="numeric"
                    className="mt-1 block w-40 rounded border border-slate-300 px-2 py-1 dark:border-neutral-600 dark:bg-neutral-900"
                    value={concurrentDraft}
                    disabled={saving}
                    placeholder={t('admin.liveQuiz.unlimited')}
                    onChange={(e) => setConcurrentDraft(e.target.value)}
                  />
                </label>
                <label className="block text-sm">
                  <span className="font-medium">{t('admin.liveQuiz.maxKits')}</span>
                  <input
                    type="text"
                    inputMode="numeric"
                    className="mt-1 block w-40 rounded border border-slate-300 px-2 py-1 dark:border-neutral-600 dark:bg-neutral-900"
                    value={kitsDraft}
                    disabled={saving}
                    placeholder={t('admin.liveQuiz.unlimited')}
                    onChange={(e) => setKitsDraft(e.target.value)}
                  />
                </label>
                <button
                  type="button"
                  className="rounded bg-slate-900 px-3 py-1.5 text-sm text-white disabled:opacity-50 dark:bg-neutral-100 dark:text-neutral-900"
                  disabled={saving}
                  onClick={() => void saveCaps()}
                >
                  {t('admin.liveQuiz.saveCaps')}
                </button>
              </div>
            </div>
          </section>

          <section aria-labelledby={`${titleId}-queue`}>
            <h2 id={`${titleId}-queue`} className="text-base font-semibold">
              {t('admin.liveQuiz.reviewTitle')} ({pendingCount})
            </h2>
            {queue.length === 0 ? (
              <p className="mt-2 text-sm text-slate-500">{t('admin.liveQuiz.emptyQueue')}</p>
            ) : (
              <ul className="mt-3 space-y-3">
                {queue.map((item) => (
                  <li
                    key={item.id}
                    className="rounded border border-slate-200 p-3 dark:border-neutral-700"
                  >
                    <div className="text-sm font-medium">
                      {item.kitTitle || item.kitId || item.id}{' '}
                      <span className="font-normal text-slate-500">({item.kind})</span>
                    </div>
                    <div className="mt-2 flex flex-wrap items-center gap-2">
                      <button
                        type="button"
                        className="rounded bg-emerald-700 px-2 py-1 text-xs text-white"
                        disabled={saving}
                        onClick={() =>
                          void postAdminIQReviewAction(item.id, 'approve').then(() => load())
                        }
                      >
                        {t('admin.liveQuiz.approve')}
                      </button>
                      <input
                        type="text"
                        className="min-w-[12rem] flex-1 rounded border border-slate-300 px-2 py-1 text-xs dark:border-neutral-600 dark:bg-neutral-900"
                        placeholder={t('admin.liveQuiz.rejectReason')}
                        value={rejectReasons[item.id] ?? ''}
                        onChange={(e) =>
                          setRejectReasons((prev) => ({ ...prev, [item.id]: e.target.value }))
                        }
                      />
                      <button
                        type="button"
                        className="rounded bg-red-700 px-2 py-1 text-xs text-white"
                        disabled={saving}
                        onClick={() =>
                          void postAdminIQReviewAction(
                            item.id,
                            'reject',
                            rejectReasons[item.id] || t('admin.liveQuiz.defaultRejectReason'),
                          ).then(() => load())
                        }
                      >
                        {t('admin.liveQuiz.reject')}
                      </button>
                    </div>
                  </li>
                ))}
              </ul>
            )}
          </section>

          <section aria-labelledby={`${titleId}-live`}>
            <h2 id={`${titleId}-live`} className="text-base font-semibold">
              {t('admin.liveQuiz.liveGamesTitle')}
            </h2>
            {games.length === 0 ? (
              <p className="mt-2 text-sm text-slate-500">{t('admin.liveQuiz.noLiveGames')}</p>
            ) : (
              <table className="mt-3 w-full text-start text-sm">
                <thead>
                  <tr className="border-b border-slate-200 text-slate-500 dark:border-neutral-700">
                    <th scope="col" className="py-1 font-medium">
                      {t('admin.liveQuiz.course')}
                    </th>
                    <th scope="col" className="py-1 font-medium">
                      {t('admin.liveQuiz.status')}
                    </th>
                    <th scope="col" className="py-1 font-medium">
                      {t('admin.liveQuiz.players')}
                    </th>
                    <th scope="col" className="py-1 font-medium">
                      {t('admin.liveQuiz.actions')}
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {games.map((g) => (
                    <tr key={g.id} className="border-b border-slate-100 dark:border-neutral-800">
                      <td className="py-1">
                        {g.courseCode}
                        {g.joinCode ? ` · ${g.joinCode}` : ''}
                      </td>
                      <td className="py-1">{g.status}</td>
                      <td className="py-1">{g.players}</td>
                      <td className="py-1">
                        <button
                          type="button"
                          className="rounded border border-slate-300 px-2 py-0.5 text-xs dark:border-neutral-600"
                          onClick={() => void postAdminIQForceEnd(g.id).then(() => load())}
                        >
                          {t('admin.liveQuiz.forceEnd')}
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
            <button
              type="button"
              className="mt-4 rounded border border-slate-300 px-3 py-1.5 text-sm dark:border-neutral-600"
              onClick={() =>
                void postAdminIQBulkArchiveKits(365, orgId).then((r) => {
                  setSaved(true)
                  setError(null)
                  void load()
                  if (r.archived === 0) {
                    /* keep quiet */
                  }
                })
              }
            >
              {t('admin.liveQuiz.bulkArchive')}
            </button>
          </section>
        </div>
      )}
    </main>
  )
}

function Stat({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="rounded-md border border-slate-200 p-3 dark:border-neutral-700">
      <dt className="text-xs text-slate-500">{label}</dt>
      <dd className="text-lg font-semibold">{value}</dd>
    </div>
  )
}
