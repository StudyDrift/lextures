import { useId, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { MoreHorizontal, RefreshCw } from 'lucide-react'
import { ChecklistStatusAffordance } from '../../../components/checklist/checklist-status-affordance'
import type { ChecklistItem } from '../../../lib/course-checklist-api-schemas'
import {
  isOutstandingStatus,
  normalizeChecklistStatus,
} from '../../../lib/course-checklist-api-schemas'
import { courseChecklistI18n } from '../../../lib/course-checklist-i18n'
import { ChecklistEvidenceTable } from './checklist-evidence-table'

type ChecklistItemRowProps = {
  item: ChecklistItem
  busy?: boolean
  error?: string | null
  onDismiss: (item: ChecklistItem) => void
  onRecheck: (item: ChecklistItem) => void
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
  highlighted,
}: ChecklistItemRowProps) {
  const navigate = useNavigate()
  const status = normalizeChecklistStatus(item.status)
  const done = status === 'done'
  const unknown = status === 'unknown'
  const mode = interactionMode(item)
  const evidenceId = useId()
  const whyId = useId()
  const [expanded, setExpanded] = useState(false)
  const [whyOpen, setWhyOpen] = useState(false)
  const [menuOpen, setMenuOpen] = useState(false)
  const interactive = isOutstandingStatus(item.status) || unknown

  const titleClass = done
    ? 'line-through text-slate-500 dark:text-neutral-500'
    : unknown
      ? 'text-slate-500 dark:text-neutral-400'
      : 'text-slate-900 dark:text-neutral-50'

  const titleNode = (
    <span className={`text-sm font-medium ${titleClass}`}>
      {item.title}
      {done ? <span className="sr-only"> {courseChecklistI18n.completedLabel}</span> : null}
    </span>
  )

  return (
    <li
      id={`item-${item.id}`}
      className={`rounded-lg border border-transparent px-2 py-3 transition-colors ${
        highlighted ? 'border-amber-300 bg-amber-50/60 dark:border-amber-700 dark:bg-amber-950/30' : ''
      } ${unknown ? 'opacity-80' : ''}`}
    >
      <div className="flex items-start gap-3">
        <div className="mt-0.5">
          <ChecklistStatusAffordance status={item.status} />
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-start justify-between gap-2">
            <div className="min-w-0 flex-1">
              {mode === 'link' && interactive && item.target?.route ? (
                <Link
                  to={item.target.route}
                  className="inline-flex min-h-11 items-center hover:underline"
                >
                  {titleNode}
                </Link>
              ) : mode === 'evidence' && interactive ? (
                <button
                  type="button"
                  className="inline-flex min-h-11 items-center text-start hover:underline"
                  aria-expanded={expanded}
                  aria-controls={evidenceId}
                  onClick={() => setExpanded((v) => !v)}
                >
                  {titleNode}
                </button>
              ) : (
                <div className="inline-flex min-h-11 items-center">{titleNode}</div>
              )}
            </div>
            <div className="flex shrink-0 items-center gap-2">
              {item.progress ? (
                <span className="text-xs tabular-nums text-slate-500 dark:text-neutral-400">
                  {item.progress.done} / {item.progress.total}
                </span>
              ) : null}
              {item.tier === 'essential' ? (
                <span className="rounded bg-amber-100 px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-amber-900 dark:bg-amber-950 dark:text-amber-200">
                  {courseChecklistI18n.essentialTier}
                </span>
              ) : null}
              {unknown ? (
                <button
                  type="button"
                  disabled={busy}
                  onClick={() => onRecheck(item)}
                  className="inline-flex min-h-11 items-center gap-1 rounded-lg border border-slate-300 px-2 text-xs font-semibold text-slate-700 dark:border-neutral-600 dark:text-neutral-200"
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
                  className="inline-flex h-11 w-11 items-center justify-center rounded-lg text-slate-500 hover:bg-slate-100 dark:hover:bg-neutral-800"
                  onClick={() => setMenuOpen((v) => !v)}
                >
                  <MoreHorizontal className="h-4 w-4" aria-hidden />
                </button>
                {menuOpen ? (
                  <div
                    role="menu"
                    className="absolute end-0 z-20 mt-1 min-w-[10rem] rounded-lg border border-slate-200 bg-white py-1 shadow-lg dark:border-neutral-700 dark:bg-neutral-900"
                  >
                    <button
                      type="button"
                      role="menuitem"
                      className="block w-full px-3 py-2 text-start text-sm text-slate-800 hover:bg-slate-50 dark:text-neutral-100 dark:hover:bg-neutral-800"
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
            className="mt-1 max-w-full truncate text-start text-xs text-slate-500 hover:text-slate-700 dark:text-neutral-400 dark:hover:text-neutral-200"
            aria-expanded={whyOpen}
            aria-controls={whyId}
            onClick={() => setWhyOpen((v) => !v)}
          >
            {whyOpen ? item.why : item.why}
          </button>
          {whyOpen ? (
            <p id={whyId} className="mt-1 text-xs text-slate-600 dark:text-neutral-400">
              {item.why}
            </p>
          ) : null}

          {item.detail || unknown ? (
            <p className="mt-1 text-sm text-slate-600 dark:text-neutral-400">
              {unknown && !item.detail ? courseChecklistI18n.unknownDetail : item.detail}
            </p>
          ) : null}

          {item.sources.length > 0 ? (
            <ul className="mt-2 flex flex-wrap gap-1.5">
              {item.sources.map((src) => (
                <li
                  key={src}
                  className="rounded bg-slate-100 px-1.5 py-0.5 text-[11px] font-medium text-slate-700 dark:bg-neutral-800 dark:text-neutral-300"
                >
                  {src}
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
                onClick={() => setExpanded((v) => !v)}
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
                    onRowNavigate={(route) => navigate(route)}
                  />
                </div>
              ) : (
                <div id={evidenceId} hidden />
              )}
            </div>
          ) : null}

          {error ? (
            <p className="mt-2 text-sm text-red-600 dark:text-red-400" role="alert">
              {error}
            </p>
          ) : null}
        </div>
      </div>
    </li>
  )
}
