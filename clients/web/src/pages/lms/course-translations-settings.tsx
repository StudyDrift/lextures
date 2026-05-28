import { useCallback, useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { Loader2 } from 'lucide-react'
import { usePermissions } from '../../context/use-permissions'
import {
  addGlossaryEntry,
  fetchCourseGlossary,
  fetchCourseTranslations,
  isTranslationMemoryEnabled,
  publishCourseTranslation,
  requestAIDraftTranslation,
  saveCourseTranslation,
  queryTranslationMemory,
  type TranslationListItem,
  type TranslationCoverage,
} from '../../lib/course-translation-api'
import { toastMutationError, toastSaveOk } from '../../lib/lms-toast'

const TARGET_LOCALE = 'es'

export default function CourseTranslationsSettings() {
  const { courseCode } = useParams<{ courseCode: string }>()
  const { allows, loading: permLoading } = usePermissions()
  const canTranslate = Boolean(
    courseCode &&
      (allows(`course:${courseCode}:content:translate`) ||
        allows(`course:${courseCode}:item:create`)),
  )
  const enabled = isTranslationMemoryEnabled()

  const [items, setItems] = useState<TranslationListItem[]>([])
  const [coverage, setCoverage] = useState<TranslationCoverage | null>(null)
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [draftBody, setDraftBody] = useState('')
  const [draftTitle, setDraftTitle] = useState('')
  const [tmHints, setTmHints] = useState<string[]>([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [glossarySource, setGlossarySource] = useState('')
  const [glossaryTarget, setGlossaryTarget] = useState('')

  const selected = items.find((i) => i.itemId === selectedId) ?? null

  const load = useCallback(async () => {
    if (!courseCode || !enabled) return
    setLoading(true)
    try {
      const data = await fetchCourseTranslations(courseCode, TARGET_LOCALE)
      setItems(data.items)
      setCoverage(data.coverage)
      if (!selectedId && data.items.length > 0) {
        setSelectedId(data.items[0].itemId)
      }
    } catch (e) {
      toastMutationError(e instanceof Error ? e.message : 'Failed to load translations')
    } finally {
      setLoading(false)
    }
  }, [courseCode, enabled, selectedId])

  useEffect(() => {
    void load()
  }, [load])

  useEffect(() => {
    if (!selected) return
    setDraftBody(selected.translatedBody ?? '')
    setDraftTitle(selected.translatedTitle ?? '')
  }, [selected])

  useEffect(() => {
    if (!courseCode || !selected || draftBody.length < 20) {
      setTmHints([])
      return
    }
    const t = window.setTimeout(() => {
      void queryTranslationMemory(courseCode, 'en', TARGET_LOCALE, draftBody).then((m) => {
        setTmHints(m.map((x) => x.translatedText).slice(0, 3))
      })
    }, 300)
    return () => window.clearTimeout(t)
  }, [courseCode, selected, draftBody])

  if (!enabled) {
    return (
      <p className="text-sm text-stone-600 dark:text-neutral-400">
        Ask a global administrator to enable translation memory in platform settings.
      </p>
    )
  }

  if (permLoading) {
    return <Loader2 className="h-6 w-6 animate-spin" aria-label="Loading" />
  }

  if (!canTranslate) {
    return (
      <p className="text-sm text-stone-600 dark:text-neutral-400">
        You do not have permission to manage course translations.{' '}
        <Link className="text-blue-600 underline" to={`/courses/${courseCode}`}>
          Back to course
        </Link>
      </p>
    )
  }

  async function handleSave(publish: boolean) {
    if (!courseCode || !selected) return
    setSaving(true)
    try {
      const saved = await saveCourseTranslation(courseCode, selected.itemId, {
        targetLocale: TARGET_LOCALE,
        translatedTitle: draftTitle || null,
        translatedBody: draftBody,
        isDraft: !publish,
        version: selected.version,
      })
      if (publish) {
        await publishCourseTranslation(courseCode, selected.itemId, TARGET_LOCALE)
      }
      toastSaveOk(publish ? 'Translation published' : 'Draft saved')
      void load()
      if (typeof saved.version === 'number') {
        setItems((prev) =>
          prev.map((it) =>
            it.itemId === selected.itemId ? { ...it, version: Number(saved.version) } : it,
          ),
        )
      }
    } catch (e) {
      toastMutationError(e instanceof Error ? e.message : 'Save failed')
    } finally {
      setSaving(false)
    }
  }

  async function handleAIDraft() {
    if (!courseCode || !selected) return
    setSaving(true)
    try {
      const row = await requestAIDraftTranslation(courseCode, selected.itemId, TARGET_LOCALE)
      setDraftBody(String(row.translatedBody ?? ''))
      if (row.translatedTitle) setDraftTitle(String(row.translatedTitle))
      toastSaveOk('AI draft created — review before publishing')
      void load()
    } catch (e) {
      toastMutationError(e instanceof Error ? e.message : 'AI draft failed')
    } finally {
      setSaving(false)
    }
  }

  async function handleAddGlossary() {
    if (!courseCode || !glossarySource.trim() || !glossaryTarget.trim()) return
    try {
      await addGlossaryEntry(courseCode, glossarySource.trim(), glossaryTarget.trim(), TARGET_LOCALE)
      setGlossarySource('')
      setGlossaryTarget('')
      toastSaveOk('Glossary entry added')
      await fetchCourseGlossary(courseCode, TARGET_LOCALE)
      void load()
    } catch (e) {
      toastMutationError(e instanceof Error ? e.message : 'Glossary save failed')
    }
  }

  return (
    <div>
      {coverage ? (
        <p className="mb-4 text-sm text-stone-600 dark:text-neutral-400" role="status">
          {Math.round(coverage.percent)}% of items have a published {TARGET_LOCALE} translation (
          {coverage.translatedItems}/{coverage.totalItems}).
        </p>
      ) : null}

      {loading ? (
        <Loader2 className="h-6 w-6 animate-spin" aria-label="Loading" />
      ) : (
        <div className="grid gap-4 lg:grid-cols-[240px_1fr]">
          <ul className="space-y-1 rounded-lg border border-stone-200 p-2 dark:border-neutral-700">
            {items.map((it) => (
              <li key={it.itemId}>
                <button
                  type="button"
                  className={`w-full rounded px-2 py-1.5 text-start text-sm ${
                    selectedId === it.itemId
                      ? 'bg-stone-200 dark:bg-neutral-700'
                      : 'hover:bg-stone-100 dark:hover:bg-neutral-800'
                  }`}
                  onClick={() => setSelectedId(it.itemId)}
                >
                  {it.title || 'Untitled'}
                  {it.hasPublished ? (
                    <span className="ms-1 text-xs text-green-700 dark:text-green-400">✓</span>
                  ) : null}
                </button>
              </li>
            ))}
          </ul>

          {selected ? (
            <div className="grid gap-4 md:grid-cols-2">
              <div>
                <h2 className="text-sm font-semibold text-stone-800 dark:text-neutral-200">Source</h2>
                <p className="mt-1 text-sm font-medium">{selected.title}</p>
                <div className="mt-2 max-h-64 overflow-auto whitespace-pre-wrap rounded border border-stone-200 p-3 text-sm dark:border-neutral-700">
                  {selected.body}
                </div>
                {selected.glossaryMatches && selected.glossaryMatches.length > 0 ? (
                  <p className="mt-2 text-xs text-amber-800 dark:text-amber-300">
                    Glossary terms in this segment:{' '}
                    {selected.glossaryMatches.map((g) => g.sourceTerm).join(', ')}
                  </p>
                ) : null}
              </div>
              <div>
                <h2 className="text-sm font-semibold text-stone-800 dark:text-neutral-200">
                  Translation ({TARGET_LOCALE})
                </h2>
                {selected.machineTranslationDraft ? (
                  <p className="text-xs text-amber-700 dark:text-amber-400">AI-generated draft — review required</p>
                ) : null}
                <input
                  className="mt-2 w-full rounded border border-stone-300 px-2 py-1 text-sm dark:border-neutral-600 dark:bg-neutral-900"
                  value={draftTitle}
                  onChange={(e) => setDraftTitle(e.target.value)}
                  placeholder="Translated title"
                  aria-label="Translated title"
                />
                <textarea
                  className="mt-2 min-h-[200px] w-full rounded border border-stone-300 p-2 text-sm dark:border-neutral-600 dark:bg-neutral-900"
                  value={draftBody}
                  onChange={(e) => setDraftBody(e.target.value)}
                  dir="ltr"
                  aria-label="Translated body"
                />
                {tmHints.length > 0 ? (
                  <div className="mt-2 rounded border border-blue-200 bg-blue-50 p-2 text-xs dark:border-blue-900 dark:bg-blue-950">
                    <p className="font-medium">Translation memory</p>
                    <ul className="mt-1 list-disc ps-4">
                      {tmHints.map((h, i) => (
                        <li key={i}>
                          <button
                            type="button"
                            className="text-start underline"
                            onClick={() => setDraftBody(h)}
                          >
                            {h.slice(0, 120)}
                            {h.length > 120 ? '…' : ''}
                          </button>
                        </li>
                      ))}
                    </ul>
                  </div>
                ) : null}
                <div className="mt-3 flex flex-wrap gap-2">
                  <button
                    type="button"
                    className="rounded bg-stone-800 px-3 py-1.5 text-sm text-white disabled:opacity-50 dark:bg-neutral-200 dark:text-neutral-900"
                    disabled={saving}
                    onClick={() => void handleSave(false)}
                  >
                    Save draft
                  </button>
                  <button
                    type="button"
                    className="rounded border border-stone-300 px-3 py-1.5 text-sm dark:border-neutral-600"
                    disabled={saving}
                    onClick={() => void handleAIDraft()}
                  >
                    AI draft
                  </button>
                  <button
                    type="button"
                    className="rounded bg-green-700 px-3 py-1.5 text-sm text-white disabled:opacity-50"
                    disabled={saving}
                    onClick={() => void handleSave(true)}
                  >
                    Publish
                  </button>
                </div>
              </div>
            </div>
          ) : null}
        </div>
      )}

      <section className="mt-8 border-t border-stone-200 pt-6 dark:border-neutral-700">
        <h2 className="text-sm font-semibold">Course glossary</h2>
        <div className="mt-2 flex flex-wrap gap-2">
          <input
            className="rounded border border-stone-300 px-2 py-1 text-sm dark:border-neutral-600 dark:bg-neutral-900"
            placeholder="Source term"
            value={glossarySource}
            onChange={(e) => setGlossarySource(e.target.value)}
            aria-label="Glossary source term"
          />
          <input
            className="rounded border border-stone-300 px-2 py-1 text-sm dark:border-neutral-600 dark:bg-neutral-900"
            placeholder="Target term"
            value={glossaryTarget}
            onChange={(e) => setGlossaryTarget(e.target.value)}
            aria-label="Glossary target term"
          />
          <button
            type="button"
            className="rounded bg-stone-800 px-3 py-1.5 text-sm text-white dark:bg-neutral-200 dark:text-neutral-900"
            onClick={() => void handleAddGlossary()}
          >
            Add term
          </button>
        </div>
      </section>
    </div>
  )
}
