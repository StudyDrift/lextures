import { useCallback, useEffect, useId, useState } from 'react'
import { Download, Loader2 } from 'lucide-react'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  downloadCETranscriptPdf,
  fetchCETranscript,
  type CETranscriptAward,
} from '../../lib/seat-time-api'
import { LmsPage } from './lms-page'

export default function CeTranscript() {
  const titleId = useId()
  const { ffCeuTracking, loading: featuresLoading } = usePlatformFeatures()
  const [awards, setAwards] = useState<CETranscriptAward[]>([])
  const [loading, setLoading] = useState(false)
  const [downloading, setDownloading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const data = await fetchCETranscript()
      setAwards(data.awards)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load CE transcript.')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (featuresLoading || !ffCeuTracking) return
    void load()
  }, [featuresLoading, ffCeuTracking, load])

  async function handleDownload() {
    setDownloading(true)
    setError(null)
    try {
      const blob = await downloadCETranscriptPdf()
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = 'ce-transcript.pdf'
      a.click()
      URL.revokeObjectURL(url)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Download failed.')
    } finally {
      setDownloading(false)
    }
  }

  if (featuresLoading) {
    return <p>Loading…</p>
  }

  if (!ffCeuTracking) {
    return (
      <LmsPage title="CE Transcript">
        <p role="alert">CEU tracking is not enabled for this institution.</p>
      </LmsPage>
    )
  }

  return (
    <LmsPage title="CE Transcript">
      <div className="mx-auto max-w-3xl space-y-6">
        <header>
          <h1 id={titleId} className="text-2xl font-semibold text-fg-default">
            Continuing education transcript
          </h1>
          <p className="mt-2 text-sm text-fg-muted">
            Download a record of CEU-bearing courses you have completed.
          </p>
        </header>

        {error ? (
          <p role="alert" className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800 dark:border-red-900/50 dark:bg-red-950/30 dark:text-red-200">
            {error}
          </p>
        ) : null}

        <div className="flex flex-wrap gap-3">
          <button
            type="button"
            onClick={() => void handleDownload()}
            disabled={downloading}
            className="inline-flex items-center gap-2 rounded-lg bg-teal-600 px-4 py-2 text-sm font-medium text-white hover:bg-teal-700 disabled:opacity-60"
          >
            {downloading ? <Loader2 className="h-4 w-4 animate-spin" aria-hidden /> : <Download className="h-4 w-4" aria-hidden />}
            Download PDF
          </button>
        </div>

        {loading ? (
          <p className="text-sm text-fg-muted">Loading awards…</p>
        ) : awards.length === 0 ? (
          <p className="rounded-xl border border-border-default bg-surface-base px-4 py-6 text-sm text-fg-muted dark:border-border-default/50 dark:text-fg-muted">
            No CEU awards yet. Complete contact-hour requirements in your CE courses to earn credits.
          </p>
        ) : (
          <div className="overflow-x-auto rounded-xl border border-border-default">
            <table className="min-w-full text-left text-sm">
              <thead className="bg-surface-base text-xs uppercase tracking-wide text-fg-muted dark:bg-surface-raised dark:text-fg-muted">
                <tr>
                  <th className="px-4 py-3 font-semibold">Course</th>
                  <th className="px-4 py-3 font-semibold">CEU</th>
                  <th className="px-4 py-3 font-semibold">Contact hours</th>
                  <th className="px-4 py-3 font-semibold">Completed</th>
                </tr>
              </thead>
              <tbody>
                {awards.map((a) => (
                  <tr key={`${a.courseTitle}-${a.completedAt}`} className="border-t border-border-subtle">
                    <td className="px-4 py-3 font-medium text-fg-default">{a.courseTitle}</td>
                    <td className="px-4 py-3">{a.ceuCredit.toFixed(2)}</td>
                    <td className="px-4 py-3">{a.contactHours.toFixed(1)}</td>
                    <td className="px-4 py-3">{new Date(a.completedAt).toLocaleDateString()}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </LmsPage>
  )
}
