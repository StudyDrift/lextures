import { mergeAttributes, Node, type JSONContent, type MarkdownToken } from '@tiptap/core'
import { ReactNodeViewRenderer } from '@tiptap/react'
import { BoardNodeView } from './board-node-view'

const boardBlockTokenizer = {
  name: 'board_block',
  level: 'block' as const,
  start: (src: string) => src.indexOf('```board'),
  tokenize: (src: string) => {
    const m = /^```board\s*\n([\s\S]*?)```/.exec(src)
    if (!m) return undefined
    return {
      type: 'board_block',
      raw: m[0],
      boardId: (m[1] ?? '').trim(),
    }
  },
}

export const BoardBlock = Node.create({
  name: 'board_block',
  group: 'block',
  atom: true,
  draggable: true,
  selectable: true,

  addAttributes() {
    return {
      boardId: {
        default: '',
        parseHTML: (el) => (el as HTMLElement).getAttribute('data-board-id') ?? '',
        renderHTML: (attrs) => ({ 'data-board-id': attrs.boardId as string }),
      },
      courseCode: {
        default: '',
        parseHTML: (el) => (el as HTMLElement).getAttribute('data-course-code') ?? '',
        renderHTML: (attrs) => ({ 'data-course-code': attrs.courseCode as string }),
      },
    }
  },

  parseHTML() {
    return [{ tag: 'div[data-type="board-block"]' }]
  },

  renderHTML({ node, HTMLAttributes }) {
    return [
      'div',
      mergeAttributes(HTMLAttributes, {
        'data-type': 'board-block',
        'data-board-id': String(node.attrs.boardId ?? ''),
        'data-course-code': String(node.attrs.courseCode ?? ''),
      }),
    ]
  },

  addNodeView() {
    return ReactNodeViewRenderer(BoardNodeView)
  },

  markdownTokenizer: boardBlockTokenizer,

  parseMarkdown: (token: MarkdownToken) => {
    const boardId = typeof token.boardId === 'string' ? token.boardId : ''
    return {
      type: 'board_block',
      attrs: { boardId, courseCode: '' },
    } as JSONContent
  },

  renderMarkdown: (node: JSONContent) => {
    const boardId = String(node.attrs?.boardId ?? '')
    return `\`\`\`board\n${boardId}\n\`\`\``
  },
})
