import { useCallback, useEffect, useId, useRef, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import {
  AlertTriangle,
  ArrowDown,
  ArrowLeft,
  ArrowUp,
  Check,
  ChevronDown,
  Copy,
  ExternalLink,
  Eye,
  EyeOff,
  FileText,
  Heading,
  Link2,
  MoreVertical,
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
import { EmptyState } from '../../components/ui/empty-state'

type PortfolioArtifactKind = 'heading' | 'content_page' | 'url'

const iconGhostPublished =
  'rounded-md p-2 text-indigo-600 transition hover:bg-indigo-50/90 hover:text-indigo-700 disabled:cursor-not-allowed disabled:opacity-50 dark:text-indigo-400 dark:hover:bg-indigo-950/45 dark:hover:text-indigo-300'
const iconGhostDraft =
  'rounded-md p-2 text-slate-400 transition hover:bg-slate-200/45 hover:text-slate-600 disabled:cursor-not-allowed disabled:opacity-50 dark:text-neutral-500 dark:hover:bg-neutral-700/35 dark:hover:text-neutral-300'
const iconGhost =
  'rounded-md p-2 text-slate-500 transition hover:bg-slate-200/45 hover:text-slate-800 disabled:cursor-not-allowed disabled:opacity-50 dark:text-neutral-400 dark:hover:bg-neutral-700/35 dark:hover:text-neutral-200'

function ArtifactTypeIcon({ type }: { type: ArtifactType }) {
  if (type === 'url') {
    return (
      <span
        className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl border border-sky-200/90 bg-sky-50 text-sky-700 dark:border-sky-500/40 dark:bg-sky-950/55 dark:text-sky-200"
        aria-hidden
      >
        <Link2 className="h-4 w-4" strokeWidth={2} />
      </span>
    )
  }
  return (
    <span
      className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl border border-indigo-200/80 bg-indigo-50 text-indigo-600 dark:border-indigo-500/35 dark:bg-indigo-950/60 dark:text-indigo-300"
      aria-hidden
    >
      <FileText className="h-4 w-4" strokeWidth={2} />
    </span>
  )
}

function ArtifactItemActions({
  artifact,
  index,
  total,
  onTogglePublished,
  onMoveUp,
  onMoveDown,
  onDelete,
}: {
  artifact: Artifact
  index: number
  total: number
  onTogglePublished: () => void
  onMoveUp: () => void
  onMoveDown: () => void
  onDelete: () => void
}) {
  const [menuOpen, setMenuOpen] = useState(false)
  const rootRef = useRef<HTMLDivElement>(null)
  const menuId = useId()

  useEffect(() => {
    if (!menuOpen) return
    function onDoc(e: MouseEvent) {
      if (!rootRef.current?.contains(e.target as Node)) setMenuOpen(false)
    }
    document.addEventListener('mousedown', onDoc)
    return () => document.removeEventListener('mousedown', onDoc)
  }, [menuOpen])

  return (
    <div className="flex shrink-0 items-center gap-0.5">
      <button
        type="button"
        onClick={onTogglePublished}
        title={
          artifact.isPublic
            ? 'Published — visible in shared portfolio'
            : 'Draft — hidden from viewers; click to publish'
        }
        aria-label={artifact.isPublic ? 'Published' : 'Draft'}
        aria-pressed={artifact.isPublic}
        className={artifact.isPublic ? iconGhostPublished : iconGhostDraft}
      >
        {artifact.isPublic ? (
          <Eye className="h-4 w-4" strokeWidth={2} aria-hidden />
        ) : (
          <EyeOff className="h-4 w-4" strokeWidth={2} aria-hidden />
        )}
      </button>
      <div ref={rootRef} className="relative">
        <button
          type="button"
          aria-haspopup="menu"
          aria-expanded={menuOpen}
          aria-controls={menuOpen ? menuId : undefined}
          onClick={() => setMenuOpen((o) => !o)}
          title="Artifact actions"
          className={iconGhost}
        >
          <MoreVertical className="h-4 w-4" strokeWidth={2} aria-hidden />
        </button>
        {menuOpen && (
          <div
            id={menuId}
            role="menu"
            aria-label="Artifact actions"
            className="absolute end-0 z-50 mt-1 min-w-[10rem] overflow-hidden rounded-xl border border-slate-200 bg-white py-1 shadow-lg shadow-slate-900/10 dark:border-neutral-600 dark:bg-neutral-800 dark:shadow-black/40"
          >
            <button
              type="button"
              role="menuitem"
              disabled={index === 0}
              onClick={() => { onMoveUp(); setMenuOpen(false) }}
              className="flex w-full items-center gap-2 px-2.5 py-2 text-start text-sm font-medium text-slate-800 transition hover:bg-slate-50 disabled:opacity-40 dark:text-neutral-100 dark:hover:bg-neutral-700/80"
            >
              <ArrowUp className="h-4 w-4" aria-hidden /> Move up
            </button>
            <button
              type="button"
              role="menuitem"
              disabled={index === total - 1}
              onClick={() => { onMoveDown(); setMenuOpen(false) }}
              className="flex w-full items-center gap-2 px-2.5 py-2 text-start text-sm font-medium text-slate-800 transition hover:bg-slate-50 disabled:opacity-40 dark:text-neutral-100 dark:hover:bg-neutral-700/80"
            >
              <ArrowDown className="h-4 w-4" aria-hidden /> Move down
            </button>
            <button
              type="button"
              role="menuitem"
              onClick={() => { onDelete(); setMenuOpen(false) }}
              className="flex w-full items-center gap-2 border-t border-slate-100 px-2.5 py-2 text-start text-sm font-medium text-rose-700 transition hover:bg-rose-50 dark:border-neutral-700 dark:text-rose-300 dark:hover:bg-rose-950/50"
            >
              <Trash2 className="h-4 w-4" aria-hidden /> Delete
            </button>
          </div>
        )}
      </div>
    </div>
  )
}

function AddArtifactMenu({ onSelect }: { onSelect: (kind: PortfolioArtifactKind) => void }) {
  const [open, setOpen] = useState(false)
  const rootRef = useRef<HTMLDivElement>(null)
  const menuId = useId()

  useEffect(() => {
    if (!open) return
    function onDoc(e: MouseEvent) {
      if (!rootRef.current?.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', onDoc)
    return () => document.removeEventListener('mousedown', onDoc)
  }, [open])

  function pick(kind: PortfolioArtifactKind) {
    onSelect(kind)
    setOpen(false)
  }

  return (
    <div ref={rootRef} className="relative inline-block shrink-0 text-start">
      <button
        type="button"
        aria-haspopup="menu"
        aria-expanded={open}
        aria-controls={open ? menuId : undefined}
        onClick={() => setOpen((o) => !o)}
        className="inline-flex items-center gap-1.5 rounded-lg border border-slate-200/70 bg-white/90 px-2 py-1.5 text-xs font-medium text-slate-700 shadow-none transition hover:border-slate-300/80 hover:bg-slate-50/90 sm:px-2.5 sm:text-sm dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-200 dark:hover:border-neutral-500 dark:hover:bg-neutral-800"
      >
        <Plus className="h-4 w-4 shrink-0" aria-hidden />
        <span className="truncate">Add artifact</span>
        <ChevronDown
          className={`h-4 w-4 shrink-0 transition ${open ? 'rotate-180' : ''}`}
          aria-hidden
        />
      </button>

      {open && (
        <div
          id={menuId}
          role="menu"
          aria-label="Artifact types"
          className="absolute end-0 z-50 mt-1 w-max min-w-[min(22rem,calc(100vw-1.5rem))] max-w-[calc(100vw-1.5rem)] overflow-hidden rounded-xl border border-slate-200 bg-white py-1 shadow-lg shadow-slate-900/10 dark:border-neutral-600 dark:bg-neutral-800 dark:shadow-black/40"
        >
          <button
            type="button"
            role="menuitem"
            onClick={() => pick('heading')}
            className="flex w-full items-start gap-3 px-2.5 py-2 text-start text-sm transition hover:bg-slate-50 dark:hover:bg-neutral-700"
          >
            <span className="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-slate-200 bg-white text-slate-500 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-400">
              <Heading className="h-4 w-4" aria-hidden />
            </span>
            <span className="flex min-w-0 flex-col gap-0.5">
              <span className="font-semibold text-slate-950 dark:text-neutral-100">Heading</span>
              <span className="text-xs text-slate-500 dark:text-neutral-400">Text label for organizing content</span>
            </span>
          </button>
          <button
            type="button"
            role="menuitem"
            onClick={() => pick('content_page')}
            className="flex w-full items-start gap-3 border-t border-slate-100 px-2.5 py-2 text-start text-sm transition hover:bg-slate-50 dark:border-neutral-700 dark:hover:bg-neutral-700"
          >
            <span className="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-indigo-200/80 bg-indigo-50 text-indigo-600 dark:border-indigo-500/35 dark:bg-indigo-950 dark:text-indigo-300">
              <FileText className="h-4 w-4" aria-hidden />
            </span>
            <span className="flex min-w-0 flex-col gap-0.5">
              <span className="font-semibold text-slate-950 dark:text-neutral-100">Content page</span>
              <span className="text-xs text-slate-500 dark:text-neutral-400">Markdown page with rich formatting</span>
            </span>
          </button>
          <button
            type="button"
            role="menuitem"
            onClick={() => pick('url')}
            className="flex w-full items-start gap-3 border-t border-slate-100 px-2.5 py-2 text-start text-sm transition hover:bg-slate-50 dark:border-neutral-700 dark:hover:bg-neutral-700"
          >
            <span className="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-sky-200/90 bg-sky-50 text-sky-700 dark:border-sky-500/40 dark:bg-sky-950 dark:text-sky-200">
              <ExternalLink className="h-4 w-4" aria-hidden />
            </span>
            <span className="flex min-w-0 flex-col gap-0.5">
              <span className="font-semibold text-slate-950 dark:text-neutral-100">External link</span>
              <span className="text-xs text-slate-500 dark:text-neutral-400">Opens a URL in a new tab</span>
            </span>
          </button>
        </div>
      )}
    </div>
  )
}

function AddArtifactForm({
  pid,
  kind,
  onAdded,
  onCancel,
}: {
  pid: string
  kind: PortfolioArtifactKind
  onAdded: (a: Artifact) => void
  onCancel: () => void
}) {
  const [title, setTitle] = useState('')
  const [textContent, setTextContent] = useState('')
  const [externalUrl, setExternalUrl] = useState('')
  const [saving, setSaving] = useState(false)
  const [err, setErr] = useState<string | null>(null)

  const kindLabel =
    kind === 'heading' ? 'Heading' : kind === 'content_page' ? 'Content Page' : 'External Link'

  const submit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!title.trim()) { setErr('Title is required.'); return }
    if (kind === 'url' && !externalUrl.trim()) { setErr('A URL is required.'); return }
    setSaving(true)
    setErr(null)
    try {
      const artifactType: ArtifactType = kind === 'url' ? 'url' : 'text_page'
      const created = await createArtifact(pid, {
        artifactType,
        title: title.trim(),
        textContent: kind === 'content_page' ? textContent : undefined,
        externalUrl: kind === 'url' ? externalUrl.trim() : undefined,
      })
      onAdded(created)
    } catch (e2) {
      setErr(e2 instanceof Error ? e2.message : 'Failed to add artifact.')
      setSaving(false)
    }
  }

  return (
    <form
      onSubmit={(e) => void submit(e)}
      className="mt-4 space-y-4 border-t border-slate-200/55 pt-4 dark:border-neutral-700/80"
    >
      <h3 className="text-sm font-semibold text-slate-900 dark:text-neutral-100">New {kindLabel}</h3>
      {err && <p className="text-sm text-destructive">{err}</p>}
      <div>
        <label htmlFor="art-title" className="mb-1 block text-sm font-medium text-slate-700 dark:text-neutral-300">
          Title *
        </label>
        <input
          id="art-title"
          type="text"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          autoFocus
          className="w-full rounded-xl border border-slate-200 bg-white px-3.5 py-2.5 text-sm text-slate-900 focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 focus:outline-none dark:border-neutral-700 dark:bg-neutral-950 dark:text-neutral-100"
          required
        />
      </div>
      {kind === 'content_page' && (
        <div>
          <label htmlFor="art-text" className="mb-1 block text-sm font-medium text-slate-700 dark:text-neutral-300">
            Content
          </label>
          <textarea
            id="art-text"
            rows={4}
            value={textContent}
            onChange={(e) => setTextContent(e.target.value)}
            className="w-full rounded-xl border border-slate-200 bg-white px-3.5 py-2.5 text-sm text-slate-900 focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 focus:outline-none dark:border-neutral-700 dark:bg-neutral-950 dark:text-neutral-100"
          />
        </div>
      )}
      {kind === 'url' && (
        <div>
          <label htmlFor="art-url" className="mb-1 block text-sm font-medium text-slate-700 dark:text-neutral-300">
            URL *
          </label>
          <input
            id="art-url"
            type="url"
            value={externalUrl}
            onChange={(e) => setExternalUrl(e.target.value)}
            className="w-full rounded-xl border border-slate-200 bg-white px-3.5 py-2.5 text-sm text-slate-900 focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 focus:outline-none dark:border-neutral-700 dark:bg-neutral-950 dark:text-neutral-100"
            placeholder="https://"
          />
        </div>
      )}
      <div className="flex gap-3">
        <button
          type="submit"
          disabled={saving}
          className="rounded-xl bg-primary px-4 py-2.5 text-sm font-semibold text-primary-foreground hover:bg-primary/90 shadow-sm transition disabled:opacity-50"
        >
          {saving ? 'Adding…' : 'Add'}
        </button>
        <button
          type="button"
          onClick={onCancel}
          className="rounded-xl border border-slate-200 bg-white px-4 py-2.5 text-sm font-medium text-slate-700 hover:bg-slate-50 transition dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-300 dark:hover:bg-neutral-800"
        >
          Cancel
        </button>
      </div>
    </form>
  )
}

function PortfolioPublishButton({
  portfolio,
  onToggle,
}: {
  portfolio: Portfolio
  onToggle: () => void
}) {
  return (
    <button
      type="button"
      onClick={onToggle}
      title={
        portfolio.isPublic
          ? 'Published — visible to anyone with the link'
          : 'Draft — private; click to publish'
      }
      aria-label={portfolio.isPublic ? 'Published' : 'Draft'}
      aria-pressed={portfolio.isPublic}
      className={`inline-flex items-center gap-2 rounded-xl border px-4 py-2.5 text-sm font-semibold shadow-sm transition ${
        portfolio.isPublic
          ? 'border-indigo-200 bg-indigo-50 text-indigo-700 hover:bg-indigo-100 dark:border-indigo-900/40 dark:bg-indigo-950/30 dark:text-indigo-300 dark:hover:bg-indigo-900/40'
          : 'border-slate-200 bg-white text-slate-700 hover:bg-slate-50 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-300 dark:hover:bg-neutral-800'
      }`}
    >
      {portfolio.isPublic ? (
        <>
          <Eye className="h-4 w-4" aria-hidden /> Published
        </>
      ) : (
        <>
          <EyeOff className="h-4 w-4" aria-hidden /> Draft
        </>
      )}
    </button>
  )
}

export default function PortfolioEditorPage() {
  const { pid = '' } = useParams<{ pid: string }>()
  const { ffEportfolio } = usePlatformFeatures()
  const [portfolio, setPortfolio] = useState<Portfolio | null>(null)
  const [artifacts, setArtifacts] = useState<Artifact[]>([])
  const [evaluations, setEvaluations] = useState<Evaluation[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)
  const [addKind, setAddKind] = useState<PortfolioArtifactKind | null>(null)

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

  const onArtifactAdded = (a: Artifact) => {
    setArtifacts((prev) => [...prev, a])
    setAddKind(null)
  }

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
        <p className="text-muted-foreground">
          The ePortfolio feature is not enabled. A global administrator can turn it on in Settings → Global
          platform.
        </p>
      </LmsPage>
    )
  }

  if (loading) {
    return (
      <LmsPage title="Portfolio">
        <div className="h-24 motion-safe:animate-pulse rounded-2xl border bg-card" aria-hidden />
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
        <PortfolioPublishButton portfolio={portfolio} onToggle={() => void toggleVisibility()} />
      }
    >
      <div className="space-y-5">
        {error && <p className="text-sm text-destructive">{error}</p>}

        {portfolio.isPublic && (
          <div className="space-y-3 rounded-2xl border border-amber-200 bg-amber-50/80 p-5 shadow-sm dark:border-amber-900/40 dark:bg-amber-950/30">
            <div className="flex items-start gap-2.5 text-sm text-amber-900 dark:text-amber-200">
              <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-amber-600 dark:text-amber-400" aria-hidden />
              <div>
                <span className="font-semibold">Published Portfolio:</span> Only artifacts you mark{' '}
                <strong>published</strong> are visible to anyone with the link. By sharing graded work you waive
                FERPA protection for that content.
              </div>
            </div>
            {publicUrl && (
              <div className="flex flex-wrap items-center gap-2 border-t border-amber-200/50 pt-3 dark:border-amber-900/30">
                <code className="truncate rounded-lg border border-slate-100 bg-white px-3 py-1.5 font-mono text-xs text-slate-750 dark:border-neutral-800 dark:bg-neutral-900 dark:text-neutral-300">
                  {publicUrl}
                </code>
                <button
                  onClick={() => void copyLink()}
                  className="inline-flex items-center gap-1.5 rounded-lg border border-slate-200 bg-white px-3 py-1.5 text-xs font-medium text-slate-700 transition hover:bg-slate-50 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-300 dark:hover:bg-neutral-800"
                >
                  {copied ? (
                    <Check className="h-3.5 w-3.5 text-emerald-600" aria-hidden />
                  ) : (
                    <Copy className="h-3.5 w-3.5 text-slate-500" aria-hidden />
                  )}
                  {copied ? 'Copied' : 'Copy link'}
                </button>
              </div>
            )}
          </div>
        )}

        {/* Artifacts — styled as a module card */}
        <div className="rounded-2xl border border-slate-200/70 bg-slate-50/60 p-4 shadow-sm dark:border-neutral-700/80 dark:bg-neutral-800/85">
          <div className="flex items-center justify-between gap-3">
            <div>
              <p className="text-sm font-semibold text-slate-950 dark:text-neutral-100">Artifacts</p>
              <p className="mt-0.5 text-xs text-slate-500 dark:text-neutral-400">
                {artifacts.length === 0
                  ? 'Add text pages, links, or headings to this portfolio.'
                  : `${artifacts.length} ${artifacts.length === 1 ? 'item' : 'items'}`}
              </p>
            </div>
            <AddArtifactMenu onSelect={(kind) => setAddKind(kind)} />
          </div>

          {artifacts.length > 0 && (
            <ul className="mt-4 divide-y divide-slate-200/55 border-t border-slate-200/55 dark:divide-neutral-700/80 dark:border-neutral-700/80">
              {artifacts.map((a, i) => (
                <li key={a.id} className="py-3">
                  <div className="flex items-center gap-2">
                    <div className="flex min-w-0 flex-1 items-center gap-3">
                      <ArtifactTypeIcon type={a.artifactType} />
                      <div className="min-w-0 flex-1">
                        <div className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1">
                          <p className="text-base font-semibold leading-snug tracking-tight text-slate-900 dark:text-neutral-100">
                            {a.title}
                          </p>
                          <p className="inline-flex shrink-0 items-center rounded bg-slate-100 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wider text-slate-600 dark:bg-neutral-750 dark:text-neutral-400">
                            {a.artifactType.replace('_', ' ')}
                            {a.fileName ? ` · ${a.fileName}` : ''}
                          </p>
                        </div>
                        {a.description && (
                          <p className="mt-0.5 text-sm text-slate-500 dark:text-neutral-400">{a.description}</p>
                        )}
                        {a.externalUrl && (
                          <div className="mt-1">
                            <a
                              href={a.externalUrl}
                              target="_blank"
                              rel="noopener noreferrer"
                              className="inline-flex items-center gap-1 text-xs font-semibold text-primary hover:underline"
                            >
                              <ExternalLink className="h-3 w-3" aria-hidden /> {a.externalUrl}
                            </a>
                          </div>
                        )}
                        {a.outcomeIds.length > 0 && (
                          <p className="mt-1 text-xs font-medium text-slate-500 dark:text-neutral-400">
                            {a.outcomeIds.length} outcome{a.outcomeIds.length === 1 ? '' : 's'} tagged
                          </p>
                        )}
                        {evalsByArtifact(a.id).map((ev) => (
                          <div
                            key={ev.id}
                            className="mt-3 rounded-xl border border-emerald-200 bg-emerald-50/70 p-3 text-xs dark:border-emerald-900 dark:bg-emerald-950/20"
                          >
                            <span className="font-semibold text-emerald-800 dark:text-emerald-300">Reviewer feedback</span>
                            {ev.totalScore != null && (
                              <span className="font-semibold text-emerald-800 dark:text-emerald-300"> · Score: {ev.totalScore}</span>
                            )}
                            {ev.feedback && (
                              <p className="mt-1 leading-relaxed text-emerald-900 dark:text-emerald-450">{ev.feedback}</p>
                            )}
                          </div>
                        ))}
                      </div>
                    </div>
                    <ArtifactItemActions
                      artifact={a}
                      index={i}
                      total={artifacts.length}
                      onTogglePublished={() => void toggleArtifactPublic(a)}
                      onMoveUp={() => void move(i, -1)}
                      onMoveDown={() => void move(i, 1)}
                      onDelete={() => void removeArtifact(a)}
                    />
                  </div>
                </li>
              ))}
            </ul>
          )}

          {artifacts.length === 0 && !addKind && (
            <div className="mt-4 border-t border-slate-200/55 pt-4 dark:border-neutral-700/80">
              <EmptyState
                icon={FileText}
                title="No artifacts yet"
                body='Use "Add artifact" above, or use "Add to Portfolio" from one of your graded submissions.'
              />
            </div>
          )}

          {addKind && (
            <AddArtifactForm
              pid={pid}
              kind={addKind}
              onAdded={onArtifactAdded}
              onCancel={() => setAddKind(null)}
            />
          )}
        </div>
      </div>
    </LmsPage>
  )
}
