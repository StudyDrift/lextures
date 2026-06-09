export type CourseCatalogView = 'cards' | 'list' | 'status' | 'gallery' | 'table'

export type KanbanColumnId = 'todo' | 'in-progress' | 'done' | 'hidden'

export const KANBAN_COLUMN_IDS: KanbanColumnId[] = ['todo', 'in-progress', 'done', 'hidden']

export function isKanbanColumnId(value: string | null | undefined): value is KanbanColumnId {
  return value === 'todo' || value === 'in-progress' || value === 'done' || value === 'hidden'
}

export type KanbanColumnLabels = Record<KanbanColumnId, string>

export const DEFAULT_KANBAN_COLUMN_LABELS: KanbanColumnLabels = {
  todo: 'Todo',
  'in-progress': 'In progress',
  done: 'Done',
  hidden: 'Hidden',
}
