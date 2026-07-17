import { useCallback, useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { LmsPage } from './lms-page'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { getAccessToken } from '../../lib/auth'
import { useCoursePageTitle } from '../../context/course-document-title-context'
import { fetchMyGameResults, gameReportExportUrl, type MyResults } from '../../lib/live-quiz-api'

export default function LiveQuizMyResultsPage() {
  const { t } = useTranslation()
  const { courseCode = '', gameId = '' } = useParams()
  const { ffInteractiveQuizzes } = usePlatformFeatures()
  const [data, setData] = useState<MyResults | null>(null)
  const [error, setError] = useState<string | null>(null)

  useCoursePageTitle(t('liveQuiz.myResults.title'))

  const load = useCallback(async () => {
    if (!courseCode || !gameId) return
    if (!ffInteractiveQuizzes) {
      setError(t('liveQuiz.error.disabled'))
      return
    }
    try {
      setData(await fetchMyGameResults(courseCode, gameId))
    } catch (e) {
      setError(e instanceof Error ? e.message : t('liveQuiz.myResults.errorLoad'))
    }
  }, [courseCode, gameId, ffInteractiveQuizzes, t])

  useEffect(() => {
    void load()
  }, [load])

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
    a.download = format === 'csv' ? 'my-quiz-results.csv' : 'my-quiz-results.html'
    a.click()
    URL.revokeObjectURL(a.href)
  }

  const base = `/courses/${encodeURIComponent(courseCode)}/live-quizzes`

  if (error && !data) {
    return (
      <LmsPage title={t('liveQuiz.myResults.title')}>
        <p className="text-destructive">{error}</p>
        <Link to={base} className="underline">
          {t('liveQuiz.kit.backToGallery')}
        </Link>
      </LmsPage>
    )
  }

  return (
    <LmsPage title={t('liveQuiz.myResults.title')}>
      <div className="mb-4 flex flex-wrap items-center gap-3">
        <div>
          <h1 className="text-2xl font-semibold">{t('liveQuiz.myResults.title')}</h1>
          <p className="text-sm text-muted-foreground">{t('liveQuiz.myResults.subtitle')}</p>
        </div>
        <div className="ms-auto flex gap-2">
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
        </div>
      </div>

      {data && (
        <>
          <section className="mb-6 grid gap-3 sm:grid-cols-3">
            <div className="rounded-md border p-3">
              <p className="text-xs text-muted-foreground">{t('liveQuiz.myResults.score')}</p>
              <p className="text-2xl font-semibold tabular-nums">{data.totalScore}</p>
            </div>
            <div className="rounded-md border p-3">
              <p className="text-xs text-muted-foreground">{t('liveQuiz.myResults.rank')}</p>
              <p className="text-2xl font-semibold tabular-nums">
                {data.rank} / {data.playerCount}
              </p>
            </div>
            <div className="rounded-md border p-3">
              <p className="text-xs text-muted-foreground">{t('liveQuiz.myResults.correct')}</p>
              <p className="text-2xl font-semibold tabular-nums">
                {data.correct} / {data.answered}
              </p>
            </div>
          </section>

          <section aria-labelledby="review-heading">
            <h2 id="review-heading" className="mb-2 text-lg font-semibold">
              {t('liveQuiz.myResults.reviewThese')}
            </h2>
            {data.reviewThese.length === 0 ? (
              <p className="text-sm text-muted-foreground">{t('liveQuiz.myResults.reviewEmpty')}</p>
            ) : (
              <div className="space-y-2">
                {data.reviewThese.map((item) => (
                  <details key={item.index} className="rounded-md border p-3" open>
                    <summary className="cursor-pointer font-medium">
                      {t('liveQuiz.myResults.reviewItem', {
                        n: item.index + 1,
                        reason: t(`liveQuiz.myResults.reason.${item.reason}`, {
                          defaultValue: item.reason,
                        }),
                      })}
                      : {item.prompt}
                    </summary>
                    <div className="mt-2 space-y-1 text-sm text-muted-foreground">
                      {item.correctOptionIds && item.correctOptionIds.length > 0 && (
                        <p>
                          {t('liveQuiz.myResults.correctAnswer')}:{' '}
                          {item.correctOptionIds.join(', ')}
                        </p>
                      )}
                      {item.explanation && <p>{item.explanation}</p>}
                      <p>
                        {t('liveQuiz.myResults.yourPoints')}: {item.points} ·{' '}
                        {t('liveQuiz.myResults.responseMs', { ms: item.responseMs })}
                      </p>
                    </div>
                  </details>
                ))}
              </div>
            )}
          </section>
        </>
      )}

      <p className="mt-6">
        <Link to={base} className="underline">
          {t('liveQuiz.kit.backToGallery')}
        </Link>
      </p>
    </LmsPage>
  )
}
