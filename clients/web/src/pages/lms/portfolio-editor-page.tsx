import { useCallback, useEffect, useId, useMemo, useRef, useState, type CSSProperties } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import {
  DndContext,
  DragOverlay,
  PointerSensor,
  closestCenter,
  useSensor,
  useSensors,
  type DragEndEvent,
  type DragStartEvent,
} from '@dnd-kit/core'
import {
  SortableContext,
  arrayMove,
  useSortable,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable'
import { KeyboardSensor, defaultKeyboardSensorOptions } from '../../lib/dnd/keyboardSensorConfig'
import { CSS, type Transform } from '@dnd-kit/utilities'
import {
  AlertTriangle,
  ArrowLeft,
  Check,
  ChevronDown,
  Copy,
  ExternalLink,
  Eye,
  EyeOff,
  FileText,
  GripVertical,
  Heading,
  Link2,
  MoreVertical,
  Pencil,
  Plus,
  Trash2,
  X,
} from 'lucide-react'
import {
  createArtifact,
  deleteArtifact as apiDeleteArtifact,
  getMyPortfolio,
  patchArtifact,
  patchPortfolio,
  type Artifact,
  type ArtifactType,
  isPortfolioContentPage,
  isPortfolioHeading,
  portfolioContentPageHref,
  type Evaluation,
  type Portfolio,
} from '../../lib/eportfolio-api'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { LmsPage } from './lms-page'
import { ModuleNameModal } from './module-name-modal'
import { EmptyState } from '../../components/ui/empty-state'
import { IconSwap } from '../../components/ui/icon-swap'

type PortfolioArtifactKind = 'heading' | 'content_page' | 'url'

const PORTFOLIO_ARTIFACTS_SORT_ID = 'sortable-portfolio-artifacts'

function sortableDragStyle(
  transform: Transform | null,
  transition: string | undefined,
  isDragging: boolean,
): CSSProperties {
  return {
    transform: CSS.Transform.toString(transform),
    transition: isDragging ? undefined : transition,
    opacity: isDragging ? 0 : undefined,
    pointerEvents: isDragging ? 'none' : undefined,
  }
}

function artifactTypeLabel(a: Artifact): string {
  if (isPortfolioHeading(a)) return 'Heading'
  switch (a.artifactType) {
    case 'text_page':
      return 'Content page'
    case 'url':
      return 'External link'
    case 'submission':
      return 'Submission'
    case 'upload':
      return 'Upload'
    default:
      return a.artifactType.replace('_', ' ')
  }
}

const iconGhostPublished =
  'rounded-md p-2 text-indigo-600 transition-[background-color,color,border-color] hover:bg-indigo-50/90 hover:text-indigo-700 disabled:cursor-not-allowed disabled:opacity-50 dark:text-indigo-400 dark:hover:bg-indigo-950/45 dark:hover:text-indigo-300'
const iconGhostDraft =
  'rounded-md p-2 text-slate-400 transition-[background-color,color,border-color] hover:bg-slate-200/45 hover:text-slate-600 disabled:cursor-not-allowed disabled:opacity-50 dark:text-neutral-500 dark:hover:bg-neutral-700/35 dark:hover:text-neutral-300'
const iconGhost =
  'rounded-md p-2 text-slate-500 transition-[background-color,color,border-color] hover:bg-slate-200/45 hover:text-slate-800 disabled:cursor-not-allowed disabled:opacity-50 dark:text-neutral-400 dark:hover:bg-neutral-700/35 dark:hover:text-neutral-200'

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
  onTogglePublished,
  onEditTitle,
  onDelete,
}: {
  artifact: Artifact
  onTogglePublished: () => void
  onEditTitle?: () => void
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
        <IconSwap
          active={artifact.isPublic}
          activeIcon={Eye}
          inactiveIcon={EyeOff}
          iconClassName="h-4 w-4"
        />
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
            {onEditTitle ? (
              <button
                type="button"
                role="menuitem"
                onClick={() => { onEditTitle(); setMenuOpen(false) }}
                className="flex w-full px-2.5 py-2 text-start text-sm font-medium text-slate-800 transition-[background-color,color,border-color] hover:bg-slate-50 dark:text-neutral-100 dark:hover:bg-neutral-700/80"
              >
                Edit title
              </button>
            ) : null}
            <button
              type="button"
              role="menuitem"
              onClick={() => { onDelete(); setMenuOpen(false) }}
              className="flex w-full items-center gap-2 border-t border-slate-100 px-2.5 py-2 text-start text-sm font-medium text-rose-700 transition-[background-color,color,border-color] hover:bg-rose-50 dark:border-neutral-700 dark:text-rose-300 dark:hover:bg-rose-950/50"
            >
              <Trash2 className="h-4 w-4" aria-hidden /> Delete
            </button>
          </div>
        )}
      </div>
    </div>
  )
}

