import { useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { FolderOpen, Globe, Lock, Plus } from 'lucide-react'
import {
  createPortfolio,
  listMyPortfolios,
  type Portfolio,
} from '../../lib/eportfolio-api'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { LmsPage } from './lms-page'

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

  if (!ffEportfolio) {
    return (
      <LmsPage title="My Portfolios">
        <p className="text-muted-foreground">The ePortfolio feature is not enabled.</p>
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
          className="inline-flex items-center gap-1.5 rounded-md bg-primary px-3 py-1.5 text-sm font-medium text-primary-foreground hover:bg-primary/90"
        >
          <Plus className="h-4 w-4" aria-hidden />
          New Portfolio
        </button>
      }
    >
      <div className="space-y-4">
        {showAdd && (
          <form onSubmit={(e) => void handleCreate(e)} className="space-y-3 rounded-lg border bg-card p-4">
            <h2 className="font-medium">Create a Portfolio</h2>
            {formError && <p className="text-sm text-destructive">{formError}</p>}
            <div>
              <label htmlFor="pf-title" className="mb-1 block text-sm font-medium">
                Title *
              </label>
              <input
                id="pf-title"
                type="text"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                className="w-full rounded-md border bg-background px-3 py-1.5 text-sm"
                placeholder="e.g. Teaching Capstone Portfolio"
                required
              />
            </div>
            <div>
              <label htmlFor="pf-intro" className="mb-1 block text-sm font-medium">
                Introduction <span className="text-xs text-muted-foreground">(optional)</span>
              </label>
              <textarea
                id="pf-intro"
                rows={3}
                value={intro}
                onChange={(e) => setIntro(e.target.value)}
                className="w-full rounded-md border bg-background px-3 py-1.5 text-sm"
                placeholder="A short introduction shown at the top of your portfolio."
              />
            </div>
            <div className="flex gap-2">
              <button
                type="submit"
                disabled={saving}
                className="rounded-md bg-primary px-3 py-1.5 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
              >
                {saving ? 'Creating…' : 'Create'}
              </button>
              <button
                type="button"
                onClick={() => {
                  setShowAdd(false)
                  setFormError(null)
                }}
                className="rounded-md border px-3 py-1.5 text-sm"
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
              <div key={i} className="h-16 motion-safe:animate-pulse rounded-lg border bg-card" />
            ))}
          </div>
        ) : portfolios.length === 0 ? (
          <div className="rounded-lg border border-dashed bg-card p-8 text-center">
            <FolderOpen className="mx-auto h-8 w-8 text-muted-foreground" aria-hidden />
            <p className="mt-2 font-medium">Start your portfolio</p>
            <p className="mt-1 text-sm text-muted-foreground">
              Collect your best assignments, projects, and reflections to share with employers and graduate
              programs.
            </p>
            <button
              onClick={() => setShowAdd(true)}
              className="mt-4 inline-flex items-center gap-1.5 rounded-md bg-primary px-3 py-1.5 text-sm font-medium text-primary-foreground hover:bg-primary/90"
            >
              <Plus className="h-4 w-4" aria-hidden />
              New Portfolio
            </button>
          </div>
        ) : (
          <ul className="space-y-2">
            {portfolios.map((p) => (
              <li key={p.id}>
                <Link
                  to={`/portfolios/${p.id}`}
                  className="flex items-center justify-between rounded-lg border bg-card p-4 hover:border-primary/50"
                >
                  <div className="min-w-0">
                    <p className="truncate font-medium">{p.title}</p>
                    {p.introText && (
                      <p className="truncate text-sm text-muted-foreground">{p.introText}</p>
                    )}
                  </div>
                  <span
                    className="ms-3 inline-flex shrink-0 items-center gap-1 text-xs text-muted-foreground"
                    title={p.isPublic ? 'Public' : 'Private'}
                  >
                    {p.isPublic ? (
                      <>
                        <Globe className="h-3.5 w-3.5" aria-hidden /> Public
                      </>
                    ) : (
                      <>
                        <Lock className="h-3.5 w-3.5" aria-hidden /> Private
                      </>
                    )}
                  </span>
                </Link>
              </li>
            ))}
          </ul>
        )}
      </div>
    </LmsPage>
  )
}
