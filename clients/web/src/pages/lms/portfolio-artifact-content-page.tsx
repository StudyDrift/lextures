import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { Pencil } from 'lucide-react'
import { SyllabusBlockEditor } from '../../components/syllabus/syllabus-block-editor'
import { MarkdownArticleView } from '../../components/syllabus/syllabus-markdown-view'
import { markdownToSectionsForEditor, sectionsToMarkdown } from '../../components/syllabus/syllabus-section-markdown'
import {
  getMyPortfolioArtifact,
  isPortfolioContentPage,
  patchArtifact,
} from '../../lib/eportfolio-api'
import {
  type ResolvedMarkdownTheme,
  resolveMarkdownTheme,
} from '../../lib/markdown-theme'
import { useLmsDarkMode } from '../../hooks/use-lms-dark-mode'
import { AuthoringSaveFootprint } from '../../components/authoring-save-footprint'
import { ReadingFocusToggle } from '../../components/layout/reading-focus-toggle'
import { formatAbsolute } from '../../lib/format-datetime'
import { toastMutationError, toastSaveOk } from '../../lib/lms-toast'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { LmsPage } from './lms-page'

function newLocalId(): string {
  if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) {
    return crypto.randomUUID()
  }
  return `local-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`
}

export default function PortfolioArtifactContentPage() {
  const { pid = '', aid = '' } = useParams<{ pid: string; aid: string }>()
  const { ffEportfolio } = usePlatformFeatures()

  const [title, setTitle] = useState('')
  const [markdown, setMarkdown] = useState('')
  const [updatedAt, setUpdatedAt] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState<string | null>(null)

  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState<ReturnType<typeof markdownToSectionsForEditor>>([])
  const [saveError, setSaveError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const [lastLocalAuthoringSave, setLastLocalAuthoringSave] = useState<string | null>(null)

  const lmsUiDark = useLmsDarkMode()
  const mdTheme = useMemo(
    (): ResolvedMarkdownTheme => resolveMarkdownTheme('classic', null, { lmsUiDark }),
    [lmsUiDark],
  )

  const load = useCallback(async () => {
    if (!pid || !aid) return
    setLoading(true)
    setLoadError(null)
    try {
      const artifact = await getMyPortfolioArtifact(pid, aid)
      if (!isPortfolioContentPage(artifact)) {
        throw new Error('This artifact is not a content page.')
      }
      setTitle(artifact.title)
      setMarkdown(artifact.textContent)
      setUpdatedAt(artifact.updatedAt)
      if (!artifact.textContent.trim()) {
        setDraft(markdownToSectionsForEditor('', newLocalId))
        setEditing(true)
      }
    } catch (e) {
      setLoadError(e instanceof Error ? e.message : 'Could not load this page.')
      setTitle('')
      setMarkdown('')
      setUpdatedAt(null)
    } finally {
      setLoading(false)
    }
  }, [pid, aid])

  useEffect(() => {
    if (!ffEportfolio) return
    void load()
  }, [ffEportfolio, load])

  function beginEdit() {
    setSaveError(null)
    setDraft(markdownToSectionsForEditor(markdown, newLocalId))
    setEditing(true)
  }

  function cancelEdit() {
    setSaveError(null)
    setEditing(false)
    setDraft([])
  }

  async function save() {
    if (!pid || !aid) return
    const body = sectionsToMarkdown(draft)
    setSaveError(null)
    setSaving(true)
    try {
      const updated = await patchArtifact(pid, aid, { textContent: body })
      setMarkdown(updated.textContent)
      setUpdatedAt(updated.updatedAt)
      setLastLocalAuthoringSave(new Date().toISOString())
      setEditing(false)
      setDraft([])
      toastSaveOk('Page saved')
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Could not save.'
      setSaveError(msg)
      toastMutationError(msg)
    } finally {
      setSaving(false)
    }
  }

  if (!ffEportfolio) {
    return (
      <LmsPage title="Content page">
        <p className="text-muted-foreground">
          The ePortfolio feature is not enabled. A global administrator can turn it on in Settings → Global
          platform.
        </p>
      </LmsPage>
    )
  }

  if (!pid || !aid) {
    return (
      <LmsPage title="Content page" description="">
        <p className="mt-6 text-sm text-slate-500">Invalid link.</p>
      </LmsPage>
    )
  }

  const description = updatedAt == null ? '' : `Updated ${formatAbsolute(updatedAt)}`
  const backTo = `/portfolios/${encodeURIComponent(pid)}`

  return (
    <LmsPage
      title={loading ? 'Content page' : title || 'Content page'}
      description={description}
      actions={
        editing ? (
          <div className="flex flex-wrap items-center gap-2">
            <ReadingFocusToggle />
            <button
              type="button"
              onClick={cancelEdit}
              disabled={saving}
              className="rounded-xl border border-slate-300 bg-white px-4 py-2.5 text-sm font-semibold text-slate-800 shadow-sm transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-100 dark:hover:bg-neutral-800"
            >
              Cancel
            </button>
            <button
              type="button"
              onClick={() => void save()}
              disabled={saving}
              className="rounded-xl bg-indigo-600 px-4 py-2.5 text-sm font-semibold text-white shadow-sm transition hover:bg-indigo-500 disabled:cursor-not-allowed disabled:opacity-60"
            >
              {saving ? 'Saving…' : 'Save'}
            </button>
          </div>
        ) : (
          <div className="flex flex-wrap items-center gap-2">
            <ReadingFocusToggle />
            <button
              type="button"
              onClick={beginEdit}
              disabled={loading || Boolean(loadError)}
              className="inline-flex items-center gap-2 rounded-xl border border-slate-300 bg-white px-4 py-2.5 text-sm font-semibold text-slate-800 shadow-sm transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-100 dark:hover:bg-neutral-800"
            >
              <Pencil className="h-4 w-4" aria-hidden />
              Edit
            </button>
          </div>
        )
      }
    >
      <p className="mt-2 text-start text-sm">
        <Link to={backTo} className="font-medium text-indigo-600 hover:text-indigo-500 dark:text-indigo-400 dark:hover:text-indigo-300">
          ← Back to portfolio
        </Link>
      </p>

      {!loading && !loadError ? (
        <div className="mt-6">
          <AuthoringSaveFootprint
            lastSavedIso={lastLocalAuthoringSave ?? updatedAt}
            saving={editing && saving}
            error={editing ? saveError : null}
            onRetry={editing ? () => void save() : undefined}
          />
        </div>
      ) : null}

      <div className="mx-auto w-full max-w-[72ch] min-w-0">
        {loadError && (
          <p className="mt-6 rounded-lg border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-800 dark:border-rose-900/60 dark:bg-rose-950/50 dark:text-rose-200">
            {loadError}
          </p>
        )}
        {loading && <p className="mt-8 text-sm text-slate-500">Loading…</p>}

        {!loading && !loadError && !editing && (
          <div className="mt-8 space-y-6 text-[1.0625rem] leading-relaxed">
            <MarkdownArticleView
              markdown={markdown}
              theme={mdTheme}
              emptyMessage="This page is empty. Click Edit to add content."
            />
          </div>
        )}
      </div>

      {!loading && !loadError && editing && (
        <div className="mt-6 -mx-6 md:-mx-8">
          {saveError && (
            <p className="mb-4 rounded-lg border border-rose-200 bg-rose-50 px-6 py-3 text-sm text-rose-800 dark:border-rose-900/60 dark:bg-rose-950/50 dark:text-rose-200 md:px-8">
              {saveError}
            </p>
          )}
          <div className="px-4 md:px-8">
            <SyllabusBlockEditor
              sections={draft}
              onChange={setDraft}
              disabled={saving}
              documentVariant="page"
            />
          </div>
        </div>
      )}
    </LmsPage>
  )
}