function AddArtifactMenu({
  onSelect,
  disabled,
  dragHandlesVisible,
  onToggleDragHandles,
  artifactListActionsEnabled,
}: {
  onSelect: (kind: PortfolioArtifactKind) => void
  disabled?: boolean
  dragHandlesVisible: boolean
  onToggleDragHandles: () => void
  artifactListActionsEnabled: boolean
}) {
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
        disabled={disabled}
        aria-haspopup="menu"
        aria-expanded={open}
        aria-controls={open ? menuId : undefined}
        onClick={() => {
          if (disabled) return
          setOpen((o) => !o)
        }}
        className="inline-flex items-center gap-1.5 rounded-lg border border-slate-200/70 bg-white/90 px-2 py-1.5 text-xs font-medium text-slate-700 shadow-none transition-[background-color,color,border-color] hover:border-slate-300/80 hover:bg-slate-50/90 disabled:cursor-not-allowed disabled:opacity-60 sm:px-2.5 sm:text-sm dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-200 dark:hover:border-neutral-500 dark:hover:bg-neutral-800"
      >
        <Plus className="h-4 w-4 shrink-0" aria-hidden />
        <span className="truncate">Add artifact</span>
        <ChevronDown
          className={`h-4 w-4 shrink-0 transition-transform ${open ? 'rotate-180' : ''}`}
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
            className="flex w-full items-start gap-3 px-2.5 py-2 text-start text-sm transition-[background-color,color,border-color] hover:bg-slate-50 dark:hover:bg-neutral-700"
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
            className="flex w-full items-start gap-3 border-t border-slate-100 px-2.5 py-2 text-start text-sm transition-[background-color,color,border-color] hover:bg-slate-50 dark:border-neutral-700 dark:hover:bg-neutral-700"
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
            className="flex w-full items-start gap-3 border-t border-slate-100 px-2.5 py-2 text-start text-sm transition-[background-color,color,border-color] hover:bg-slate-50 dark:border-neutral-700 dark:hover:bg-neutral-700"
          >
            <span className="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-sky-200/90 bg-sky-50 text-sky-700 dark:border-sky-500/40 dark:bg-sky-950 dark:text-sky-200">
              <ExternalLink className="h-4 w-4" aria-hidden />
            </span>
            <span className="flex min-w-0 flex-col gap-0.5">
              <span className="font-semibold text-slate-950 dark:text-neutral-100">External link</span>
              <span className="text-xs text-slate-500 dark:text-neutral-400">Opens a URL in a new tab</span>
            </span>
          </button>
          {artifactListActionsEnabled ? (
            <>
              <div className="my-1 border-t border-slate-100 dark:border-neutral-700" role="separator" />
              <button
                type="button"
                role="menuitemcheckbox"
                aria-checked={dragHandlesVisible}
                onClick={onToggleDragHandles}
                className="flex w-full items-start gap-2 px-2.5 py-2 text-start text-sm transition-[background-color,color,border-color] hover:bg-slate-50 dark:hover:bg-neutral-700"
              >
                <span
                  className={`mt-0.5 flex h-4 w-4 shrink-0 items-center justify-center rounded border ${
                    dragHandlesVisible
                      ? 'border-indigo-600 bg-indigo-600 text-white dark:border-indigo-500 dark:bg-indigo-500'
                      : 'border-slate-300 bg-white dark:border-neutral-500 dark:bg-neutral-800'
                  }`}
                  aria-hidden
                >
                  {dragHandlesVisible ? <Check className="h-3 w-3" strokeWidth={3} /> : null}
                </span>
                <span className="flex min-w-0 flex-1 flex-col gap-0.5">
                  <span className="font-semibold text-slate-950 dark:text-neutral-100">Drag and drop</span>
                  <span className="text-xs text-slate-500 dark:text-neutral-400">Show reorder handles for artifacts</span>
                </span>
              </button>
            </>
          ) : null}
        </div>
      )}
    </div>
  )
}

