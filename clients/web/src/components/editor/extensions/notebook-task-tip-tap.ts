import { mergeAttributes, Node, type JSONContent, type MarkdownToken } from '@tiptap/core'
import { ReactNodeViewRenderer } from '@tiptap/react'
import { NotebookTaskNodeView } from './notebook-task-node-view'

export type NotebookTaskContext = {
  courseCode: string
  pageId: string
}

export type NotebookTaskExtensionOptions = {
  notebookTaskContext: NotebookTaskContext | null
}

const taskBlockTokenizer = {
  name: 'notebook_task',
  level: 'block' as const,
  start: (src: string) => src.indexOf('```task'),
  tokenize: (src: string) => {
    const m = /^```task\s*\n([\s\S]*?)```/.exec(src)
    if (!m) return undefined
    const inner = (m[1] ?? '').trimEnd()
    const nl = inner.indexOf('\n')
    const metaLine = nl >= 0 ? inner.slice(0, nl).trim() : inner.trim()
    const text = nl >= 0 ? inner.slice(nl + 1) : ''
    return {
      type: 'notebook_task',
      raw: m[0],
      metaLine,
      text,
    }
  },
}

function parseTaskMeta(metaLine: string): { taskId: string; checked: boolean; dueAt: string | null } {
  try {
    const raw = JSON.parse(metaLine) as Record<string, unknown>
    return {
      taskId: typeof raw.id === 'string' ? raw.id : crypto.randomUUID(),
      checked: raw.checked === true,
      dueAt: typeof raw.dueAt === 'string' ? raw.dueAt : null,
    }
  } catch {
    return { taskId: crypto.randomUUID(), checked: false, dueAt: null }
  }
}

export function newNotebookTaskId(): string {
  if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) {
    return crypto.randomUUID()
  }
  return `task-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`
}

export const NotebookTask = Node.create<NotebookTaskExtensionOptions>({
  name: 'notebook_task',
  group: 'block',
  content: 'inline*',
  defining: true,

  addOptions() {
    return {
      notebookTaskContext: null,
    }
  },

  addAttributes() {
    return {
      taskId: {
        default: null,
        parseHTML: (el) => (el as HTMLElement).getAttribute('data-task-id'),
        renderHTML: (attrs) => {
          if (!attrs.taskId) return {}
          return { 'data-task-id': attrs.taskId as string }
        },
      },
      checked: {
        default: false,
        parseHTML: (el) => (el as HTMLElement).getAttribute('data-checked') === 'true',
        renderHTML: (attrs) => ({ 'data-checked': attrs.checked ? 'true' : 'false' }),
      },
      dueAt: {
        default: null,
        parseHTML: (el) => (el as HTMLElement).getAttribute('data-due-at'),
        renderHTML: (attrs) => {
          if (!attrs.dueAt) return {}
          return { 'data-due-at': attrs.dueAt as string }
        },
      },
    }
  },

  parseHTML() {
    return [{ tag: 'div[data-type="notebook-task"]' }]
  },

  renderHTML({ node, HTMLAttributes }) {
    return [
      'div',
      mergeAttributes(HTMLAttributes, {
        'data-type': 'notebook-task',
        'data-task-id': String(node.attrs.taskId ?? ''),
        'data-checked': node.attrs.checked ? 'true' : 'false',
        ...(node.attrs.dueAt ? { 'data-due-at': String(node.attrs.dueAt) } : {}),
      }),
      0,
    ]
  },

  addNodeView() {
    return ReactNodeViewRenderer(NotebookTaskNodeView)
  },

  markdownTokenizer: taskBlockTokenizer,

  parseMarkdown: (token: MarkdownToken) => {
    const metaLine = typeof token.metaLine === 'string' ? token.metaLine : '{}'
    const text = typeof token.text === 'string' ? token.text : ''
    const meta = parseTaskMeta(metaLine)
    const content: JSONContent[] = text ? [{ type: 'text', text }] : []
    return {
      type: 'notebook_task',
      attrs: {
        taskId: meta.taskId,
        checked: meta.checked,
        dueAt: meta.dueAt,
      },
      content,
    } as JSONContent
  },

  renderMarkdown: (node: JSONContent, helpers) => {
    const meta = JSON.stringify({
      id: node.attrs?.taskId ?? newNotebookTaskId(),
      checked: node.attrs?.checked === true,
      dueAt: node.attrs?.dueAt ?? null,
    })
    const text = helpers.renderChildren(node.content ?? [], '\n')
    return `\`\`\`task\n${meta}\n${text}\n\`\`\`\n`
  },
})
