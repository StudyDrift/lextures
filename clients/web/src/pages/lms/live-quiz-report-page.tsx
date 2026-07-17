import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { GradebookLinkDialog } from '../../components/live-quiz/gradebook-link-dialog'
import { LmsPage } from './lms-page'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { getAccessToken } from '../../lib/auth'
import { useCoursePageTitle } from '../../context/course-document-title-context'
import {
  fetchGameReport,
  fetchGameSafetyEvents,
  gameReportExportUrl,
  rebuildGameReport,
  type GameReportResponse,
  type IntegrityFlag,
  type QuestionAggregate,
} from '../../lib/live-quiz-api'

export default function LiveQuizReportPage() {
  const { t } = useTranslation()
  const { courseCode = '', gameId = '' } = useParams()
  const { ffIqGradebookPush } = usePlatformFeatures()
  const [data, setData] = useState<GameReportResponse | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [q, setQ] = useState('')
  const [sortKey, setSortKey] = useState<'rank' | 'score' | 'nickname'>('rank')
  const [gbOpen, setGbOpen] = useState(false)
  const [busy, setBusy] = useState(false)
  const [integrityFlags, setIntegrityFlags] = useState<IntegrityFlag[]>([])

  useCoursePageTitle(data?.title ?? t('liveQuiz.report.title'))

  const load = useCallback(async () => {
    if (!courseCode || !gameId) return
    setError(null)
    try {
      setData(await fetchGameReport(courseCode, gameId))
      try {
        const safety = await fetchGameSafetyEvents(courseCode, gameId)
        setIntegrityFlags(safety.integrityFlags ?? [])
      } catch {
        setIntegrityFlags([])
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : t('liveQuiz.report.errorLoad'))
    }
  }, [courseCode, gameId, t])

  useEffect(() => {
    void load()
  }, [load])

  const hardest = useMemo(() => {
    const qs = data?.report.perQuestion ?? []
    return [...qs]
      .filter((x) => x.answerCount > 0)
      .sort((a, b) => (a.hardestRank ?? 99) - (b.hardestRank ?? 99))
      .slice(0, 3)
  }, [data])

  const players = useMemo(() => {
    let rows = data?.players ?? []
    if (q.trim()) {
      const needle = q.trim().toLowerCase()
      rows = rows.filter((p) => p.nickname.toLowerCase().includes(needle))
    }
    const sorted = [...rows]
    sorted.sort((a, b) => {
      if (sortKey === 'score') return b.totalScore - a.totalScore
      if (sortKey === 'nickname') return a.nickname.localeCompare(b.nickname)
      return (a.rank || 999) - (b.rank || 999)
    })
    return sorted
  }, [data, q, sortKey])

  async function handleRebuild() {
    setBusy(true)
    try {
      await rebuildGameReport(courseCode, gameId)
      await load()
    } catch (e) {
      setError(e instanceof Error ? e.message : t('liveQuiz.report.errorLoad'))
    } finally {
      setBusy(false)
    }
  }

  async function downloadExport(format: 'csv' | 'pdf') {
    const url = gameReportExportUrl(courseCode, gameId, format)
    const tok = getAccessToken()
    const res = await fetch(url, {
      headers: tok ? { Authorization: `Bearer ${tok}` } : {},
    })
    if (!res.ok) {
      setError(t('liveQuiz.report.exportError'))
      return
    }
    const blob = await res.blob()
    const a = document.createElement('a')
    a.href = URL.createObjectURL(blob)
    a.download = format === 'csv' ? 'live-quiz-report.csv' : 'live-quiz-report.html'
    a.click()
    URL.revokeObjectURL(a.href)
  }

  const base = `/courses/${encodeURIComponent(courseCode)}/live-quizzes`

  if (error && !data) {
    return (
      <LmsPage title={t('liveQuiz.report.title')}>
        <p className="text-destructive">{error}</p>
        <Link to={base} className="underline">
          {t('liveQuiz.kit.backToGallery')}
        </Link>
      </LmsPage>
    )
  }

  const rep = data?.report

  return (
    <LmsPage title={t('liveQuiz.report.title')}>
      <div className="mb-4 flex flex-wrap items-center gap-3">
        <div>
          <h1 className="text-2xl font-semibold">{data?.title ?? t('liveQuiz.report.title')}</h1>
          <p className="text-sm text-muted-foreground">{t('liveQuiz.report.subtitle')}</p>
        </div>
        <div className="ms-auto flex flex-wrap gap-2">
          <button
            type="button"
            className="rounded-md border px-3 py-2 text-sm"
            onClick={() => void downloadExport('csv')}
          >
            {t('liveQuiz.report.exportCsv')}
          </button>
          <button
            type="button"
            className="rounded-md border px-3 py-2 text-sm"
            onClick={() => void downloadExport('pdf')}
          >
            {t('liveQuiz.report.exportPdf')}
          </button>
          {ffIqGradebookPush && (
            <button
              type="button"
              className="rounded-md bg-primary px-3 py-2 text-sm text-primary-foreground"
              onClick={() => setGbOpen(true)}
            >
              {data?.gradebookLink
                ? t('liveQuiz.gradebook.manage')
                : t('liveQuiz.gradebook.push')}
            </button>
          )}
          <button
            type="button"
            className="rounded-md border px-3 py-2 text-sm disabled:opacity-50"
            disabled={busy}
            onClick={() => void handleRebuild()}
          >
            {t('liveQuiz.report.rebuild')}
          </button>
        </div>
      </div>

      {error && <p className="mb-3 text-sm text-destructive">{error}</p>}

      {rep && (
        <>
          <section className="mb-6 grid gap-3 sm:grid-cols-2 lg:grid-cols-4" aria-label={t('liveQuiz.report.summary')}>
            <StatTile label={t('liveQuiz.report.players')} value={String(rep.playerCount)} />
            <StatTile label={t('liveQuiz.report.answered')} value={String(rep.answeredCount)} />
            <StatTile
              label={t('liveQuiz.report.avgScore')}
              value={rep.scoreAvg != null ? String(rep.scoreAvg) : '—'}
            />
            <StatTile
              label={t('liveQuiz.report.medianScore')}
              value={rep.scoreMedian != null ? String(rep.scoreMedian) : '—'}
            />
          </section>

          {(data?.guestCount ?? 0) > 0 && (
            <p className="mb-4 text-sm text-muted-foreground">
              {t('liveQuiz.report.guestNote', { count: data?.guestCount ?? 0 })}
            </p>
          )}

          <section className="mb-6" aria-labelledby="integrity-heading">
            <h2 id="integrity-heading" className="mb-2 text-lg font-semibold">
              {t('liveQuiz.report.integrityFlags')}
            </h2>
            {integrityFlags.length === 0 ? (
              <p className="text-sm text-muted-foreground">{t('liveQuiz.report.integrityEmpty')}</p>
            ) : (
              <ul className="list-disc space-y-1 ps-5 text-sm">
                {integrityFlags.map((f, i) => (
                  <li key={`${f.kind}-${i}`}>
                    {f.kind}: {f.detail}
                    {f.playerId ? ` (${f.playerId.slice(0, 8)})` : ''}
                  </li>
                ))}
              </ul>
            )}
          </section>

          {hardest.length > 0 && (
            <section className="mb-6" aria-labelledby="hardest-heading">
              <h2 id="hardest-heading" className="mb-2 text-lg font-semibold">
                {t('liveQuiz.report.hardest')}
              </h2>
              <ul className="list-disc space-y-1 ps-5 text-sm">
                {hardest.map((qItem) => (
                  <li key={qItem.index}>
                    {t('liveQuiz.report.hardestItem', {
                      n: qItem.index + 1,
                      pct: qItem.correctPct,
                      prompt: qItem.prompt,
                    })}
                  </li>
                ))}
              </ul>
            </section>
          )}

          <section className="mb-6" aria-labelledby="perq-heading">
            <h2 id="perq-heading" className="mb-2 text-lg font-semibold">
              {t('liveQuiz.report.perQuestion')}
            </h2>
            <QuestionBars questions={rep.perQuestion} />
            <table className="mt-3 w-full text-start text-sm">
              <caption className="sr-only">{t('liveQuiz.report.perQuestionTable')}</caption>
              <thead>
                <tr className="border-b">
                  <th className="py-2 pe-2">#</th>
                  <th className="py-2 pe-2">{t('liveQuiz.report.prompt')}</th>
                  <th className="py-2 pe-2">{t('liveQuiz.report.correctPct')}</th>
                  <th className="py-2 pe-2">{t('liveQuiz.report.avgMs')}</th>
                  <th className="py-2">{t('liveQuiz.report.answers')}</th>
                </tr>
              </thead>
              <tbody>
                {rep.perQuestion.map((row) => (
                  <tr key={row.index} className="border-b border-border/40">
                    <td className="py-2 pe-2">{row.index + 1}</td>
                    <td className="py-2 pe-2">{row.prompt}</td>
                    <td className="py-2 pe-2">{row.correctPct}%</td>
                    <td className="py-2 pe-2">{Math.round(row.avgMs)}</td>
                    <td className="py-2">{row.answerCount}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </section>

          <section aria-labelledby="players-heading">
            <div className="mb-2 flex flex-wrap items-end gap-3">
              <h2 id="players-heading" className="text-lg font-semibold">
                {t('liveQuiz.report.playerResults')}
              </h2>
              <label className="text-sm">
                <span className="sr-only">{t('liveQuiz.report.search')}</span>
                <input
                  className="rounded-md border bg-background px-3 py-1.5"
                  placeholder={t('liveQuiz.report.search')}
                  value={q}
                  onChange={(e) => setQ(e.target.value)}
                />
              </label>
              <label className="text-sm">
                <span className="me-1">{t('liveQuiz.report.sort')}</span>
                <select
                  className="rounded-md border bg-background px-2 py-1.5"
                  value={sortKey}
                  onChange={(e) => setSortKey(e.target.value as typeof sortKey)}
                >
                  <option value="rank">{t('liveQuiz.report.sortRank')}</option>
                  <option value="score">{t('liveQuiz.report.sortScore')}</option>
                  <option value="nickname">{t('liveQuiz.report.sortName')}</option>
                </select>
              </label>
            </div>
            <table className="w-full text-start text-sm">
              <thead>
                <tr className="border-b">
                  <th className="py-2 pe-2">{t('liveQuiz.report.rank')}</th>
                  <th className="py-2 pe-2">{t('liveQuiz.report.nickname')}</th>
                  <th className="py-2 pe-2">{t('liveQuiz.report.score')}</th>
                  <th className="py-2 pe-2">{t('liveQuiz.report.correct')}</th>
                  <th className="py-2">{t('liveQuiz.report.guest')}</th>
                </tr>
              </thead>
              <tbody>
                {players.map((p) => (
                  <tr key={p.playerId} className="border-b border-border/40">
                    <td className="py-2 pe-2">{p.rank || '—'}</td>
                    <td className="py-2 pe-2">{p.nickname}</td>
                    <td className="py-2 pe-2">{p.totalScore}</td>
                    <td className="py-2 pe-2">
                      {p.correct}/{p.answered}
                    </td>
                    <td className="py-2">{p.isGuest ? t('common.yes', { defaultValue: 'Yes' }) : ''}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </section>
        </>
      )}

      <p className="mt-6">
        <Link to={base} className="underline">
          {t('liveQuiz.kit.backToGallery')}
        </Link>
      </p>

      <GradebookLinkDialog
        courseCode={courseCode}
        gameId={gameId}
        open={gbOpen}
        existing={data?.gradebookLink}
        onClose={() => setGbOpen(false)}
        onChanged={() => void load()}
      />
    </LmsPage>
  )
}

function StatTile({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-md border p-3">
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="text-2xl font-semibold tabular-nums">{value}</p>
    </div>
  )
}

function QuestionBars({ questions }: { questions: QuestionAggregate[] }) {
  const { t } = useTranslation()
  return (
    <div className="space-y-2" role="img" aria-label={t('liveQuiz.report.perQuestionChart')}>
      {questions.map((q) => (
        <div key={q.index} className="flex items-center gap-2 text-sm">
          <span className="w-8 shrink-0 tabular-nums">Q{q.index + 1}</span>
          <div className="h-3 flex-1 rounded bg-muted">
            <div
              className="h-3 rounded bg-primary/80"
              style={{ width: `${Math.min(100, Math.max(0, q.correctPct))}%` }}
            />
          </div>
          <span className="w-14 shrink-0 text-end tabular-nums">{q.correctPct}%</span>
        </div>
      ))}
    </div>
  )
}
