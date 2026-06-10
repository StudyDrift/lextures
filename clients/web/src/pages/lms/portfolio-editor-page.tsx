import { useCallback, useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import {
  AlertTriangle,
  ArrowDown,
  ArrowLeft,
  ArrowUp,
  Check,
  Copy,
  ExternalLink,
  FileText,
  Plus,
  Trash2,
} from 'lucide-react'
import {
  createArtifact,
  deleteArtifact as apiDeleteArtifact,
  getMyPortfolio,
  patchArtifact,
  patchPortfolio,
  type Artifact,
  type ArtifactType,
  type Evaluation,
  type Portfolio,
} from '../../lib/eportfolio-api'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { LmsPage } from './lms-page'

export default function PortfolioEditorPage() {
  const { pid = '' } = useParams<{ pid: string }>()
  const { ffEportfolio } = usePlatformFeatures()
  const [portfolio, setPortfolio] = useState<Portfolio | null>(null)
  const [artifacts, setArtifacts] = useState<Artifact[]>([])
  const [evaluations, setEvaluations] = useState<Evaluation[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const detail = await getMyPortfolio(pid)
      setPortfolio(detail.portfolio)
      setArtifacts(detail.artifacts)
      setEvaluations(detail.evaluations)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load portfolio.')
    } finally {
      setLoading(false)
    }
  }, [pid])

  useEffect(() => {
    if (!ffEportfolio) return
    void load()
  }, [ffEportfolio, load])

  const evalsByArtifact = (aid: string) => evaluations.filter((e) => e.artifactId === aid)

  const toggleVisibility = async () => {
    if (!portfolio) return
    try {
      const updated = await patchPortfolio(pid, { isPublic: !portfolio.isPublic })
      setPortfolio(updated)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update visibility.')
    }
  }

  const move = async (index: number, dir: -1 | 1) => {
    const next = index + dir
    if (next < 0 || next >= artifacts.length) return
    const reordered = [...artifacts]
    const [item] = reordered.splice(index, 1)
    reordered.splice(next, 0, item)
    setArtifacts(reordered)
    try {
      await patchPortfolio(pid, { order: reordered.map((a) => a.id) })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to reorder.')
      void load()
    }
  }

  const toggleArtifactPublic = async (a: Artifact) => {
    try {
      const updated = await patchArtifact(pid, a.id, { isPublic: !a.isPublic })
      setArtifacts((prev) => prev.map((x) => (x.id === a.id ? updated : x)))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update artifact.')
    }
  }

  const removeArtifact = async (a: Artifact) => {
    if (!window.confirm(`Remove "${a.title}" from this portfolio?`)) return
    try {
      await apiDeleteArtifact(pid, a.id)
      setArtifacts((prev) => prev.filter((x) => x.id !== a.id))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to remove artifact.')
    }
  }

  const onArtifactAdded = (a: Artifact) => setArtifacts((prev) => [...prev, a])

  const publicUrl =
    portfolio?.publicSlug != null ? `${window.location.origin}/p/${portfolio.publicSlug}` : null

  const copyLink = async () => {
    if (!publicUrl) return
    await navigator.clipboard.writeText(publicUrl)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  if (!ffEportfolio) {
    return (
      <LmsPage title="Portfolio">
        <p className="text-muted-foreground">The ePortfolio feature is not enabled.</p>
      </LmsPage>
    )
  }

  if (loading) {
    return (
      <LmsPage title="Portfolio">
        <div className="h-24 motion-safe:animate-pulse rounded-lg border bg-card" aria-hidden />
      </LmsPage>
    )
  }

  if (error && !portfolio) {
    return (
      <LmsPage title="Portfolio">
        <p className="text-sm text-destructive">{error}</p>
        <Link to="/portfolios" className="mt-3 inline-flex items-center gap-1 text-sm text-primary">
          <ArrowLeft className="h-4 w-4" aria-hidden /> Back to portfolios
        </Link>
      </LmsPage>
    )
  }

  if (!portfolio) return null

  return (
    <LmsPage
      title={portfolio.title}
      titleContent={
        <div>
          <Link to="/portfolios" className="mb-1 inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground">
            <ArrowLeft className="h-3.5 w-3.5" aria-hidden /> Portfolios
          </Link>
          <h1 className="text-2xl font-semibold tracking-tight">{portfolio.title}</h1>
          {portfolio.introText && (
            <p className="mt-1 max-w-2xl text-sm text-muted-foreground">{portfolio.introText}</p>
          )}
        </div>
      }
      actions={
        <button
          onClick={() => void toggleVisibility()}
          className="rounded-md border px-3 py-1.5 text-sm font-medium hover:bg-accent"
        >
          {portfolio.isPublic ? 'Make Private' : 'Make Public'}
        </button>
      }
    >
      <div className="space-y-5">
        {error && <p className="text-sm text-destructive">{error}</p>}

        {portfolio.isPublic && (
          <div className="space-y-2 rounded-lg border border-amber-300 bg-amber-50 p-4 dark:border-amber-800 dark:bg-amber-950/40">
            <p className="flex items-start gap-2 text-sm text-amber-900 dark:text-amber-200">
              <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" aria-hidden />
              This portfolio is public. Only artifacts you mark <strong>public</strong> are visible to anyone
              with the link. By sharing graded work you waive FERPA protection for that content.
            </p>
            {publicUrl && (
              <div className="flex items-center gap-2">
                <code className="truncate rounded bg-background px-2 py-1 text-xs">{publicUrl}</code>
                <button
                  onClick={() => void copyLink()}
                  className="inline-flex items-center gap-1 rounded-md border px-2 py-1 text-xs hover:bg-accent"
                >
                  {copied ? <Check className="h-3.5 w-3.5" aria-hidden /> : <Copy className="h-3.5 w-3.5" aria-hidden />}
                  {copied ? 'Copied' : 'Copy link'}
                </button>
              </div>
            )}
          </div>
        )}

        <AddArtifact pid={pid} onAdded={onArtifactAdded} />

        <section aria-label="Artifacts" className="space-y-2">
          <h2 className="text-sm font-medium text-muted-foreground">
            Artifacts ({artifacts.length})
          </h2>
          {artifacts.length === 0 ? (
            <p className="rounded-lg border border-dashed bg-card p-6 text-center text-sm text-muted-foreground">
              No artifacts yet. Add a text page or link above, or use “Add to Portfolio” from one of your
              graded submissions.
            </p>
          ) : (
            <ul className="space-y-2">
              {artifacts.map((a, i) => (
                <li
                  key={a.id}
                  className="rounded-lg border bg-card p-4"
                >
                  <div className="flex items-start justify-between gap-3">
                    <div className="min-w-0">
                      <p className="flex items-center gap-1.5 font-medium">
                        <FileText className="h-4 w-4 text-muted-foreground" aria-hidden />
                        {a.title}
                      </p>
                      {a.description && (
                        <p className="mt-0.5 text-sm text-muted-foreground">{a.description}</p>
                      )}
                      <p className="mt-1 text-xs uppercase tracking-wide text-muted-foreground">
                        {a.artifactType.replace('_', ' ')}
                        {a.fileName ? ` · ${a.fileName}` : ''}
                      </p>
                      {a.externalUrl && (
                        <a
                          href={a.externalUrl}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="mt-1 inline-flex items-center gap-1 text-xs text-primary"
                        >
                          <ExternalLink className="h-3 w-3" aria-hidden /> {a.externalUrl}
                        </a>
                      )}
                      {a.outcomeIds.length > 0 && (
                        <p className="mt-1 text-xs text-muted-foreground">
                          {a.outcomeIds.length} outcome{a.outcomeIds.length === 1 ? '' : 's'} tagged
                        </p>
                      )}
                      {evalsByArtifact(a.id).map((ev) => (
                        <div
                          key={ev.id}
                          className="mt-2 rounded-md border border-emerald-200 bg-emerald-50 p-2 text-xs dark:border-emerald-900 dark:bg-emerald-950/40"
                        >
                          <span className="font-medium">Reviewer feedback</span>
                          {ev.totalScore != null && <span> · Score: {ev.totalScore}</span>}
                          {ev.feedback && <p className="mt-0.5 text-muted-foreground">{ev.feedback}</p>}
                        </div>
                      ))}
                    </div>
                    <div className="flex shrink-0 flex-col items-end gap-1">
                      <div className="flex gap-1">
                        <button
                          onClick={() => void move(i, -1)}
                          disabled={i === 0}
                          aria-label={`Move ${a.title} up`}
                          className="rounded border p-1 hover:bg-accent disabled:opacity-30"
                        >
                          <ArrowUp className="h-3.5 w-3.5" aria-hidden />
                        </button>
                        <button
                          onClick={() => void move(i, 1)}
                          disabled={i === artifacts.length - 1}
                          aria-label={`Move ${a.title} down`}
                          className="rounded border p-1 hover:bg-accent disabled:opacity-30"
                        >
                          <ArrowDown className="h-3.5 w-3.5" aria-hidden />
                        </button>
                        <button
                          onClick={() => void removeArtifact(a)}
                          aria-label={`Remove ${a.title}`}
                          className="rounded border p-1 text-destructive hover:bg-destructive/10"
                        >
                          <Trash2 className="h-3.5 w-3.5" aria-hidden />
                        </button>
                      </div>
                      <label className="flex items-center gap-1.5 text-xs">
                        <input
                          type="checkbox"
                          checked={a.isPublic}
                          onChange={() => void toggleArtifactPublic(a)}
                        />
                        Public
                      </label>
                    </div>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </section>
      </div>
    </LmsPage>
  )
}

function AddArtifact({ pid, onAdded }: { pid: string; onAdded: (a: Artifact) => void }) {
  const [open, setOpen] = useState(false)
  const [type, setType] = useState<ArtifactType>('text_page')
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [textContent, setTextContent] = useState('')
  const [externalUrl, setExternalUrl] = useState('')
  const [saving, setSaving] = useState(false)
  const [err, setErr] = useState<string | null>(null)

  const reset = () => {
    setTitle('')
    setDescription('')
    setTextContent('')
    setExternalUrl('')
    setErr(null)
  }

  const submit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!title.trim()) {
      setErr('Title is required.')
      return
    }
    if (type === 'url' && !externalUrl.trim()) {
      setErr('A URL is required for link artifacts.')
      return
    }
    setSaving(true)
    setErr(null)
    try {
      const created = await createArtifact(pid, {
        artifactType: type,
        title: title.trim(),
        description: description.trim() || undefined,
        textContent: type === 'text_page' ? textContent : undefined,
        externalUrl: type === 'url' ? externalUrl.trim() : undefined,
      })
      onAdded(created)
      reset()
      setOpen(false)
    } catch (e2) {
      setErr(e2 instanceof Error ? e2.message : 'Failed to add artifact.')
    } finally {
      setSaving(false)
    }
  }

  if (!open) {
    return (
      <button
        onClick={() => setOpen(true)}
        className="inline-flex items-center gap-1.5 rounded-md bg-primary px-3 py-1.5 text-sm font-medium text-primary-foreground hover:bg-primary/90"
      >
        <Plus className="h-4 w-4" aria-hidden />
        Add Artifact
      </button>
    )
  }

  return (
    <form onSubmit={(e) => void submit(e)} className="space-y-3 rounded-lg border bg-card p-4">
      <h2 className="font-medium">Add an Artifact</h2>
      {err && <p className="text-sm text-destructive">{err}</p>}
      <div className="grid gap-3 sm:grid-cols-2">
        <div>
          <label htmlFor="art-type" className="mb-1 block text-sm font-medium">
            Type
          </label>
          <select
            id="art-type"
            value={type}
            onChange={(e) => setType(e.target.value as ArtifactType)}
            className="w-full rounded-md border bg-background px-3 py-1.5 text-sm"
          >
            <option value="text_page">Text page</option>
            <option value="url">External link</option>
          </select>
        </div>
        <div>
          <label htmlFor="art-title" className="mb-1 block text-sm font-medium">
            Title *
          </label>
          <input
            id="art-title"
            type="text"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            className="w-full rounded-md border bg-background px-3 py-1.5 text-sm"
            required
          />
        </div>
      </div>
      <div>
        <label htmlFor="art-desc" className="mb-1 block text-sm font-medium">
          Description
        </label>
        <input
          id="art-desc"
          type="text"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          className="w-full rounded-md border bg-background px-3 py-1.5 text-sm"
        />
      </div>
      {type === 'text_page' && (
        <div>
          <label htmlFor="art-text" className="mb-1 block text-sm font-medium">
            Content
          </label>
          <textarea
            id="art-text"
            rows={4}
            value={textContent}
            onChange={(e) => setTextContent(e.target.value)}
            className="w-full rounded-md border bg-background px-3 py-1.5 text-sm"
          />
        </div>
      )}
      {type === 'url' && (
        <div>
          <label htmlFor="art-url" className="mb-1 block text-sm font-medium">
            URL *
          </label>
          <input
            id="art-url"
            type="url"
            value={externalUrl}
            onChange={(e) => setExternalUrl(e.target.value)}
            className="w-full rounded-md border bg-background px-3 py-1.5 text-sm"
            placeholder="https://"
          />
        </div>
      )}
      <div className="flex gap-2">
        <button
          type="submit"
          disabled={saving}
          className="rounded-md bg-primary px-3 py-1.5 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
        >
          {saving ? 'Adding…' : 'Add'}
        </button>
        <button
          type="button"
          onClick={() => {
            setOpen(false)
            reset()
          }}
          className="rounded-md border px-3 py-1.5 text-sm"
        >
          Cancel
        </button>
      </div>
    </form>
  )
}