function ArtifactRowContent({
  artifact,
  pid,
  evaluations,
}: {
  artifact: Artifact
  pid: string
  evaluations: Evaluation[]
}) {
  const heading = isPortfolioHeading(artifact)
  const contentPage = isPortfolioContentPage(artifact)
  const evals = evaluations.filter((e) => e.artifactId === artifact.id)

  if (heading) {
    return (
      <p className="text-lg font-bold leading-snug tracking-tight text-slate-950 sm:text-xl dark:text-neutral-100">
        {artifact.title}
      </p>
    )
  }

  if (contentPage) {
    return (
      <div className="flex min-w-0 items-center gap-3">
        <ArtifactTypeIcon type="text_page" />
        <div className="min-w-0 flex-1">
          <Link
            to={portfolioContentPageHref(pid, artifact.id)}
            className="min-w-0 flex-1 text-base font-semibold leading-snug tracking-tight text-indigo-600 hover:text-indigo-500 dark:text-indigo-400 dark:hover:text-indigo-300"
          >
            {artifact.title}
          </Link>
        </div>
      </div>
    )
  }

  return (
    <div className="flex min-w-0 items-center gap-3">
      <ArtifactTypeIcon type={artifact.artifactType} />
      <div className="min-w-0 flex-1">
        <div className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1">
          <p className="text-base font-semibold leading-snug tracking-tight text-slate-900 dark:text-neutral-100">
            {artifact.title}
          </p>
          <p className="inline-flex shrink-0 items-center rounded bg-slate-100 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wider text-slate-600 dark:bg-neutral-750 dark:text-neutral-400">
            {artifactTypeLabel(artifact)}
            {artifact.fileName ? ` · ${artifact.fileName}` : ''}
          </p>
        </div>
        {artifact.description && (
          <p className="mt-0.5 text-sm text-slate-500 dark:text-neutral-400">{artifact.description}</p>
        )}
        {artifact.externalUrl && (
          <div className="mt-1">
            <a
              href={artifact.externalUrl}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-1 text-xs font-semibold text-primary hover:underline"
            >
              <ExternalLink className="h-3 w-3" aria-hidden /> {artifact.externalUrl}
            </a>
          </div>
        )}
        {artifact.outcomeIds.length > 0 && (
          <p className="mt-1 text-xs font-medium text-slate-500 dark:text-neutral-400">
            {artifact.outcomeIds.length} outcome{artifact.outcomeIds.length === 1 ? '' : 's'} tagged
          </p>
        )}
        {evals.map((ev) => (
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
  )
}

type SortableArtifactRowProps = {
  artifact: Artifact
  pid: string
  evaluations: Evaluation[]
  disabled: boolean
  dragHandlesVisible: boolean
  onTogglePublished: () => void
  onEditTitle?: () => void
  onDelete: () => void
}

function SortableArtifactRow({
  artifact,
  pid,
  evaluations,
  disabled,
  dragHandlesVisible,
  onTogglePublished,
  onEditTitle,
  onDelete,
}: SortableArtifactRowProps) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: artifact.id,
    disabled,
  })
  const style = sortableDragStyle(transform, transition, isDragging)
  const gripAlwaysOn = dragHandlesVisible || isDragging

  return (
    <li ref={setNodeRef} style={style} className="group py-3">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:gap-2">
        <div className="flex min-w-0 flex-1 items-center gap-2">
          {(!disabled || dragHandlesVisible || isDragging) && (
            <button
              type="button"
              className={`flex h-11 w-11 shrink-0 cursor-grab touch-none items-center justify-center rounded-lg border-0 bg-transparent p-0 text-slate-400 shadow-none transition-[opacity,background-color,color,border-color] hover:text-slate-600 active:cursor-grabbing focus-visible:pointer-events-auto focus-visible:opacity-100 focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-indigo-500 sm:h-9 sm:w-9 dark:text-neutral-500 dark:hover:text-neutral-300 ${
                gripAlwaysOn
                  ? 'opacity-100'
                  : 'opacity-0 pointer-events-none group-hover:opacity-100 group-hover:pointer-events-auto group-focus-within:opacity-100 group-focus-within:pointer-events-auto'
              }`}
              aria-label="Drag to reorder item"
              {...listeners}
              {...attributes}
            >
              <GripVertical className="h-4 w-4" strokeWidth={2} aria-hidden />
            </button>
          )}
          <div className="min-w-0 flex-1">
            <ArtifactRowContent artifact={artifact} pid={pid} evaluations={evaluations} />
          </div>
        </div>
        <div
          className={`flex shrink-0 justify-end sm:items-center sm:self-center sm:ps-0 ${!disabled ? 'ps-[3.25rem]' : 'ps-0'}`}
        >
          <ArtifactItemActions
            artifact={artifact}
            onTogglePublished={onTogglePublished}
            onEditTitle={onEditTitle}
            onDelete={onDelete}
          />
        </div>
      </div>
    </li>
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
      const artifactType: ArtifactType =
        kind === 'url' ? 'url' : kind === 'heading' ? 'heading' : 'text_page'
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
          className="rounded-xl bg-primary px-4 py-2.5 text-sm font-semibold text-primary-foreground hover:bg-primary/90 shadow-sm transition-[background-color,color,border-color] disabled:opacity-50"
        >
          {saving ? 'Adding…' : 'Add'}
        </button>
        <button
          type="button"
          onClick={onCancel}
          className="rounded-xl border border-slate-200 bg-white px-4 py-2.5 text-sm font-medium text-slate-700 hover:bg-slate-50 transition-[background-color,color,border-color] dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-300 dark:hover:bg-neutral-800"
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
      className={`inline-flex items-center gap-2 rounded-xl border px-4 py-2.5 text-sm font-semibold shadow-sm transition-[background-color,color,border-color] ${
        portfolio.isPublic
          ? 'border-indigo-200 bg-indigo-50 text-indigo-700 hover:bg-indigo-100 dark:border-indigo-900/40 dark:bg-indigo-950/30 dark:text-indigo-300 dark:hover:bg-indigo-900/40'
          : 'border-slate-200 bg-white text-slate-700 hover:bg-slate-50 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-300 dark:hover:bg-neutral-800'
      }`}
    >
      <IconSwap
        active={portfolio.isPublic}
        activeIcon={Eye}
        inactiveIcon={EyeOff}
        iconClassName="h-4 w-4"
      />
      {portfolio.isPublic ? 'Published' : 'Draft'}
    </button>
  )
}

