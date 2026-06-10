import { useCallback, useEffect, useId, useState } from 'react'
import { Link } from 'react-router-dom'
import { Award, Download, Link2, Loader2 } from 'lucide-react'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  downloadCCR,
  fetchMyCCR,
  generateMyCCR,
  type CCRAchievement,
  type CCRDocument,
} from '../../lib/ccr-api'
import { LmsPage } from './lms-page'

function groupByType(achievements: CCRAchievement[]): Record<string, CCRAchievement[]> {
  const groups: Record<string, CCRAchievement[]> = {}
  for (const a of achievements) {
    const key = a.type.replace(/_/g, ' ')
    groups[key] = groups[key] ?? []
    groups[key].push(a)
  }
  return groups
}

export default function MyCCR() {
  const titleId = useId()
  const { ffCoCurricularTranscript, loading: featuresLoading } = usePlatformFeatures()
  const [achievements, setAchievements] = useState<CCRAchievement[]>([])
  const [documents, setDocuments] = useState<CCRDocument[]>([])
  const [loading, setLoading] = useState(false)
  const [generating, setGenerating] = useState(false)
  const [sharePublicly, setSharePublicly] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [copiedUrl, setCopiedUrl] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const data = await fetchMyCCR()
      setAchievements(data.achievements)
      setDocuments(data.documents)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load CCR.')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (featuresLoading || !ffCoCurricularTranscript) return
    void load()
  }, [featuresLoading, ffCoCurricularTranscript, load])

  async function handleGenerate() {
    setGenerating(true)
    setError(null)
    try {
      const result = await generateMyCCR(sharePublicly)
      setDocuments((prev) => [result.document, ...prev])
      setAchievements(result.achievements)
      if (result.verificationUrl) {
        setCopiedUrl(result.verificationUrl)
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Generation failed.')
    } finally {
      setGenerating(false)
    }
  }

  async function handleDownload(docId: string, format: 'json' | 'pdf') {
    try {
      const blob = await downloadCCR(docId, format)
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = format === 'pdf' ? 'ccr.pdf' : 'ccr.json'
      a.click()
      URL.revokeObjectURL(url)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Download failed.')
    }
  }

  async function copyVerification(url: string) {
    await navigator.clipboard.writeText(url)
    setCopiedUrl(url)
  }

  if (featuresLoading) {
    return <p>Loading…</p>
  }

  if (!ffCoCurricularTranscript) {
    return (
      <LmsPage title="My Achievements">
        <p role="alert">Co-curricular transcript is not enabled for this institution.</p>
      </LmsPage>
    )
  }

  const grouped = groupByType(achievements)

  return (
    <LmsPage title="My Achievements">
      <main aria-labelledby={titleId}>
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <h1 id={titleId} className="text-2xl font-semibold text-slate-900 dark:text-neutral-100">
              Comprehensive Learner Record
            </h1>
            <p className="mt-1 text-sm text-slate-600 dark:text-neutral-400">
              Generate a verifiable record of your courses, badges, and co-curricular achievements.
            </p>
          </div>
          <div className="flex flex-col items-end gap-2">
            <label className="flex items-center gap-2 text-sm text-slate-700 dark:text-neutral-300">
              <input
                type="checkbox"
                checked={sharePublicly}
                onChange={(e) => setSharePublicly(e.target.checked)}
              />
              Create shareable verification link
            </label>
            <button
              type="button"
              onClick={() => void handleGenerate()}
              disabled={generating}
              className="inline-flex items-center gap-2 rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700 disabled:opacity-60"
            >
              {generating ? <Loader2 className="h-4 w-4 animate-spin" aria-hidden /> : <Award className="h-4 w-4" aria-hidden />}
              {generating ? 'Generating…' : 'Generate CCR'}
            </button>
          </div>
        </div>

        {error ? (
          <p role="alert" className="mt-4 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800 dark:border-red-900/40 dark:bg-red-950/30 dark:text-red-200">
            {error}
          </p>
        ) : null}

        {loading ? <p className="mt-6 text-sm text-slate-600">Loading achievements…</p> : null}

        {!loading && achievements.length === 0 ? (
          <p className="mt-6 rounded-2xl border border-slate-200 bg-slate-50 px-4 py-6 text-sm text-slate-700 dark:border-neutral-700 dark:bg-neutral-900/50 dark:text-neutral-200">
            Complete courses and earn badges to build your CCR.
          </p>
        ) : null}

        {Object.entries(grouped).map(([type, items]) => (
          <section key={type} className="mt-8" aria-label={type}>
            <h2 className="text-lg font-semibold capitalize text-slate-900 dark:text-neutral-100">{type}</h2>
            <ul className="mt-3 space-y-2">
              {items.map((item) => (
                <li
                  key={item.id}
                  className="rounded-xl border border-slate-200 bg-white px-4 py-3 dark:border-neutral-700 dark:bg-neutral-900"
                >
                  <p className="font-medium text-slate-900 dark:text-neutral-100">{item.title}</p>
                  {item.description ? (
                    <p className="mt-1 text-sm text-slate-600 dark:text-neutral-400">{item.description}</p>
                  ) : null}
                  <p className="mt-1 text-xs text-slate-500 dark:text-neutral-500">{item.issuedAt}</p>
                </li>
              ))}
            </ul>
          </section>
        ))}

        {documents.length > 0 ? (
          <section className="mt-10" aria-label="Generated records">
            <h2 className="text-lg font-semibold text-slate-900 dark:text-neutral-100">Generated records</h2>
            <ul className="mt-3 space-y-3">
              {documents.map((doc) => (
                <li
                  key={doc.id}
                  className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-slate-200 bg-white px-4 py-3 dark:border-neutral-700 dark:bg-neutral-900"
                >
                  <div>
                    <p className="font-medium text-slate-900 dark:text-neutral-100">{doc.generatedAt}</p>
                    {doc.shareable && doc.verificationUrl ? (
                      <p className="mt-1 text-xs text-slate-500 break-all">{doc.verificationUrl}</p>
                    ) : (
                      <p className="mt-1 text-xs text-slate-500">Private — no public verification link</p>
                    )}
                  </div>
                  <div className="flex flex-wrap gap-2">
                    <button
                      type="button"
                      aria-label="Download CCR JSON"
                      onClick={() => void handleDownload(doc.id, 'json')}
                      className="inline-flex items-center gap-1 rounded-lg border border-slate-200 px-3 py-1.5 text-sm hover:bg-slate-50 dark:border-neutral-600 dark:hover:bg-neutral-800"
                    >
                      <Download className="h-4 w-4" aria-hidden />
                      JSON
                    </button>
                    <button
                      type="button"
                      aria-label="Download CCR PDF"
                      onClick={() => void handleDownload(doc.id, 'pdf')}
                      className="inline-flex items-center gap-1 rounded-lg border border-slate-200 px-3 py-1.5 text-sm hover:bg-slate-50 dark:border-neutral-600 dark:hover:bg-neutral-800"
                    >
                      <Download className="h-4 w-4" aria-hidden />
                      PDF
                    </button>
                    {doc.verificationUrl ? (
                      <button
                        type="button"
                        aria-label="Copy verification link"
                        onClick={() => void copyVerification(doc.verificationUrl!)}
                        className="inline-flex items-center gap-1 rounded-lg border border-violet-200 px-3 py-1.5 text-sm text-violet-800 hover:bg-violet-50 dark:border-violet-800 dark:text-violet-200 dark:hover:bg-violet-950/40"
                      >
                        <Link2 className="h-4 w-4" aria-hidden />
                        Copy link
                      </button>
                    ) : null}
                  </div>
                </li>
              ))}
            </ul>
          </section>
        ) : null}

        {copiedUrl ? (
          <p role="status" className="mt-4 text-sm text-green-700 dark:text-green-300">
            Verification link ready:{' '}
            <Link to={copiedUrl.replace(/^https?:\/\/[^/]+/, '')} className="underline">
              open verify page
            </Link>
          </p>
        ) : null}
      </main>
    </LmsPage>
  )
}
