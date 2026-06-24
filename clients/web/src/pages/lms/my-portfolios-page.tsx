import { useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { ChevronRight, Eye, EyeOff, FolderOpen, Plus } from 'lucide-react'
import {
  createPortfolio,
  listMyPortfolios,
  patchPortfolio,
  type Portfolio,
} from '../../lib/eportfolio-api'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { LmsPage } from './lms-page'
import { EmptyState } from '../../components/ui/empty-state'
import { IconSwap } from '../../components/ui/icon-swap'

const iconGhostPublished =
  'lex-icon-hit rounded-md text-indigo-600 transition-colors hover:bg-indigo-50/90 hover:text-indigo-700 disabled:cursor-not-allowed disabled:opacity-50 dark:text-indigo-400 dark:hover:bg-indigo-950/45 dark:hover:text-indigo-300'
const iconGhostDraft =
  'lex-icon-hit rounded-md text-slate-400 transition-colors hover:bg-slate-200/45 hover:text-slate-600 disabled:cursor-not-allowed disabled:opacity-50 dark:text-neutral-500 dark:hover:bg-neutral-700/35 dark:hover:text-neutral-300'

export default function MyPortfoliosPage() {
  const { ffEportfolio } = usePlatformFeatures()
  const [portfolios, setPortfolios] = useState<Portfolio[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [showAdd, setShowAdd] = useState(false)
  const [title, setTitle] = useState('')
  const [intro, setIntro] = useState('')
  const [formError, setFormError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      setPortfolios(await listMyPortfolios())
    } catch {
      setError('Failed to load portfolios.')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (!ffEportfolio) return
    void load()
  }, [ffEportfolio, load])

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!title.trim()) {
      setFormError('Please enter a title.')
      return
    }
    setSaving(true)
    setFormError(null)
    try {
      const created = await createPortfolio(title.trim(), intro.trim())
      setPortfolios((prev) => [created, ...prev])
      setShowAdd(false)
      setTitle('')
      setIntro('')
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Failed to create portfolio.')
    } finally {
      setSaving(false)
    }
  }

  const togglePublic = async (p: Portfolio) => {
    try {
      const updated = await patchPortfolio(p.id, { isPublic: !p.isPublic })
      setPortfolios((prev) => prev.map((x) => (x.id === p.id ? updated : x)))
    } catch {
      setError('Failed to update portfolio visibility.')
    }
  }

  if (!ffEportfolio) {
    return (
      <LmsPage title="My Portfolios">
        <p className="text-muted-foreground">
          The ePortfolio feature is not enabled. A global administrator can turn it on in Settings → Global
          platform.
        </p>
      </LmsPage>
    )
  }

  return (
    <LmsPage
      title="My Portfolios"
      description="Curate evidence of your work from any course into a shareable ePortfolio."
      actions={
        <button
          onClick={() => setShowAdd((v) => !v)}
          className="inline-flex items-center gap-2 rounded-xl bg-primary px-4 py-2.5 text-sm font-semibold text-primary-foreground hover:bg-primary/90 shadow-sm transition-[background-color,color,border-color]"
        >
          <Plus className="h-4 w-4" aria-hidden />
          New Portfolio
        </button>
      }
    >
      <div className="space-y-4">
        {showAdd && (
          <form
            onSubmit={(e) => void handleCreate(e)}
            className="space-y-4 rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-neutral-800 dark:bg-neutral-900"
          >
            <h2 className="text-base font-semibold text-slate-900 dark:text-neutral-100">Create a Portfolio</h2>
            {formError && <p className="text-sm text-destructive">{formError}</p>}
            <div>
              <label htmlFor="pf-title" className="mb-1.5 block text-sm font-medium text-slate-700 dark:text-neutral-300">
                Title *
              </label>
              <input
                id="pf-title"
                type="text"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                className="w-full rounded-xl border border-slate-200 bg-white px-3.5 py-2.5 text-sm text-slate-900 focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 focus:outline-none dark:border-neutral-700 dark:bg-neutral-950 dark:text-neutral-100"
                placeholder="e.g. Teaching Capstone Portfolio"
                required
              />
            </div>
            <div>
              <label htmlFor="pf-intro" className="mb-1.5 block text-sm font-medium text-slate-700 dark:text-neutral-300">
                Introduction <span className="text-xs text-muted-foreground">(optional)</span>
              </label>
              <textarea
                id="pf-intro"
                rows={3}
                value={intro}
                onChange={(e) => setIntro(e.target.value)}
                className="w-full rounded-xl border border-slate-200 bg-white px-3.5 py-2.5 text-sm text-slate-900 focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 focus:outline-none dark:border-neutral-700 dark:bg-neutral-950 dark:text-neutral-100"
                placeholder="A short introduction shown at the top of your portfolio."
              />
            </div>
            <div className="flex gap-3">
              <button
                type="submit"
                disabled={saving}
                className="rounded-xl bg-primary px-4 py-2.5 text-sm font-semibold text-primary-foreground hover:bg-primary/90 shadow-sm transition-[background-color,color,border-color] disabled:opacity-50"
              >
                {saving ? 'Creating…' : 'Create'}
              </button>
              <button
                type="button"
                onClick={() => {
                  setShowAdd(false)
                  setFormError(null)
                }}
                className="rounded-xl border border-slate-200 bg-white px-4 py-2.5 text-sm font-medium text-slate-700 hover:bg-slate-50 transition-[background-color,color,border-color] dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-300 dark:hover:bg-neutral-800"
              >
                Cancel
              </button>
            </div>
          </form>
        )}

        {error && <p className="text-sm text-destructive">{error}</p>}
        {loading ? (
          <div className="space-y-2" aria-hidden>
            {[0, 1, 2].map((i) => (
              <div key={i} className="h-16 motion-safe:animate-pulse rounded-2xl border bg-card" />
            ))}
          </div>
        ) : portfolios.length === 0 ? (
          <EmptyState
            icon={FolderOpen}
            title="Start your portfolio"
            body="Collect your best assignments, projects, and reflections to share with employers and graduate programs."
            primaryAction={{
              label: 'New Portfolio',
              onClick: () => setShowAdd(true),
            }}
          />
        ) : (
          <ul className="space-y-3">
            {portfolios.map((p) => (
              <li key={p.id} className="group">
                <Link
                  to={`/portfolios/${p.id}`}
                  className="block w-full rounded-2xl border border-slate-200/70 bg-slate-50/60 p-4 shadow-sm transition-[background-color,color,border-color] hover:border-slate-300/80 hover:bg-slate-100/60 dark:border-neutral-700/80 dark:bg-neutral-800/85 dark:hover:border-neutral-600/80 dark:hover:bg-neutral-800"
                >
                  <div className="flex items-center gap-3">
                    <div className="min-w-0 flex-1">
                      <p className="text-sm font-semibold text-slate-950 dark:text-neutral-100">{p.title}</p>
                      <p className="mt-0.5 truncate text-xs text-slate-500 dark:text-neutral-400">
                        {p.introText
                          ? p.introText
                          : p.isPublic
                            ? 'Published · visible to anyone with the link'
                            : 'Draft · private'}
                      </p>
                    </div>
                    <div className="flex shrink-0 items-center gap-0.5">
                      <button
                        type="button"
                        onClick={(e) => { e.preventDefault(); void togglePublic(p) }}
                        title={
                          p.isPublic
                            ? 'Published — visible to anyone with the link'
                            : 'Draft — private; click to publish'
                        }
                        aria-label={p.isPublic ? 'Published' : 'Draft'}
                        aria-pressed={p.isPublic}
                        className={p.isPublic ? iconGhostPublished : iconGhostDraft}
                      >
                        <IconSwap
                          active={p.isPublic}
                          activeIcon={Eye}
                          inactiveIcon={EyeOff}
                        />
                      </button>
                      <ChevronRight className="h-4 w-4 text-slate-400 dark:text-neutral-500" aria-hidden />
                    </div>
                  </div>
                </Link>
              </li>
            ))}
          </ul>
        )}
      </div>
    </LmsPage>
  )
}
