import { useEffect, useId, useRef, useState } from 'react'
import {
  BookOpen,
  BookMarked,
  BookCopy,
  ChevronDown,
  CircleHelp,
  ClipboardList,
  ExternalLink,
  FileText,
  Heading,
  Plug,
  Plus,
  Puzzle,
  Sparkles,
} from 'lucide-react'

export type ModuleItemKind =
  | 'heading'
  | 'content_page'
  | 'assignment'
  | 'quiz'
  | 'external_link'
  | 'lti_link'
  | 'h5p'
  | 'vibe_activity'
  | 'library_resource'
  | 'textbook_resource'

type AddModuleItemMenuProps = {
  onAdd: (kind: ModuleItemKind) => void
  onFindOpenResources?: () => void
  oerLibraryEnabled?: boolean
  disabled?: boolean
  h5pEnabled?: boolean
  /** When false, LTI tool is shown disabled (no registered external tools). */
  ltiToolsAvailable?: boolean
  /** When true, shows the Library Resource option (HE e-reserves). */
  heLibraryEnabled?: boolean
  /** When true, shows the Textbook Resource option (bookstore / Inclusive Access). */
  bookstoreEnabled?: boolean
}

export function AddModuleItemMenu({
  onAdd,
  onFindOpenResources,
  oerLibraryEnabled = false,
  disabled,
  h5pEnabled,
  ltiToolsAvailable = true,
  heLibraryEnabled = false,
  bookstoreEnabled = false,
}: AddModuleItemMenuProps) {
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

  function pick(kind: ModuleItemKind) {
    onAdd(kind)
    setOpen(false)
  }

  return (
    <div ref={rootRef} className="relative inline-block max-w-full text-start">
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
        className="inline-flex max-w-full items-center gap-1.5 rounded-lg border border-slate-200/70 bg-white/90 px-2 py-1.5 text-xs font-medium text-slate-700 shadow-none transition hover:border-slate-300/80 hover:bg-slate-50/90 disabled:cursor-not-allowed disabled:opacity-60 sm:px-2.5 sm:text-sm dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-200 dark:hover:border-neutral-500 dark:hover:bg-neutral-800"
      >
        <Plus className="h-4 w-4 shrink-0" aria-hidden />
        <span className="truncate sm:hidden">Add item</span>
        <span className="hidden truncate sm:inline">Add module item</span>
        <ChevronDown
          className={`h-4 w-4 shrink-0 transition ${open ? 'rotate-180' : ''}`}
          aria-hidden
        />
      </button>

      {open && (
        <div
          id={menuId}
          role="menu"
          aria-label="Module item types"
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
            <span className="min-w-0 flex flex-col gap-0.5">
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
            <span className="min-w-0 flex flex-col gap-0.5">
              <span className="font-semibold text-slate-950 dark:text-neutral-100">Content page</span>
              <span className="text-xs text-slate-500 dark:text-neutral-400">Markdown page with rich formatting</span>
            </span>
          </button>
          <button
            type="button"
            role="menuitem"
            onClick={() => pick('assignment')}
            className="flex w-full items-start gap-3 border-t border-slate-100 px-2.5 py-2 text-start text-sm transition hover:bg-slate-50 dark:border-neutral-700 dark:hover:bg-neutral-700"
          >
            <span className="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-amber-200/90 bg-amber-50 text-amber-800 dark:border-amber-500/40 dark:bg-amber-950 dark:text-amber-200">
              <ClipboardList className="h-4 w-4" aria-hidden />
            </span>
            <span className="min-w-0 flex flex-col gap-0.5">
              <span className="font-semibold text-slate-950 dark:text-neutral-100">Assignment</span>
              <span className="text-xs text-slate-500 dark:text-neutral-400">Graded or submitted work</span>
            </span>
          </button>
          <button
            type="button"
            role="menuitem"
            onClick={() => pick('quiz')}
            className="flex w-full items-start gap-3 border-t border-slate-100 px-2.5 py-2 text-start text-sm transition hover:bg-slate-50 dark:border-neutral-700 dark:hover:bg-neutral-700"
          >
            <span className="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-emerald-200/90 bg-emerald-50 text-emerald-700 dark:border-emerald-500/35 dark:bg-emerald-950 dark:text-emerald-200">
              <CircleHelp className="h-4 w-4" aria-hidden />
            </span>
            <span className="min-w-0 flex flex-col gap-0.5">
              <span className="font-semibold text-slate-950 dark:text-neutral-100">Quiz</span>
              <span className="text-xs text-slate-500 dark:text-neutral-400">
                Questions and auto-graded checks
              </span>
            </span>
          </button>
          {oerLibraryEnabled && onFindOpenResources && (
            <button
              type="button"
              role="menuitem"
              onClick={() => {
                onFindOpenResources()
                setOpen(false)
              }}
              className="flex w-full items-start gap-3 border-t border-slate-100 px-2.5 py-2 text-start text-sm transition hover:bg-slate-50 dark:border-neutral-700 dark:hover:bg-neutral-700"
            >
              <span className="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-teal-200/90 bg-teal-50 text-teal-700 dark:border-teal-500/40 dark:bg-teal-950 dark:text-teal-200">
                <BookOpen className="h-4 w-4" aria-hidden />
              </span>
              <span className="min-w-0 flex flex-col gap-0.5">
                <span className="font-semibold text-slate-950 dark:text-neutral-100">Find open resources</span>
                <span className="text-xs text-slate-500 dark:text-neutral-400">
                  Search OER Commons, MERLOT, and OpenStax
                </span>
              </span>
            </button>
          )}
          <button
            type="button"
            role="menuitem"
            onClick={() => pick('external_link')}
            className="flex w-full items-start gap-3 border-t border-slate-100 px-2.5 py-2 text-start text-sm transition hover:bg-slate-50 dark:border-neutral-700 dark:hover:bg-neutral-700"
          >
            <span className="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-violet-200/90 bg-violet-50 text-violet-700 dark:border-violet-500/40 dark:bg-violet-950 dark:text-violet-200">
              <ExternalLink className="h-4 w-4" aria-hidden />
            </span>
            <span className="min-w-0 flex flex-col gap-0.5">
              <span className="font-semibold text-slate-950 dark:text-neutral-100">External link</span>
              <span className="text-xs text-slate-500 dark:text-neutral-400">
                Opens a URL in a new tab
              </span>
            </span>
          </button>
          {h5pEnabled ? (
            <button
              type="button"
              role="menuitem"
              onClick={() => pick('h5p')}
              className="flex w-full items-start gap-3 border-t border-slate-100 px-2.5 py-2 text-start text-sm transition hover:bg-slate-50 dark:border-neutral-700 dark:hover:bg-neutral-700"
            >
              <span className="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-teal-200/90 bg-teal-50 text-teal-800 dark:border-teal-500/40 dark:bg-teal-950 dark:text-teal-200">
                <Puzzle className="h-4 w-4" aria-hidden />
              </span>
              <span className="min-w-0 flex flex-col gap-0.5">
                <span className="font-semibold text-slate-950 dark:text-neutral-100">Interactive H5P</span>
                <span className="text-xs text-slate-500 dark:text-neutral-400">
                  Upload an interactive .h5p activity
                </span>
              </span>
            </button>
          ) : null}
          <button
            type="button"
            role="menuitem"
            onClick={() => pick('vibe_activity')}
            className="flex w-full items-start gap-3 border-t border-slate-100 px-2.5 py-2 text-start text-sm transition hover:bg-slate-50 dark:border-neutral-700 dark:hover:bg-neutral-700"
          >
            <span className="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-rose-200/90 bg-rose-50 text-rose-700 dark:border-rose-500/40 dark:bg-rose-950 dark:text-rose-200">
              <Sparkles className="h-4 w-4" aria-hidden />
            </span>
            <span className="min-w-0 flex flex-col gap-0.5">
              <span className="font-semibold text-slate-950 dark:text-neutral-100">Vibe Activity</span>
              <span className="text-xs text-slate-500 dark:text-neutral-400">
                AI-assisted interactive HTML web activity
              </span>
            </span>
          </button>
          {heLibraryEnabled && (
            <button
              type="button"
              role="menuitem"
              onClick={() => pick('library_resource')}
              className="flex w-full items-start gap-3 border-t border-slate-100 px-2.5 py-2 text-start text-sm transition hover:bg-slate-50 dark:border-neutral-700 dark:hover:bg-neutral-700"
            >
              <span className="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-cyan-200/90 bg-cyan-50 text-cyan-700 dark:border-cyan-500/40 dark:bg-cyan-950 dark:text-cyan-200">
                <BookMarked className="h-4 w-4" aria-hidden />
              </span>
              <span className="min-w-0 flex flex-col gap-0.5">
                <span className="font-semibold text-slate-950 dark:text-neutral-100">Library Resource</span>
                <span className="text-xs text-slate-500 dark:text-neutral-400">
                  Alma catalog item or Leganto reading list
                </span>
              </span>
            </button>
          )}
          {bookstoreEnabled && (
            <button
              type="button"
              role="menuitem"
              onClick={() => pick('textbook_resource')}
              className="flex w-full items-start gap-3 border-t border-slate-100 px-2.5 py-2 text-start text-sm transition hover:bg-slate-50 dark:border-neutral-700 dark:hover:bg-neutral-700"
            >
              <span className="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-orange-200/90 bg-orange-50 text-orange-700 dark:border-orange-500/40 dark:bg-orange-950 dark:text-orange-200">
                <BookCopy className="h-4 w-4" aria-hidden />
              </span>
              <span className="min-w-0 flex flex-col gap-0.5">
                <span className="font-semibold text-slate-950 dark:text-neutral-100">Textbook Resource</span>
                <span className="text-xs text-slate-500 dark:text-neutral-400">
                  VitalSource or RedShelf Inclusive Access deep link
                </span>
              </span>
            </button>
          )}
          <button
            type="button"
            role="menuitem"
            disabled={!ltiToolsAvailable}
            onClick={() => {
              if (!ltiToolsAvailable) return
              pick('lti_link')
            }}
            className="flex w-full items-start gap-3 border-t border-slate-100 px-2.5 py-2 text-start text-sm transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60 dark:border-neutral-700 dark:hover:bg-neutral-700"
          >
            <span className="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-purple-200/90 bg-purple-50 text-purple-800 dark:border-purple-500/40 dark:bg-purple-950 dark:text-purple-200">
              <Plug className="h-4 w-4" aria-hidden />
            </span>
            <span className="min-w-0 flex flex-col gap-0.5">
              <span className="font-semibold text-slate-950 dark:text-neutral-100">LTI tool</span>
              <span className="text-xs text-slate-500 dark:text-neutral-400">
                {ltiToolsAvailable
                  ? 'Embedded publisher or external LTI 1.3 tool'
                  : 'No LTI tools registered — add under Settings → LTI tools'}
              </span>
            </span>
          </button>
        </div>
      )}
    </div>
  )
}
