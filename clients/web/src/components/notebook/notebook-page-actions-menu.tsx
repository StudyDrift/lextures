import { ChevronDown, ChevronRight, Folder, Sparkles } from 'lucide-react'
import { useMemo, useState, type RefObject } from 'react'
import {
  isNotebookGroup,
  notebookGroupMoveTargets,
  notebookPagePathLabel,
  type CourseNotebookPage,
} from '../../lib/course-notebook-tree'

export type NotebookPageActionsMenuProps = {
  open: boolean
  onToggle: () => void
  onClose: () => void
  menuRef: RefObject<HTMLDivElement | null>
  pages: CourseNotebookPage[]
  activePage: CourseNotebookPage
  onMoveToGroup: (pageId: string, groupId: string) => void
  onMoveToRoot: (pageId: string) => void
  onFlashcards: () => void
  flashcardsEnabled?: boolean
}

export function NotebookPageActionsMenu({
  open,
  onToggle,
  onClose,
  menuRef,
  pages,
  activePage,
  onMoveToGroup,
  onMoveToRoot,
  onFlashcards,
  flashcardsEnabled = true,
}: NotebookPageActionsMenuProps) {
  const [moveOpen, setMoveOpen] = useState(false)
  const moveTargets = useMemo(
    () => notebookGroupMoveTargets(pages, activePage.id),
    [pages, activePage.id],
  )
  const canMoveToRoot = activePage.parentId !== null
  const currentParent = activePage.parentId
    ? pages.find((p) => p.id === activePage.parentId)
    : null

  function closeAll() {
    setMoveOpen(false)
    onClose()
  }

  return (
    <div className="relative shrink-0" ref={menuRef}>
      <button
        type="button"
        onClick={onToggle}
        className="inline-flex items-center gap-1.5 rounded-lg border border-slate-200 bg-white px-3 py-1.5 text-xs font-medium text-slate-700 shadow-sm transition-[background-color,color,border-color] hover:bg-slate-50 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-300 dark:hover:bg-neutral-800"
        aria-expanded={open}
        aria-haspopup="menu"
      >
        Actions
        <ChevronDown className="h-3.5 w-3.5" aria-hidden />
      </button>
      {open ? (
        <div
          role="menu"
          className="absolute right-0 top-full z-20 mt-1 w-56 rounded-xl border border-slate-200 bg-white py-1 shadow-lg dark:border-neutral-700 dark:bg-neutral-900"
        >
          {flashcardsEnabled ? (
            <button
              type="button"
              role="menuitem"
              onClick={() => {
                closeAll()
                onFlashcards()
              }}
              className="flex w-full items-center gap-2 px-3 py-2 text-sm text-slate-700 transition-[background-color,color,border-color] hover:bg-slate-50 dark:text-neutral-300 dark:hover:bg-neutral-800"
            >
              <Sparkles className="h-4 w-4 text-indigo-500" aria-hidden />
              Create Flash Cards
            </button>
          ) : null}

          {moveTargets.length > 0 || canMoveToRoot ? (
            <>
              {flashcardsEnabled ? (
                <div className="my-1 border-t border-slate-100 dark:border-neutral-800" role="separator" />
              ) : null}
              <div className="relative">
                <button
                  type="button"
                  role="menuitem"
                  aria-expanded={moveOpen}
                  aria-haspopup="menu"
                  onClick={() => setMoveOpen((v) => !v)}
                  className="flex w-full items-center gap-2 px-3 py-2 text-sm text-slate-700 transition-[background-color,color,border-color] hover:bg-slate-50 dark:text-neutral-300 dark:hover:bg-neutral-800"
                >
                  <Folder className="h-4 w-4 text-amber-500 dark:text-amber-400" aria-hidden />
                  <span className="flex-1 text-start">Move to group</span>
                  <ChevronRight className="h-3.5 w-3.5 text-slate-400" aria-hidden />
                </button>
                {moveOpen ? (
                  <div
                    role="menu"
                    className="absolute right-full top-0 z-30 me-1 max-h-64 w-60 overflow-y-auto rounded-xl border border-slate-200 bg-white py-1 shadow-lg dark:border-neutral-700 dark:bg-neutral-900"
                  >
                    {moveTargets.length === 0 ? (
                      <p className="px-3 py-2 text-xs text-slate-500 dark:text-neutral-400">
                        No groups available.
                      </p>
                    ) : (
                      moveTargets.map((group) => {
                        const selected = currentParent?.id === group.id
                        return (
                          <button
                            key={group.id}
                            type="button"
                            role="menuitem"
                            disabled={selected}
                            onClick={() => {
                              closeAll()
                              onMoveToGroup(activePage.id, group.id)
                            }}
                            className="flex w-full flex-col items-start gap-0.5 px-3 py-2 text-start text-sm transition-[background-color,color,border-color] hover:bg-slate-50 disabled:cursor-default disabled:opacity-50 dark:hover:bg-neutral-800"
                          >
                            <span className="font-medium text-slate-800 dark:text-neutral-100">
                              {group.title || 'Untitled group'}
                              {selected ? ' (current)' : ''}
                            </span>
                            <span className="text-xs text-slate-500 dark:text-neutral-400">
                              {notebookPagePathLabel(pages, group.id)}
                            </span>
                          </button>
                        )
                      })
                    )}
                  </div>
                ) : null}
              </div>
              {canMoveToRoot ? (
                <button
                  type="button"
                  role="menuitem"
                  onClick={() => {
                    closeAll()
                    onMoveToRoot(activePage.id)
                  }}
                  className="flex w-full items-center gap-2 px-3 py-2 text-sm text-slate-700 transition-[background-color,color,border-color] hover:bg-slate-50 dark:text-neutral-300 dark:hover:bg-neutral-800"
                >
                  Move to top level
                </button>
              ) : null}
            </>
          ) : null}

          {isNotebookGroup(activePage) && moveTargets.length === 0 && !canMoveToRoot ? (
            <p className="px-3 py-2 text-xs text-slate-500 dark:text-neutral-400">
              Create a group in the sidebar to organize pages.
            </p>
          ) : null}
        </div>
      ) : null}
    </div>
  )
}
