import { mergeAttributes, Node, type JSONContent, type MarkdownToken } from '@tiptap/core'
import { ReactNodeViewRenderer } from '@tiptap/react'
import { parseWhiteboardElements, serializeWhiteboardElements } from '../../../lib/whiteboard/serialize'
import type { DrawEl } from '../../../lib/whiteboard/types'
import { WhiteboardNodeView } from './whiteboard-node-view'

const drawingBlockTokenizer = {
  name: 'whiteboard_block',
  level: 'block' as const,
  start: (src: string) => src.indexOf('```drawing'),
  tokenize: (src: string) => {
    const m = /^```drawing\s*\n([\s\S]*?)```/.exec(src)
    if (!m) return undefined
    return {
      type: 'whiteboard_block',
      raw: m[0],
      elementsJson: (m[1] ?? '').trim(),
    }
  },
}

export const WhiteboardBlock = Node.create({
  name: 'whiteboard_block',
  group: 'block',
  atom: true,
  draggable: true,
  selectable: true,

  addAttributes() {
    return {
      elements: {
        default: '[]',
        parseHTML: (el) => (el as HTMLElement).getAttribute('data-elements') ?? '[]',
        renderHTML: (attrs) => ({ 'data-elements': attrs.elements as string }),
      },
    }
  },

  parseHTML() {
    return [{ tag: 'div[data-type="whiteboard-block"]' }]
  },

  renderHTML({ node, HTMLAttributes }) {
    return [
      'div',
      mergeAttributes(HTMLAttributes, {
        'data-type': 'whiteboard-block',
        'data-elements': String(node.attrs.elements ?? '[]'),
      }),
    ]
  },

  addNodeView() {
    return ReactNodeViewRenderer(WhiteboardNodeView)
  },

  markdownTokenizer: drawingBlockTokenizer,

  parseMarkdown: (token: MarkdownToken) => {
    const json = typeof token.elementsJson === 'string' ? token.elementsJson : '[]'
    return {
      type: 'whiteboard_block',
      attrs: { elements: json || '[]' },
    } as JSONContent
  },

  renderMarkdown: (node: JSONContent) => {
    const elements = parseWhiteboardElements(node.attrs?.elements ?? '[]')
    const body = serializeWhiteboardElements(elements as DrawEl[])
    return `\`\`\`drawing\n${body}\n\`\`\``
  },
})
