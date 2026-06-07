import type { NotebookTaskContext } from '../components/editor/extensions/notebook-task-tip-tap'

let activeContext: NotebookTaskContext | null = null

export function setNotebookTaskContext(ctx: NotebookTaskContext | null): void {
  activeContext = ctx
}

export function getNotebookTaskContext(): NotebookTaskContext | null {
  return activeContext
}
