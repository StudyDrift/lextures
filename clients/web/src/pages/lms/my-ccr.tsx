import { useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { Award, Download, Link2, Loader2, Sparkles } from 'lucide-react'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  downloadCCR,
  fetchMyCCR,
  generateCCR,
  type CCRAchievement,
  type CCRDocumentSummary,
} from '../../lib/ccr-api'
import { readApiErrorMessage } from '../../lib/errors'
import { LmsPage } from './lms-page'

function groupAchievements(achievements: CCRAchievement[]): Record<string, CCRAchievement[]> {
  return achievements.reduce<Record<string, CCRAchievement[]>>((acc, item) => {
    const key = item.achievementType
    acc[key] = acc[key] ?? []
    acc[key].push(item)
    return acc
  }, {})
}

function typeLabel(type: string): string {
  switch (type) {
    case 'course_completion':
      return 'Course completions'
    case 'badge':
      return 'Open Badges'
    case 'certificate':
      return 'Certificates'
    case 'portfolio':
      return 'Portfolio milestones'
    case 'extracurricular':
      return 'Extracurricular'
    default:
      return type
  }
}

export default function MyCCRPage() {
  const { ffCoCurricularTranscript, loading: featuresLoading } = usePlatformFeatures()
  const [achievements, setAchievements] = useState<CCRAchievement[]>([])
  const [documents, setDocuments] = useState<CCRDocumentSummary[]>([])
  const [loading, setLoading] = useState(true)
  const [generating, setGenerating] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [consentToShare, setConsentToShare] = useState(false)
  const [copiedToken, setCopiedToken] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const data = await fetchMyCCR()
      setAchievements(data.achievements)
      setDocuments(data.documents)
    } catch (err) {
      setError(readApiErrorMessage(err))
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (!featuresLoading && ffCoCurricularTranscript) {
      void load()
    }
  }, [featuresLoading, ffCoCurricularTranscript, load])

  const onGenerate = async () => {
    setGenerating(true)
    setError(null)
    try {
      await generateCCR(consentToShare)
      await load()
    } catch (err) {
      setError(readApiErrorMessage(err))
    } finally {
      setGenerating(false)
    }
  }

  const onDownload = async (doc: CCRDocumentSummary, format: 'json' | 'pdf') => {
    try {
      const blob = await downloadCCR(doc.id, format)
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = format === 'pdf' ? 'ccr.pdf' : 'ccr.json'
      a.click()
      URL.revokeObjectURL(url)
    } catch (err) {
      setError(readApiErrorMessage(err))
    }
  }

  const copyShareLink = async (doc: CCRDocumentSummary) => {
    if (!doc.shareToken) return
    const url = `${window.location.origin}/verify/${doc.shareToken}`
    await navigator.clipboard.writeText(url)
    setCopiedToken(doc.id)
    window.setTimeout(() => setCopiedToken(null), 2000)
  }

  if (featuresLoading) {
    return (
      <LmsPage title="My Achievements">
        <p className="text-sm text-muted-foreground">Loading…</p>
      </LmsPage>
    )
  }

  if (!ffCoCurricularTranscript) {
    return (
      <LmsPage title="My Achievements">
        <p className="text-sm text-muted-foreground">Co-curricular transcripts are not enabled on this platform.</p>
      </LmsPage>
    )
  }

  const grouped = groupAchievements(achievements)

  return (
    <LmsPage title="My Achievements">
      <div className="max-w-4xl space-y-6">
        <div className="rounded-xl border bg-card p-5 shadow-sm">
          <div className="flex items-start gap-3">
            <Sparkles className="mt-1 h-5 w-5 text-primary" aria-hidden="true" />
            <div>
              <h2 className="text-lg font-semibold">Comprehensive Learner Record</h2>
              <p className="mt-1 text-sm text-muted-foreground">
                Generate a verifiable record of your courses, badges, portfolio milestones, and extracurricular achievements.
              </p>
            </div>
          </div>
          <div className="mt-4 flex flex-wrap items-center gap-3">
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={consentToShare}
                onChange={(e) => setConsentToShare(e.target.checked)}
              />
              I consent to creating a shareable verification link
            </label>
            <button
              type="button"
              className="inline-flex items-center gap-2 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground disabled:opacity-60"
              disabled={generating || achievements.length === 0}
              aria-label="Generate comprehensive learner record"
              onClick={() => void onGenerate()}
            >
              {generating ? <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" /> : <Award className="h-4 w-4" aria-hidden="true" />}
              {generating ? 'Generating…' : 'Generate CCR'}
            </button>
          </div>
          {error ? <p className="mt-3 text-sm text-destructive">{error}</p> : null}
        </div>

        {loading ? (
          <p className="text-sm text-muted-foreground">Loading achievements…</p>
        ) : achievements.length === 0 ? (
          <div className="rounded-xl border border-dashed p-8 text-center text-sm text-muted-foreground">
            Complete courses and earn badges to build your CCR.
          </div>
        ) : (
          Object.entries(grouped).map(([type, items]) => (
            <section key={type} className="rounded-xl border bg-card p-5 shadow-sm">
              <h3 className="text-base font-semibold">{typeLabel(type)}</h3>
              <ul className="mt-3 space-y-2">
                {items.map((item) => (
                  <li key={item.id} className="rounded-md border px-3 py-2 text-sm">
                    <div className="font-medium">{item.title}</div>
                    <div className="text-muted-foreground">{new Date(item.issuedAt).toLocaleDateString()}</div>
                    {item.description ? <div className="mt-1">{item.description}</div> : null}
                  </li>
                ))}
              </ul>
            </section>
          ))
        )}

        {documents.length > 0 ? (
          <section className="rounded-xl border bg-card p-5 shadow-sm">
            <h3 className="text-base font-semibold">Generated records</h3>
            <ul className="mt-3 space-y-3">
              {documents.map((doc) => (
                <li key={doc.id} className="flex flex-wrap items-center gap-2 rounded-md border px-3 py-3 text-sm">
                  <span>{new Date(doc.generatedAt).toLocaleString()}</span>
                  <button
                    type="button"
                    className="inline-flex items-center gap-1 rounded-md border px-2 py-1"
                    aria-label="Download CCR JSON"
                    onClick={() => void onDownload(doc, 'json')}
                  >
                    <Download className="h-4 w-4" aria-hidden="true" /> JSON
                  </button>
                  <button
                    type="button"
                    className="inline-flex items-center gap-1 rounded-md border px-2 py-1"
                    aria-label="Download CCR PDF"
                    onClick={() => void onDownload(doc, 'pdf')}
                  >
                    <Download className="h-4 w-4" aria-hidden="true" /> PDF
                  </button>
                  {doc.shareToken ? (
                    <button
                      type="button"
                      className="inline-flex items-center gap-1 rounded-md border px-2 py-1"
                      onClick={() => void copyShareLink(doc)}
                    >
                      <Link2 className="h-4 w-4" aria-hidden="true" />
                      {copiedToken === doc.id ? 'Copied' : 'Copy verify link'}
                    </button>
                  ) : null}
                  {doc.shareToken ? (
                    <Link className="text-primary underline" to={`/verify/${doc.shareToken}`}>
                      Open verification page
                    </Link>
                  ) : null}
                </li>
              ))}
            </ul>
          </section>
        ) : null}
      </div>
    </LmsPage>
  )
}
