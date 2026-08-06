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
        className="inline-flex items-center gap-1.5 rounded-lg border border-border-default bg-surface-raised px-3 py-1.5 text-xs font-medium text-fg-muted shadow-sm transition-[background-color,color,border-color] hover:bg-surface-base dark:border-border-default dark:bg-surface-raised dark:text-fg-muted dark:hover:bg-surface-overlay"
        aria-expanded={open}
        aria-haspopup="menu"
      >
        Actions
        <ChevronDown className="h-3.5 w-3.5" aria-hidden />
      </button>
      {open ? (
        <div
          role="menu"
          className="absolute right-0 top-full z-20 mt-1 w-56 rounded-xl border border-border-default bg-surface-raised py-1 shadow-lg dark:border-border-default dark:bg-surface-raised"
        >
          {flashcardsEnabled ? (
            <button
              type="button"
              role="menuitem"
              onClick={() => {
                closeAll()
                onFlashcards()
              }}
              className="flex w-full items-center gap-2 px-3 py-2 text-sm text-fg-muted transition-[background-color,color,border-color] hover:bg-surface-base dark:text-fg-muted dark:hover:bg-surface-overlay"
            >
              <Sparkles className="h-4 w-4 text-indigo-500" aria-hidden />
              Create Flash Cards
            </button>
          ) : null}

          {moveTargets.length > 0 || canMoveToRoot ? (
            <>
              {flashcardsEnabled ? (
                <div className="my-1 border-t border-border-subtle" role="separator" />
              ) : null}
              <div className="relative">
                <button
                  type="button"
                  role="menuitem"
                  aria-expanded={moveOpen}
                  aria-haspopup="menu"
                  onClick={() => setMoveOpen((v) => !v)}
                  className="flex w-full items-center gap-2 px-3 py-2 text-sm text-fg-muted transition-[background-color,color,border-color] hover:bg-surface-base dark:text-fg-muted dark:hover:bg-surface-overlay"
                >
                  <Folder className="h-4 w-4 text-amber-500 dark:text-amber-400" aria-hidden />
                  <span className="flex-1 text-start">Move to group</span>
                  <ChevronRight className="h-3.5 w-3.5 text-fg-subtle" aria-hidden />
                </button>
                {moveOpen ? (
                  <div
                    role="menu"
                    className="absolute right-full top-0 z-30 me-1 max-h-64 w-60 overflow-y-auto rounded-xl border border-border-default bg-surface-raised py-1 shadow-lg dark:border-border-default dark:bg-surface-raised"
                  >
                    {moveTargets.length === 0 ? (
                      <p className="px-3 py-2 text-xs text-fg-muted">
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
                            className="flex w-full flex-col items-start gap-0.5 px-3 py-2 text-start text-sm transition-[background-color,color,border-color] hover:bg-surface-base disabled:cursor-default disabled:opacity-50 dark:hover:bg-surface-overlay"
                          >
                            <span className="font-medium text-fg-default">
                              {group.title || 'Untitled group'}
                              {selected ? ' (current)' : ''}
                            </span>
                            <span className="text-xs text-fg-muted">
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
                  className="flex w-full items-center gap-2 px-3 py-2 text-sm text-fg-muted transition-[background-color,color,border-color] hover:bg-surface-base dark:text-fg-muted dark:hover:bg-surface-overlay"
                >
                  Move to top level
                </button>
              ) : null}
            </>
          ) : null}

          {isNotebookGroup(activePage) && moveTargets.length === 0 && !canMoveToRoot ? (
            <p className="px-3 py-2 text-xs text-fg-muted">
              Create a group in the sidebar to organize pages.
            </p>
          ) : null}
        </div>
      ) : null}
    </div>
  )
}