export default function PortfolioEditorPage() {
  const { pid = '' } = useParams<{ pid: string }>()
  const navigate = useNavigate()
  const { ffEportfolio } = usePlatformFeatures()
  const [portfolio, setPortfolio] = useState<Portfolio | null>(null)
  const [artifacts, setArtifacts] = useState<Artifact[]>([])
  const [evaluations, setEvaluations] = useState<Evaluation[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)
  const [addKind, setAddKind] = useState<PortfolioArtifactKind | null>(null)
  const [headingModalOpen, setHeadingModalOpen] = useState(false)
  const [headingModalKey, setHeadingModalKey] = useState(0)
  const [headingSaving, setHeadingSaving] = useState(false)
  const [headingSaveError, setHeadingSaveError] = useState<string | null>(null)
  const [contentPageModalOpen, setContentPageModalOpen] = useState(false)
  const [contentPageModalKey, setContentPageModalKey] = useState(0)
  const [contentPageSaving, setContentPageSaving] = useState(false)
  const [contentPageSaveError, setContentPageSaveError] = useState<string | null>(null)
  const [editTitleTarget, setEditTitleTarget] = useState<Artifact | null>(null)
  const [editTitleModalKey, setEditTitleModalKey] = useState(0)
  const [editTitleSaving, setEditTitleSaving] = useState(false)
  const [editTitleError, setEditTitleError] = useState<string | null>(null)
  const [portfolioTitleModalOpen, setPortfolioTitleModalOpen] = useState(false)
  const [portfolioTitleModalKey, setPortfolioTitleModalKey] = useState(0)
  const [portfolioTitleSaving, setPortfolioTitleSaving] = useState(false)
  const [portfolioTitleError, setPortfolioTitleError] = useState<string | null>(null)
  const [deleteConfirmArtifact, setDeleteConfirmArtifact] = useState<Artifact | null>(null)
  const [deletingArtifactId, setDeletingArtifactId] = useState<string | null>(null)
  const deleteDialogTitleId = useId()
  const [activeDragId, setActiveDragId] = useState<string | null>(null)
  const [dragHandlesVisible, setDragHandlesVisible] = useState(false)

  const anyModalBusy =
    headingModalOpen ||
    headingSaving ||
    contentPageModalOpen ||
    contentPageSaving ||
    editTitleTarget !== null ||
    editTitleSaving ||
    portfolioTitleModalOpen ||
    portfolioTitleSaving ||
    deleteConfirmArtifact !== null ||
    Boolean(deletingArtifactId) ||
    addKind !== null

  const artifactIds = useMemo(() => artifacts.map((a) => a.id), [artifacts])

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 6 } }),
    useSensor(KeyboardSensor, defaultKeyboardSensorOptions),
  )

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

  useEffect(() => {
    if (!deleteConfirmArtifact) return
    const artifactId = deleteConfirmArtifact.id
    function onKey(e: KeyboardEvent) {
      if (e.key !== 'Escape') return
      if (deletingArtifactId === artifactId) return
      e.preventDefault()
      setDeleteConfirmArtifact(null)
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [deleteConfirmArtifact, deletingArtifactId])

  const toggleVisibility = async () => {
    if (!portfolio) return
    try {
      const updated = await patchPortfolio(pid, { isPublic: !portfolio.isPublic })
      setPortfolio(updated)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update visibility.')
    }
  }

  const persistArtifactOrder = useCallback(
    async (order: string[]) => {
      try {
        await patchPortfolio(pid, { order })
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to reorder.')
        void load()
      }
    },
    [pid, load],
  )

  const handleDragStart = useCallback((event: DragStartEvent) => {
    setActiveDragId(String(event.active.id))
  }, [])

  const handleDragEnd = useCallback(
    (event: DragEndEvent) => {
      const { active, over } = event
      setActiveDragId(null)
      if (!over || active.id === over.id) return

      setArtifacts((prev) => {
        const oldIndex = prev.findIndex((a) => a.id === active.id)
        const newIndex = prev.findIndex((a) => a.id === over.id)
        if (oldIndex < 0 || newIndex < 0) return prev
        const reordered = arrayMove(prev, oldIndex, newIndex)
        void persistArtifactOrder(reordered.map((a) => a.id))
        return reordered
      })
    },
    [persistArtifactOrder],
  )

  const handleDragCancel = useCallback(() => {
    setActiveDragId(null)
  }, [])

  const activeDragArtifact = useMemo(() => {
    if (!activeDragId) return null
    return artifacts.find((a) => a.id === activeDragId) ?? null
  }, [activeDragId, artifacts])

  const toggleArtifactPublic = async (a: Artifact) => {
    try {
      const updated = await patchArtifact(pid, a.id, { isPublic: !a.isPublic })
      setArtifacts((prev) => prev.map((x) => (x.id === a.id ? updated : x)))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update artifact.')
    }
  }

  const requestDeleteArtifact = (a: Artifact) => {
    setDeleteConfirmArtifact(a)
  }

  const confirmDeleteArtifact = async () => {
    if (!deleteConfirmArtifact) return
    const artifact = deleteConfirmArtifact
    setDeletingArtifactId(artifact.id)
    try {
      await apiDeleteArtifact(pid, artifact.id)
      setArtifacts((prev) => prev.filter((x) => x.id !== artifact.id))
      setDeleteConfirmArtifact(null)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to remove artifact.')
    } finally {
      setDeletingArtifactId(null)
    }
  }

  const onArtifactAdded = (a: Artifact) => {
    setArtifacts((prev) => [...prev, a])
    setAddKind(null)
  }

  const openAddHeading = () => {
    setHeadingSaveError(null)
    setHeadingModalKey((k) => k + 1)
    setHeadingModalOpen(true)
  }

  const saveNewHeading = async (title: string) => {
    setHeadingSaveError(null)
    setHeadingSaving(true)
    try {
      const created = await createArtifact(pid, { artifactType: 'heading', title })
      setArtifacts((prev) => [...prev, created])
      setHeadingModalOpen(false)
    } catch (err) {
      setHeadingSaveError(err instanceof Error ? err.message : 'Could not save heading.')
    } finally {
      setHeadingSaving(false)
    }
  }

  const openAddContentPage = () => {
    setContentPageSaveError(null)
    setContentPageModalKey((k) => k + 1)
    setContentPageModalOpen(true)
  }

  const saveNewContentPage = async (title: string) => {
    setContentPageSaveError(null)
    setContentPageSaving(true)
    try {
      const created = await createArtifact(pid, { artifactType: 'text_page', title })
      setArtifacts((prev) => [...prev, created])
      setContentPageModalOpen(false)
      navigate(portfolioContentPageHref(pid, created.id))
    } catch (err) {
      setContentPageSaveError(err instanceof Error ? err.message : 'Could not save page.')
    } finally {
      setContentPageSaving(false)
    }
  }

  const openEditTitle = (a: Artifact) => {
    setEditTitleError(null)
    setEditTitleTarget(a)
    setEditTitleModalKey((k) => k + 1)
  }

  const saveEditTitle = async (title: string) => {
    if (!editTitleTarget) return
    setEditTitleError(null)
    setEditTitleSaving(true)
    try {
      const updated = await patchArtifact(pid, editTitleTarget.id, { title })
      setArtifacts((prev) => prev.map((x) => (x.id === editTitleTarget.id ? updated : x)))
      setEditTitleTarget(null)
    } catch (err) {
      setEditTitleError(err instanceof Error ? err.message : 'Could not save title.')
    } finally {
      setEditTitleSaving(false)
    }
  }

  const openPortfolioTitleEdit = () => {
    setPortfolioTitleError(null)
    setPortfolioTitleModalOpen(true)
    setPortfolioTitleModalKey((k) => k + 1)
  }

  const savePortfolioTitle = async (title: string) => {
    setPortfolioTitleError(null)
    setPortfolioTitleSaving(true)
    try {
      const updated = await patchPortfolio(pid, { title })
      setPortfolio(updated)
      setPortfolioTitleModalOpen(false)
    } catch (err) {
      setPortfolioTitleError(err instanceof Error ? err.message : 'Could not save title.')
    } finally {
      setPortfolioTitleSaving(false)
    }
  }

  const onAddArtifactKind = (kind: PortfolioArtifactKind) => {
    if (kind === 'heading') {
      openAddHeading()
      return
    }
    if (kind === 'content_page') {
      openAddContentPage()
      return
    }
    setAddKind(kind)
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
          <Link to="/portfolios" className="inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground">
            <ArrowLeft className="h-3.5 w-3.5" aria-hidden /> Portfolios
          </Link>
          <div className="mt-2 mb-2 flex items-center gap-2">
            <h1 className="text-3xl font-bold tracking-tight">{portfolio.title}</h1>
            <button
              type="button"
              onClick={openPortfolioTitleEdit}
              disabled={anyModalBusy}
              title="Rename portfolio"
              aria-label="Rename portfolio"
              className="rounded-lg p-1.5 text-muted-foreground transition-[background-color,color,border-color] hover:bg-slate-100 hover:text-foreground disabled:cursor-not-allowed disabled:opacity-50 dark:hover:bg-neutral-800"
            >
              <Pencil className="h-4 w-4" aria-hidden />
            </button>
          </div>
          {portfolio.introText && (
            <p className="max-w-2xl text-sm text-muted-foreground">{portfolio.introText}</p>
          )}
        </div>
      }
      actions={
        <PortfolioPublishButton portfolio={portfolio} onToggle={() => void toggleVisibility()} />
      }
    >
      <div className="mt-5 space-y-5 sm:mt-6">
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
                  className="inline-flex items-center gap-1.5 rounded-lg border border-slate-200 bg-white px-3 py-1.5 text-xs font-medium text-slate-700 transition-[background-color,color,border-color] hover:bg-slate-50 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-300 dark:hover:bg-neutral-800"
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
            <AddArtifactMenu
              onSelect={onAddArtifactKind}
              disabled={anyModalBusy}
              dragHandlesVisible={dragHandlesVisible}
              onToggleDragHandles={() => setDragHandlesVisible((v) => !v)}
              artifactListActionsEnabled={artifacts.length > 0}
            />
          </div>

          {artifacts.length > 0 && (
            <DndContext
              sensors={sensors}
              collisionDetection={closestCenter}
              onDragStart={handleDragStart}
              onDragEnd={handleDragEnd}
              onDragCancel={handleDragCancel}
              accessibility={{
                announcements: {
                  onDragStart({ active }) {
                    const item = artifacts.find((a) => a.id === String(active.id))
                    if (!item) return
                    const pos = artifacts.findIndex((a) => a.id === String(active.id)) + 1
                    return `Picked up "${item.title}". Current position: ${pos} of ${artifacts.length}. Press arrow keys to move, Space to drop, Escape to cancel.`
                  },
                  onDragOver() {
                    return undefined
                  },
                  onDragEnd({ active, over }) {
                    const item = artifacts.find((a) => a.id === String(active.id))
                    if (!item) return
                    if (!over || active.id === over.id) {
                      return `Dragging cancelled. "${item.title}" returned to its original position.`
                    }
                    const pos = artifacts.findIndex((a) => a.id === String(over.id)) + 1
                    return `"${item.title}" dropped at position ${pos} of ${artifacts.length}.`
                  },
                  onDragCancel({ active }) {
                    const item = artifacts.find((a) => a.id === String(active.id))
                    return `Dragging cancelled. "${item?.title ?? 'Item'}" returned to its original position.`
                  },
                },
              }}
            >
              <SortableContext
                id={PORTFOLIO_ARTIFACTS_SORT_ID}
                items={artifactIds}
                strategy={verticalListSortingStrategy}
              >
                <ul className="mt-4 divide-y divide-slate-200/55 border-t border-slate-200/55 dark:divide-neutral-700/80 dark:border-neutral-700/80">
                  {artifacts.map((a) => (
                    <SortableArtifactRow
                      key={a.id}
                      artifact={a}
                      pid={pid}
                      evaluations={evaluations}
                      disabled={anyModalBusy}
                      dragHandlesVisible={dragHandlesVisible || anyModalBusy}
                      onTogglePublished={() => void toggleArtifactPublic(a)}
                      onEditTitle={
                        isPortfolioHeading(a) || isPortfolioContentPage(a)
                          ? () => openEditTitle(a)
                          : undefined
                      }
                      onDelete={() => requestDeleteArtifact(a)}
                    />
                  ))}
                </ul>
              </SortableContext>
              {activeDragId && activeDragArtifact ? (
                <DragOverlay dropAnimation={null}>
                  <div className="pointer-events-none max-w-lg rounded-xl border border-slate-300 bg-white px-3 py-2 shadow-lg dark:border-neutral-600 dark:bg-neutral-800">
                    <p className="text-sm font-semibold text-slate-950 dark:text-neutral-100">
                      {activeDragArtifact.title}
                    </p>
                    <p className="text-xs text-slate-500 dark:text-neutral-400">
                      {isPortfolioHeading(activeDragArtifact)
                        ? 'Heading'
                        : isPortfolioContentPage(activeDragArtifact)
                          ? 'Content page'
                          : activeDragArtifact.artifactType === 'url'
                            ? 'External link'
                            : artifactTypeLabel(activeDragArtifact)}
                    </p>
                  </div>
                </DragOverlay>
              ) : null}
            </DndContext>
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

      <ModuleNameModal
        key={`portfolio-heading-${headingModalKey}`}
        open={headingModalOpen}
        onClose={() => {
          if (!headingSaving) setHeadingModalOpen(false)
        }}
        onSave={(title) => void saveNewHeading(title)}
        saving={headingSaving}
        errorMessage={headingSaveError}
        mode="heading"
      />

      <ModuleNameModal
        key={`portfolio-content-page-${contentPageModalKey}`}
        open={contentPageModalOpen}
        onClose={() => {
          if (!contentPageSaving) setContentPageModalOpen(false)
        }}
        onSave={(title) => void saveNewContentPage(title)}
        saving={contentPageSaving}
        errorMessage={contentPageSaveError}
        mode="content_page"
      />

      {deleteConfirmArtifact ? (
        <div
          className="fixed inset-0 z-50 flex items-end justify-center bg-slate-900/40 p-4 sm:items-center"
          role="dialog"
          aria-modal="true"
          aria-labelledby={deleteDialogTitleId}
          onClick={(e) => {
            if (
              e.target === e.currentTarget &&
              deletingArtifactId !== deleteConfirmArtifact.id
            ) {
              setDeleteConfirmArtifact(null)
            }
          }}
        >
          <div className="w-full max-w-md overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-xl dark:border-neutral-600 dark:bg-neutral-800">
            <div className="flex items-start justify-between gap-3 border-b border-slate-200 px-4 py-3 dark:border-neutral-600">
              <h3
                id={deleteDialogTitleId}
                className="text-sm font-semibold text-slate-900 dark:text-neutral-100"
              >
                Delete item
              </h3>
              <button
                type="button"
                onClick={() => {
                  if (deletingArtifactId !== deleteConfirmArtifact.id) setDeleteConfirmArtifact(null)
                }}
                disabled={deletingArtifactId === deleteConfirmArtifact.id}
                className="shrink-0 rounded-lg p-1.5 text-slate-500 hover:bg-slate-100 hover:text-slate-800 disabled:cursor-not-allowed disabled:opacity-50 dark:text-neutral-400 dark:hover:bg-neutral-700 dark:hover:text-neutral-200"
                aria-label="Close"
              >
                <X className="h-5 w-5" aria-hidden />
              </button>
            </div>
            <div className="p-4">
              <p className="text-sm leading-relaxed text-slate-600 dark:text-neutral-300">
                Remove this artifact from your portfolio? This cannot be undone.
              </p>
              {deleteConfirmArtifact.title ? (
                <p className="mt-2 text-sm font-medium text-slate-900 dark:text-neutral-100">
                  {deleteConfirmArtifact.title}
                </p>
              ) : null}
              <div className="mt-5 flex flex-wrap justify-end gap-2">
                <button
                  type="button"
                  onClick={() => setDeleteConfirmArtifact(null)}
                  disabled={deletingArtifactId === deleteConfirmArtifact.id}
                  className="rounded-xl border border-slate-200 bg-white px-4 py-2 text-sm font-medium text-slate-700 shadow-sm hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-50 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-200 dark:hover:bg-neutral-700/80"
                >
                  Cancel
                </button>
                <button
                  type="button"
                  onClick={() => void confirmDeleteArtifact()}
                  disabled={deletingArtifactId === deleteConfirmArtifact.id}
                  className="rounded-xl bg-indigo-600 px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-indigo-500 dark:hover:bg-indigo-400"
                >
                  {deletingArtifactId === deleteConfirmArtifact.id ? 'Deleting…' : 'Delete'}
                </button>
              </div>
            </div>
          </div>
        </div>
      ) : null}

      <ModuleNameModal
        key={`portfolio-edit-title-${editTitleModalKey}`}
        open={editTitleTarget !== null}
        onClose={() => {
          if (!editTitleSaving) setEditTitleTarget(null)
        }}
        onSave={(title) => void saveEditTitle(title)}
        saving={editTitleSaving}
        errorMessage={editTitleError}
        mode={
          editTitleTarget && isPortfolioHeading(editTitleTarget)
            ? 'heading'
            : 'content_page'
        }
        initialTitle={editTitleTarget?.title ?? ''}
        dialogTitleOverride="Edit title"
        submitLabelOverride="Save title"
      />

      <ModuleNameModal
        key={`portfolio-rename-${portfolioTitleModalKey}`}
        open={portfolioTitleModalOpen}
        onClose={() => {
          if (!portfolioTitleSaving) setPortfolioTitleModalOpen(false)
        }}
        onSave={(title) => void savePortfolioTitle(title)}
        saving={portfolioTitleSaving}
        errorMessage={portfolioTitleError}
        mode="portfolio"
        initialTitle={portfolio.title}
      />
    </LmsPage>
  )
}
