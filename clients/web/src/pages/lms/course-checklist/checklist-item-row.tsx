import { useId, useMemo, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { CircleHelp, MoreHorizontal, RefreshCw } from 'lucide-react'
import { ChecklistStatusAffordance } from '../../../components/checklist/checklist-status-affordance'
import type { ChecklistItem } from '../../../lib/course-checklist-api-schemas'
import {
  isOutstandingStatus,
  normalizeChecklistStatus,
} from '../../../lib/course-checklist-api-schemas'
import { courseChecklistI18n } from '../../../lib/course-checklist-i18n'
import { resolveChecklistHelp } from '../../../lib/checklist-help'
import { courseDesignResearchHref } from '../../../lib/checklist-research-anchors'
import { emitChecklistTelemetry } from '../../../lib/checklist-telemetry'
import { hrefForTarget } from '../../../lib/use-focus-anchor'
import { ChecklistEvidenceTable } from './checklist-evidence-table'
import { ChecklistHelpPopover } from './checklist-help-popover'
import { ChecklistResearchDialog } from './checklist-research-dialog'

type ChecklistItemRowProps = {
  item: ChecklistItem
  busy?: boolean
  error?: string | null
  onDismiss: (item: ChecklistItem) => void
  onRecheck: (item: ChecklistItem) => void
  /** Assisted-fix click; parent owns AI opt-out and routing. */
  onAssist?: (item: ChecklistItem) => void
  /** When true, AI-required actions are hidden (opt-out / AI unavailable). */
  hideAiActions?: boolean
  highlighted?: boolean
}

function interactionMode(item: ChecklistItem): 'evidence' | 'link' | 'static' {
  if (item.evidence && item.evidence.rows.length >= 1) return 'evidence'
  if (item.target?.route) return 'link'
  return 'static'
}

export function ChecklistItemRow({
  item,
  busy,
  error,
  onDismiss,
  onRecheck,
  onAssist,
  hideAiActions,
  highlighted,
}: ChecklistItemRowProps) {
  const navigate = useNavigate()
  const { courseCode = '' } = useParams<{ courseCode: string }>()
  const status = normalizeChecklistStatus(item.status)
  const done = status === 'done'
  const unknown = status === 'unknown'
  const mode = interactionMode(item)
  const evidenceId = useId()
  const whyId = useId()
  const [expanded, setExpanded] = useState(false)
  const [whyOpen, setWhyOpen] = useState(false)
  const [menuOpen, setMenuOpen] = useState(false)
  const [helpOpen, setHelpOpen] = useState(false)
  const [researchOpen, setResearchOpen] = useState(false)
  const [researchSource, setResearchSource] = useState<string | null>(null)
  const interactive = isOutstandingStatus(item.status) || unknown
  const hasHelp = !!resolveChecklistHelp(item.helpRef)
  const showAction =
    !!item.action &&
    interactive &&
    !(item.action.requiresAi && hideAiActions) &&
    !!onAssist
  const targetHref = useMemo(
    () =>
      hrefForTarget(
        {
          route: item.target?.route,
          anchor: item.target?.anchor,
          entityKey: item.target?.entityKey,
        },
        courseCode ? { courseCode } : undefined,
      ),
    [item.target, courseCode],
  )

  const trackNavigate = (resolved: boolean) => {
    emitChecklistTelemetry('checklist_target_navigated', {
      itemId: item.id,
      anchorId: item.target?.anchor ?? undefined,
      resolved,
    })
  }

  const titleClass = done
    ? 'line-through text-fg-subtle'
    : unknown
      ? 'text-fg-muted'
      : 'text-fg-default'

  const titleNode = (
    <span className={`text-sm font-medium ${titleClass}`}>
      {item.title}
      {done ? <span className="sr-only"> {courseChecklistI18n.completedLabel}</span> : null}
    </span>
  )

  return (
    <li
      id={`item-${item.id}`}
      className={`rounded-lg border border-transparent px-2 py-3 transition-colors ${ highlighted ? 'border-amber-300 bg-amber-50/60 dark:border-amber-700 dark:bg-amber-950/30' : '' } ${unknown ? 'opacity-80' : ''}`}
    >
      <div className="flex items-start gap-3">
        <div className="mt-0.5">
          <ChecklistStatusAffordance status={item.status} />
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-start justify-between gap-2">
            <div className="min-w-0 flex-1">
              {mode === 'link' && interactive && targetHref ? (
                <Link
                  to={targetHref}
                  className="inline-flex min-h-11 items-center hover:underline"
                  onClick={() => trackNavigate(true)}
                >
                  {titleNode}
                </Link>
              ) : mode === 'evidence' && interactive ? (
                <button
                  type="button"
                  className="inline-flex min-h-11 items-center text-start hover:underline"
                  aria-expanded={expanded}
                  aria-controls={evidenceId}
                  onClick={() => {
                    setExpanded((v) => {
                      const next = !v
                      if (next) emitChecklistTelemetry('checklist_item_expanded', { itemId: item.id })
                      return next
                    })
                  }}
                >
                  {titleNode}
                </button>
              ) : (
                <div className="inline-flex min-h-11 items-center">{titleNode}</div>
              )}
            </div>
            <div className="flex shrink-0 items-center gap-2">
              {item.progress ? (
                <span className="text-xs tabular-nums text-fg-muted">
                  {item.progress.done} / {item.progress.total}
                </span>
              ) : null}
              {item.tier === 'essential' ? (
                <span className="rounded bg-amber-100 px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-amber-900 dark:bg-amber-950 dark:text-amber-200">
                  {courseChecklistI18n.essentialTier}
                </span>
              ) : null}
              {hasHelp ? (
                <button
                  type="button"
                  className="inline-flex h-11 w-11 items-center justify-center rounded-lg text-fg-muted hover:bg-surface-sunken dark:hover:bg-surface-overlay"
                  aria-label={courseChecklistI18n.aboutThisCheck}
                  onClick={() => setHelpOpen(true)}
                >
                  <CircleHelp className="h-4 w-4" aria-hidden />
                </button>
              ) : null}
              {showAction && item.action ? (
                <button
                  type="button"
                  disabled={busy}
                  onClick={() => onAssist?.(item)}
                  className="inline-flex min-h-11 items-center rounded-lg border border-amber-300 px-2 text-xs font-semibold text-amber-900 dark:border-amber-800 dark:text-amber-200"
                >
                  {item.action.label}
                </button>
              ) : null}
              {unknown ? (
                <button
                  type="button"
                  disabled={busy}
                  onClick={() => onRecheck(item)}
                  className="inline-flex min-h-11 items-center gap-1 rounded-lg border border-border-strong px-2 text-xs font-semibold text-fg-muted dark:border-border-default dark:text-fg-default"
                >
                  <RefreshCw className="h-3.5 w-3.5" aria-hidden />
                  {courseChecklistI18n.recheck}
                </button>
              ) : null}
              <div className="relative">
                <button
                  type="button"
                  aria-label={courseChecklistI18n.overflowMenu}
                  aria-haspopup="menu"
                  aria-expanded={menuOpen}
                  className="inline-flex h-11 w-11 items-center justify-center rounded-lg text-fg-muted hover:bg-surface-sunken dark:hover:bg-surface-overlay"
                  onClick={() => setMenuOpen((v) => !v)}
                >
                  <MoreHorizontal className="h-4 w-4" aria-hidden />
                </button>
                {menuOpen ? (
                  <div
                    role="menu"
                    className="absolute end-0 z-20 mt-1 min-w-[10rem] rounded-lg border border-border-default bg-surface-raised py-1 shadow-lg dark:border-border-default dark:bg-surface-raised"
                  >
                    <button
                      type="button"
                      role="menuitem"
                      className="block w-full px-3 py-2 text-start text-sm text-fg-default hover:bg-surface-base dark:text-fg-default dark:hover:bg-surface-overlay"
                      onClick={() => {
                        setMenuOpen(false)
                        onDismiss(item)
                      }}
                    >
                      {courseChecklistI18n.dismiss}
                    </button>
                  </div>
                ) : null}
              </div>
            </div>
          </div>

          <button
            type="button"
            className="mt-1 max-w-full truncate text-start text-xs text-fg-muted hover:text-fg-muted dark:text-fg-muted dark:hover:text-fg-default"
            aria-expanded={whyOpen}
            aria-controls={whyId}
            onClick={() => setWhyOpen((v) => !v)}
          >
            {whyOpen ? item.why : item.why}
          </button>
          {whyOpen ? (
            <p id={whyId} className="mt-1 text-xs text-fg-muted">
              {item.why}
            </p>
          ) : null}

          {item.detail || unknown ? (
            <p className="mt-1 text-sm text-fg-muted">
              {unknown && !item.detail ? courseChecklistI18n.unknownDetail : item.detail}
            </p>
          ) : null}

          {item.sources.length > 0 ? (
            <ul className="mt-2 flex flex-wrap gap-1.5">
              {item.sources.map((src) => (
                <li key={src}>
                  <a
                    href={courseDesignResearchHref(src)}
                    className="rounded bg-surface-sunken px-1.5 py-0.5 text-[11px] font-medium text-fg-muted underline-offset-2 hover:underline dark:bg-surface-overlay dark:text-fg-muted"
                    onClick={(e) => {
                      // Stay on the checklist: open the research dialog scrolled to this standard.
                      e.preventDefault()
                      setResearchSource(src)
                      setResearchOpen(true)
                    }}
                  >
                    {src}
                  </a>
                </li>
              ))}
            </ul>
          ) : null}

          {mode === 'evidence' && interactive ? (
            <div className="mt-2">
              <button
                type="button"
                className="inline-flex min-h-11 items-center text-xs font-semibold text-amber-800 dark:text-amber-300"
                aria-expanded={expanded}
                aria-controls={evidenceId}
                onClick={() => {
                  setExpanded((v) => {
                    const next = !v
                    if (next) {
                      emitChecklistTelemetry('checklist_item_expanded', { itemId: item.id })
                      emitChecklistTelemetry('checklist_evidence_clicked', { itemId: item.id })
                    }
                    return next
                  })
                }}
              >
                {expanded
                  ? courseChecklistI18n.hideEvidence
                  : courseChecklistI18n.showEvidence(item.evidence!.rows.length)}
              </button>
              {expanded && item.evidence ? (
                <div id={evidenceId}>
                  <ChecklistEvidenceTable
                    evidence={item.evidence}
                    fallbackTarget={item.target}
                    courseCode={courseCode}
                    onRowNavigate={(route) => {
                      trackNavigate(true)
                      navigate(route)
                    }}
                  />
                </div>
              ) : (
                <div id={evidenceId} hidden />
              )}
            </div>
          ) : null}

          {error ? (
            <p className="mt-2 text-sm text-danger-fg" role="alert">
              {error}
            </p>
          ) : null}
        </div>
      </div>
      <ChecklistHelpPopover
        helpRef={item.helpRef}
        itemId={item.id}
        open={helpOpen}
        onClose={() => setHelpOpen(false)}
        sources={item.sources}
      />
      <ChecklistResearchDialog
        open={researchOpen}
        sourceLabel={researchSource}
        onClose={() => {
          setResearchOpen(false)
          setResearchSource(null)
        }}
      />
    </li>
  )
}
