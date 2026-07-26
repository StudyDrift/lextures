import { mergeAttributes, Node, type JSONContent, type MarkdownToken } from '@tiptap/core'
import { ReactNodeViewRenderer } from '@tiptap/react'
import { parseFencePayload, serializeFence } from '../../../lib/content-tools/lex-tool-fence'
import { ContentToolNodeView } from './content-tool-node-view'

const contentToolBlockTokenizer = {
  name: 'content_tool_block',
  level: 'block' as const,
  start: (src: string) => src.indexOf('```lex-tool'),
  tokenize: (src: string) => {
    const m = /^```lex-tool\s*\n([\s\S]*?)```/.exec(src)
    if (!m) return undefined
    const payload = parseFencePayload(m[1] ?? '')
    if (!payload) return undefined
    return {
      type: 'content_tool_block',
      raw: m[0],
      instanceId: payload.instanceId,
      toolId: payload.toolId,
    }
  },
}

export const ContentToolBlock = Node.create({
  name: 'content_tool_block',
  group: 'block',
  atom: true,
  draggable: true,
  selectable: true,

  addAttributes() {
    return {
      instanceId: {
        default: '',
        parseHTML: (el) => (el as HTMLElement).getAttribute('data-instance-id') ?? '',
        renderHTML: (attrs) => ({ 'data-instance-id': attrs.instanceId as string }),
      },
      toolId: {
        default: '',
        parseHTML: (el) => (el as HTMLElement).getAttribute('data-tool-id') ?? '',
        renderHTML: (attrs) => ({ 'data-tool-id': attrs.toolId as string }),
      },
      courseCode: {
        default: '',
        parseHTML: (el) => (el as HTMLElement).getAttribute('data-course-code') ?? '',
        renderHTML: (attrs) => ({ 'data-course-code': attrs.courseCode as string }),
      },
    }
  },

  parseHTML() {
    return [{ tag: 'div[data-type="content-tool-block"]' }]
  },

  renderHTML({ node, HTMLAttributes }) {
    return [
      'div',
      mergeAttributes(HTMLAttributes, {
        'data-type': 'content-tool-block',
        'data-instance-id': String(node.attrs.instanceId ?? ''),
        'data-tool-id': String(node.attrs.toolId ?? ''),
        'data-course-code': String(node.attrs.courseCode ?? ''),
      }),
    ]
  },

  addNodeView() {
    return ReactNodeViewRenderer(ContentToolNodeView)
  },

  markdownTokenizer: contentToolBlockTokenizer,

  parseMarkdown: (token: MarkdownToken) => {
    const instanceId = typeof token.instanceId === 'string' ? token.instanceId : ''
    const toolId = typeof token.toolId === 'string' ? token.toolId : ''
    return {
      type: 'content_tool_block',
      attrs: { instanceId, toolId, courseCode: '' },
    } as JSONContent
  },

  renderMarkdown: (node: JSONContent) => {
    const instanceId = String(node.attrs?.instanceId ?? '')
    const toolId = String(node.attrs?.toolId ?? '')
    return `\`\`\`lex-tool\n${serializeFence({ instanceId, toolId, v: 1 })}\n\`\`\``
  },
})
