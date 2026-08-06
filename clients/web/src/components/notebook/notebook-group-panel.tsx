import { FileText, Folder, Plus } from 'lucide-react'
import {
  isNotebookGroup,
  sortedChildren,
  type CourseNotebookPage,
} from '../../lib/course-notebook-tree'

type NotebookGroupPanelProps = {
  group: CourseNotebookPage
  pages: CourseNotebookPage[]
  onSelectPage: (id: string) => void
  onAddPage: (parentId: string) => void
  onAddGroup: (parentId: string) => void
}

export function NotebookGroupPanel({
  group,
  pages,
  onSelectPage,
  onAddPage,
  onAddGroup,
}: NotebookGroupPanelProps) {
  const children = sortedChildren(pages, group.id)

  return (
    <div className="mx-auto flex min-h-0 w-full max-w-[72ch] flex-1 flex-col px-4 py-6 md:px-6 md:py-8">
      <div className="rounded-xl border border-dashed border-border-default bg-slate-50/80 p-6 dark:border-border-default/40">
        <div className="flex items-start gap-3">
          <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-indigo-100 text-accent-fg dark:bg-indigo-950/60 dark:text-indigo-300">
            <Folder className="h-5 w-5" aria-hidden />
          </div>
          <div className="min-w-0 flex-1">
            <p className="text-sm font-medium text-fg-default">Page group</p>
            <p className="mt-1 text-sm text-fg-muted">
              Groups organize nested pages in the sidebar. Add pages or subgroups below, or use the
              buttons in the sidebar tree.
            </p>
            <div className="mt-4 flex flex-wrap gap-2">
              <button
                type="button"
                onClick={() => onAddPage(group.id)}
                className="inline-flex items-center gap-1.5 rounded-lg border border-border-default bg-surface-raised px-3 py-1.5 text-sm font-medium text-fg-muted shadow-sm transition-[background-color,color,border-color] hover:bg-surface-base dark:border-border-default dark:bg-surface-raised dark:text-fg-default dark:hover:bg-surface-overlay"
              >
                <Plus className="h-4 w-4" aria-hidden />
                Add page
              </button>
              <button
                type="button"
                onClick={() => onAddGroup(group.id)}
                className="inline-flex items-center gap-1.5 rounded-lg border border-border-default bg-surface-raised px-3 py-1.5 text-sm font-medium text-fg-muted shadow-sm transition-[background-color,color,border-color] hover:bg-surface-base dark:border-border-default dark:bg-surface-raised dark:text-fg-default dark:hover:bg-surface-overlay"
              >
                <Plus className="h-4 w-4" aria-hidden />
                Add subgroup
              </button>
            </div>
          </div>
        </div>
      </div>

      {children.length > 0 ? (
        <div className="mt-6">
          <h3 className="text-xs font-semibold uppercase tracking-wide text-fg-muted">
            In this group
          </h3>
          <ul className="mt-2 flex flex-col gap-1">
            {children.map((child) => (
              <li key={child.id}>
                <button
                  type="button"
                  onClick={() => onSelectPage(child.id)}
                  className="flex w-full items-center gap-2 rounded-lg px-3 py-2 text-start text-sm text-fg-muted transition-[background-color,color,border-color] hover:bg-surface-sunken dark:text-fg-default dark:hover:bg-neutral-800/80"
                >
                  {isNotebookGroup(child) ? (
                    <Folder className="h-4 w-4 shrink-0 text-amber-500 dark:text-amber-400" aria-hidden />
                  ) : (
                    <FileText className="h-4 w-4 shrink-0 text-fg-subtle" aria-hidden />
                  )}
                  <span className="min-w-0 flex-1 truncate font-medium">{child.title || 'Untitled'}</span>
                </button>
              </li>
            ))}
          </ul>
        </div>
      ) : (
        <p className="mt-6 text-sm text-fg-muted">
          No pages in this group yet. Add one with the buttons above.
        </p>
      )}
    </div>
  )
}
