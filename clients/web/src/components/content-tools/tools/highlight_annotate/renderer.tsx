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
import { HowToCoach, TaskPanel } from './task-panel'
import { TagMenu } from './tag-menu'
import {
  asAnnotations,
  asTags,
  newAnnotationId,
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
  const menuRef = useRef<HTMLDivElement | null>(null)
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
  const [applying, setApplying] = useState(false)

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

  useEffect(() => {
    if (!menuOpen) return
    menuRef.current?.querySelector<HTMLElement>('button[role="menuitem"]')?.focus()
  }, [menuOpen])

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

  function closeMenu() {
    setMenuOpen(false)
    setPendingQuote(null)
    setMenuUnitIndex(null)
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
    sel.removeAllRanges()
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
    if (!pendingQuote || readOnly || applying) return
    if (annotations.length >= maxAnnotations) {
      setError(t('contentTools.tools.highlight_annotate.maxReached'))
      return
    }
    if (requireNote && !noteDraft.trim()) {
      setError(t('contentTools.tools.highlight_annotate.noteRequired'))
      menuRef.current?.querySelector<HTMLTextAreaElement>('[data-testid="ha-note-input"]')?.focus()
      return
    }
    setApplying(true)
    try {
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
      closeMenu()
      setNoteForId(null)
    } finally {
      setApplying(false)
    }
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
    if (e.key === 'Escape' && menuOpen) {
      e.preventDefault()
      closeMenu()
    }
  }

  const progress = Math.min(active.length, minAnnotations)
  const remaining = Math.max(0, minAnnotations - active.length)
  const complete = active.length >= minAnnotations
  const progressPct = Math.min(100, Math.round((progress / Math.max(1, minAnnotations)) * 100))
  const showCoach = !readOnly && !complete && !menuOpen
  const unitWord =
    granularity === 'paragraph'
      ? t('contentTools.tools.highlight_annotate.unitWord.paragraph')
      : granularity === 'line'
        ? t('contentTools.tools.highlight_annotate.unitWord.line')
        : t('contentTools.tools.highlight_annotate.unitWord.sentence')

  return (
    <div
      className="space-y-4"
      data-content-tool="highlight_annotate"
      data-testid="highlight-annotate-tool"
    >
      <TaskPanel
        promptId={promptId}
        prompt={prompt}
        tags={tags}
        countsByTag={countsByTag}
        progress={progress}
        minAnnotations={minAnnotations}
        activeCount={active.length}
        remaining={remaining}
        complete={complete}
        progressPct={progressPct}
        t={t}
      />

      {showCoach ? <HowToCoach unitWord={unitWord} requireNote={requireNote} t={t} /> : null}

      {error ? (
        <p
          className="rounded-lg border border-rose-200 bg-rose-50 px-3 py-2 text-sm text-rose-700 dark:border-rose-900 dark:bg-rose-950/40 dark:text-rose-200"
          role="alert"
        >
          {error}
        </p>
      ) : null}

      <div className="space-y-2">
        <div className="flex flex-wrap items-end justify-between gap-2">
          <div>
            <h3 className="text-sm font-semibold text-slate-800 dark:text-neutral-100">
              {t('contentTools.tools.highlight_annotate.passageTitle')}
            </h3>
            {!readOnly ? (
              <p className="text-xs text-slate-500 dark:text-neutral-400">
                {unitMode
                  ? t('contentTools.tools.highlight_annotate.passageHintTap', { unit: unitWord })
                  : t('contentTools.tools.highlight_annotate.passageHintSelect')}
              </p>
            ) : null}
          </div>
          {!readOnly ? (
            <label className="inline-flex cursor-pointer items-center gap-1.5 text-xs text-slate-600 dark:text-neutral-300">
              <input
                type="checkbox"
                className="rounded border-slate-300 text-indigo-600 focus:ring-indigo-500"
                checked={!unitMode}
                onChange={(e) => setUnitMode(!e.target.checked)}
                data-testid="ha-unit-mode"
              />
              {t('contentTools.tools.highlight_annotate.selectWordsInstead')}
            </label>
          ) : null}
        </div>

        <PassagePanel
          passageRef={passageRef}
          promptId={promptId}
          units={units}
          tags={tags}
          active={active}
          focusedUnit={focusedUnit}
          menuUnitIndex={menuUnitIndex}
          unitMode={unitMode}
          readOnly={readOnly}
          showStartCue={!readOnly && unitMode && active.length === 0 && !menuOpen}
          t={t}
          onFocusUnit={setFocusedUnit}
          onOpenUnit={openMenuForUnit}
          onUnitKeyDown={onUnitKeyDown}
          onPointerUp={onPointerUp}
        />
      </div>

      {menuOpen && pendingQuote && !readOnly ? (
        <TagMenu
          menuRef={menuRef}
          menuId={menuId}
          quote={pendingQuote.quote}
          tags={tags}
          noteDraft={noteDraft}
          requireNote={requireNote}
          applying={applying}
          t={t}
          onNoteDraftChange={setNoteDraft}
          onApplyTag={(tagId) => void applyTag(tagId)}
          onClose={closeMenu}
        />
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
