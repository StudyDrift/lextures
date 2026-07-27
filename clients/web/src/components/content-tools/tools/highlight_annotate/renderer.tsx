import { useEffect, useId, useMemo, useRef, useState, type KeyboardEvent } from 'react'
import type { ContentToolRendererProps } from '../../host/runtime-contract'
import {
  buildQuoteAnchor,
  plainPassageFromMarkdown,
  reanchorAnnotations,
  segmentPassage,
  type UnitGranularity,
} from '../../../../lib/text-anchoring'
import { AnnotationsPanel } from './annotations-panel'
import { PassagePanel } from './passage-panel'
import {
  asAnnotations,
  asTags,
  newAnnotationId,
  underlineStyle,
  type Anchor,
  type Annotation,
  type FilterNoteResult,
} from './types'

export default function HighlightAnnotateRenderer({
  config,
  state,
  readOnly,
  save,
  runAction,
  t,
  announce,
}: ContentToolRendererProps) {
  const promptId = useId()
  const menuId = useId()
  const passageRef = useRef<HTMLDivElement | null>(null)
  const [menuOpen, setMenuOpen] = useState(false)
  const [menuUnitIndex, setMenuUnitIndex] = useState<number | null>(null)
  const [pendingQuote, setPendingQuote] = useState<{ quote: string; anchor: Anchor } | null>(
    null,
  )
  const [noteDraft, setNoteDraft] = useState('')
  const [noteForId, setNoteForId] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [focusedUnit, setFocusedUnit] = useState(0)
  const [unitMode, setUnitMode] = useState(true)

  const prompt = typeof config.prompt === 'string' ? config.prompt : ''
  const passageMd = typeof config.passageMarkdown === 'string' ? config.passageMarkdown : ''
  const granularity = (
    config.unitGranularity === 'paragraph' || config.unitGranularity === 'line'
      ? config.unitGranularity
      : 'sentence'
  ) as UnitGranularity
  const tags = asTags(config)
  const minAnnotations =
    typeof config.minAnnotations === 'number' && config.minAnnotations > 0
      ? config.minAnnotations
      : 1
  const maxAnnotations =
    typeof config.maxAnnotations === 'number' && config.maxAnnotations > 0
      ? Math.min(50, config.maxAnnotations)
      : 20
  const requireNote = config.requireNote === true

  const passage = useMemo(() => plainPassageFromMarkdown(passageMd), [passageMd])
  const units = useMemo(() => segmentPassage(passage, granularity), [passage, granularity])
  const rawAnnotations = asAnnotations(state)
  const annotations = useMemo(
    () => reanchorAnnotations(passage, rawAnnotations),
    [passage, rawAnnotations],
  )
  const active = annotations.filter((a) => !a.orphaned)
  const orphaned = annotations.filter((a) => a.orphaned)

  useEffect(() => {
    const changed = annotations.some(
      (a, i) => Boolean(a.orphaned) !== Boolean(rawAnnotations[i]?.orphaned),
    )
    if (!changed || readOnly) return
    save({
      v: 1,
      annotations,
      ...(typeof state.completedAt === 'string' ? { completedAt: state.completedAt } : {}),
    })
    // eslint-disable-next-line react-hooks/exhaustive-deps -- only when orphan resolution drifts
  }, [annotations, rawAnnotations, readOnly])

  const countsByTag = useMemo(() => {
    const m = new Map<string, number>()
    for (const a of active) m.set(a.tagId, (m.get(a.tagId) ?? 0) + 1)
    return m
  }, [active])

  function persist(next: Annotation[]) {
    const completed =
      next.filter((a) => !a.orphaned).length >= minAnnotations
        ? typeof state.completedAt === 'string' && state.completedAt
          ? state.completedAt
          : new Date().toISOString()
        : undefined
    save({
      v: 1,
      annotations: next.slice(0, maxAnnotations),
      ...(completed ? { completedAt: completed } : {}),
    })
  }

  function openMenuForUnit(index: number) {
    if (readOnly) return
    const unit = units[index]
    if (!unit) return
    const built = buildQuoteAnchor(passage, unit.start, unit.end, index)
    if (!built) return
    setPendingQuote({ quote: built.quote, anchor: built.anchor })
    setMenuUnitIndex(index)
    setMenuOpen(true)
    setNoteDraft('')
    setError(null)
  }

  function onPointerUp() {
    if (readOnly || unitMode || !passageRef.current) return
    const sel = window.getSelection()
    if (!sel || sel.isCollapsed) return
    const range = sel.getRangeAt(0)
    if (!passageRef.current.contains(range.commonAncestorContainer)) return
    const quote = range.toString()
    if (!quote.trim()) return
    const full = passageRef.current.innerText || passage
    let start = full.indexOf(quote)
    if (start < 0) start = passage.indexOf(quote)
    if (start < 0) return
    const built = buildQuoteAnchor(passage, start, start + quote.length)
    if (!built) return
    const mid = start + quote.length / 2
    const unit = units.find((u) => mid >= u.start && mid < u.end)
    if (unit) built.anchor.unitIndex = unit.index
    setPendingQuote({ quote: built.quote, anchor: built.anchor })
    setMenuUnitIndex(unit?.index ?? null)
    setMenuOpen(true)
    setNoteDraft('')
    setError(null)
  }

  async function screenNote(note: string): Promise<boolean> {
    if (!note) return true
    try {
      const res = (await runAction('filter_note', { note })) as FilterNoteResult
      if (res?.error === 'filtered') {
        setError(res.message || t('contentTools.tools.highlight_annotate.filterBlocked'))
        return false
      }
      return true
    } catch {
      setError(t('contentTools.tools.highlight_annotate.filterBlocked'))
      return false
    }
  }

  async function applyTag(tagId: string) {
    if (!pendingQuote || readOnly) return
    if (annotations.length >= maxAnnotations) {
      setError(t('contentTools.tools.highlight_annotate.maxReached'))
      return
    }
    if (requireNote && !noteDraft.trim()) {
      setError(t('contentTools.tools.highlight_annotate.noteRequired'))
      return
    }
    const note = noteDraft.trim()
    if (!(await screenNote(note))) return
    const ann: Annotation = {
      id: newAnnotationId(),
      tagId,
      quote: pendingQuote.quote,
      anchor: pendingQuote.anchor,
      createdAt: new Date().toISOString(),
      ...(note ? { note } : {}),
    }
    persist([...active, ...orphaned.filter((o) => o.id !== ann.id), ann])
    const tag = tags.find((tg) => tg.id === tagId)
    announce(
      t('contentTools.tools.highlight_annotate.createdAnnounce', {
        tag: tag?.label ?? tagId,
      }),
    )
    setMenuOpen(false)
    setPendingQuote(null)
    setNoteDraft('')
    setNoteForId(null)
  }

  function removeAnnotation(id: string) {
    if (readOnly) return
    persist(annotations.filter((a) => a.id !== id))
    announce(t('contentTools.tools.highlight_annotate.deletedAnnounce'))
  }

  async function saveNote(id: string) {
    if (readOnly) return
    const note = noteDraft.trim()
    if (!(await screenNote(note))) return
    persist(annotations.map((a) => (a.id === id ? { ...a, note: note || undefined } : a)))
    setNoteForId(null)
    setNoteDraft('')
  }

  function onUnitKeyDown(e: KeyboardEvent, index: number) {
    if (readOnly) return
    if (e.key === 'ArrowRight' || e.key === 'ArrowDown') {
      e.preventDefault()
      const next = Math.min(units.length - 1, index + 1)
      setFocusedUnit(next)
      passageRef.current?.querySelector<HTMLElement>(`[data-unit-index="${next}"]`)?.focus()
      return
    }
    if (e.key === 'ArrowLeft' || e.key === 'ArrowUp') {
      e.preventDefault()
      const next = Math.max(0, index - 1)
      setFocusedUnit(next)
      passageRef.current?.querySelector<HTMLElement>(`[data-unit-index="${next}"]`)?.focus()
      return
    }
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault()
      openMenuForUnit(index)
    }
  }

  const progress = Math.min(active.length, minAnnotations)

  return (
    <div
      className="space-y-3"
      data-content-tool="highlight_annotate"
      data-testid="highlight-annotate-tool"
    >
      <div
        className="rounded border border-slate-200 bg-slate-50 px-3 py-2 dark:border-neutral-700 dark:bg-neutral-900/60"
        data-testid="ha-prompt"
      >
        <p id={promptId} className="text-sm font-medium text-slate-900 dark:text-neutral-100">
          {prompt}
        </p>
        <ul
          className="mt-2 flex flex-wrap gap-2"
          aria-label={t('contentTools.tools.highlight_annotate.tagLegend')}
        >
          {tags.map((tag, i) => (
            <li
              key={tag.id}
              className="inline-flex items-center gap-1.5 rounded border border-slate-200 px-2 py-0.5 text-xs dark:border-neutral-600"
            >
              <span
                aria-hidden
                className="inline-block h-2.5 w-2.5 rounded-sm"
                style={{ backgroundColor: tag.color, ...underlineStyle(tag.color, i) }}
              />
              <span>{tag.label}</span>
              <span className="text-slate-500">({countsByTag.get(tag.id) ?? 0})</span>
            </li>
          ))}
        </ul>
      </div>

      <div className="flex flex-wrap items-center gap-3 text-xs text-slate-600 dark:text-neutral-300">
        <p data-testid="ha-progress">
          {t('contentTools.tools.highlight_annotate.progress', {
            done: progress,
            required: minAnnotations,
          })}
        </p>
        {!readOnly ? (
          <label className="inline-flex items-center gap-1">
            <input
              type="checkbox"
              checked={unitMode}
              onChange={(e) => setUnitMode(e.target.checked)}
              data-testid="ha-unit-mode"
            />
            {t('contentTools.tools.highlight_annotate.unitMode')}
          </label>
        ) : null}
      </div>

      {error ? (
        <p className="text-sm text-rose-600" role="alert">
          {error}
        </p>
      ) : null}

      {!readOnly && annotations.length === 0 ? (
        <p className="text-xs text-slate-500" data-testid="ha-empty-hint">
          {t('contentTools.tools.highlight_annotate.emptyHint')}
        </p>
      ) : null}

      <PassagePanel
        passageRef={passageRef}
        promptId={promptId}
        units={units}
        tags={tags}
        active={active}
        focusedUnit={focusedUnit}
        unitMode={unitMode}
        readOnly={readOnly}
        t={t}
        onFocusUnit={setFocusedUnit}
        onOpenUnit={openMenuForUnit}
        onUnitKeyDown={onUnitKeyDown}
        onPointerUp={onPointerUp}
      />

      {menuOpen && pendingQuote && !readOnly ? (
        <div
          role="menu"
          id={menuId}
          aria-label={t('contentTools.tools.highlight_annotate.tagMenu')}
          className="rounded border border-slate-300 bg-white p-2 shadow-sm dark:border-neutral-600 dark:bg-neutral-950"
          data-testid="ha-tag-menu"
        >
          <p className="mb-2 line-clamp-2 text-xs text-slate-600 dark:text-neutral-300">
            “{pendingQuote.quote}”
          </p>
          <div className="flex flex-wrap gap-2">
            {tags.map((tag) => (
              <button
                key={tag.id}
                type="button"
                role="menuitem"
                className="rounded border border-slate-200 px-2 py-1 text-xs dark:border-neutral-600"
                style={{ borderColor: tag.color }}
                data-testid={`ha-tag-${tag.id}`}
                onClick={() => void applyTag(tag.id)}
              >
                {tag.label}
              </button>
            ))}
          </div>
          {requireNote || noteDraft || menuUnitIndex != null ? (
            <label className="mt-2 block space-y-1 text-xs">
              <span>{t('contentTools.tools.highlight_annotate.noteLabel')}</span>
              <textarea
                className="w-full rounded border border-slate-300 bg-white px-2 py-1 text-sm dark:border-neutral-600 dark:bg-neutral-950"
                rows={2}
                value={noteDraft}
                data-testid="ha-note-input"
                onChange={(e) => setNoteDraft(e.target.value)}
              />
            </label>
          ) : null}
          <button
            type="button"
            className="mt-2 text-xs text-slate-600 underline dark:text-neutral-300"
            onClick={() => {
              setMenuOpen(false)
              setPendingQuote(null)
            }}
          >
            {t('contentTools.tools.highlight_annotate.cancel')}
          </button>
        </div>
      ) : null}

      <AnnotationsPanel
        promptId={promptId}
        tags={tags}
        active={active}
        orphaned={orphaned}
        readOnly={readOnly}
        noteForId={noteForId}
        noteDraft={noteDraft}
        passageRef={passageRef}
        t={t}
        onFocusUnit={setFocusedUnit}
        onEditNote={(id, note) => {
          setNoteForId(id)
          setNoteDraft(note)
        }}
        onNoteDraftChange={setNoteDraft}
        onSaveNote={(id) => void saveNote(id)}
        onRemove={removeAnnotation}
      />
    </div>
  )
}
