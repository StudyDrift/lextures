import type { Editor } from '@tiptap/core'
import type { EditorState } from '@tiptap/pm/state'
import { getMentionBlockContext } from './markdown-body-mention'

export type SlashCommandId =
  | 'heading1'
  | 'heading2'
  | 'heading3'
  | 'paragraph'
  | 'image'
  | 'bulletList'
  | 'orderedList'
  | 'codeBlock'
  | 'blockquote'
  | 'horizontalRule'
  | 'equation'

export type SlashCommand = {
  id: SlashCommandId
  label: string
  description: string
  keywords: string[]
}

const BASE_SLASH_COMMANDS: SlashCommand[] = [
  {
    id: 'heading1',
    label: 'Heading 1',
    description: 'Large section heading',
    keywords: ['h1', 'title', 'heading'],
  },
  {
    id: 'heading2',
    label: 'Heading 2',
    description: 'Medium section heading',
    keywords: ['h2', 'heading'],
  },
  {
    id: 'heading3',
    label: 'Heading 3',
    description: 'Small section heading',
    keywords: ['h3', 'heading'],
  },
  {
    id: 'paragraph',
    label: 'Paragraph',
    description: 'Plain text block',
    keywords: ['text', 'p', 'body'],
  },
  {
    id: 'image',
    label: 'Insert image',
    description: 'Upload and embed an image',
    keywords: ['image', 'photo', 'picture', 'img', 'upload'],
  },
  {
    id: 'bulletList',
    label: 'Bullet list',
    description: 'Unordered list',
    keywords: ['ul', 'list', 'bullets'],
  },
  {
    id: 'orderedList',
    label: 'Numbered list',
    description: 'Ordered list',
    keywords: ['ol', 'list', 'numbers'],
  },
  {
    id: 'codeBlock',
    label: 'Code',
    description: 'Code block with syntax highlighting',
    keywords: ['code', 'pre', 'snippet'],
  },
  {
    id: 'blockquote',
    label: 'Quote',
    description: 'Indented quotation',
    keywords: ['quote', 'blockquote'],
  },
  {
    id: 'horizontalRule',
    label: 'Divider',
    description: 'Horizontal line',
    keywords: ['hr', 'divider', 'line', 'rule'],
  },
]

const EQUATION_COMMAND: SlashCommand = {
  id: 'equation',
  label: 'Equation',
  description: 'Insert LaTeX math',
  keywords: ['math', 'latex', 'equation', 'formula'],
}

/** Slash command in the current block: `/` after whitespace or block start, query has no spaces. */
export function getSlashState(text: string, caret: number): { start: number; query: string } | null {
  const before = text.slice(0, caret)
  const slash = before.lastIndexOf('/')
  if (slash < 0) return null
  if (slash > 0 && !/\s/.test(before[slash - 1]!)) return null
  const afterSlash = before.slice(slash + 1)
  if (afterSlash.includes('\n') || afterSlash.includes(' ')) return null
  return { start: slash, query: afterSlash }
}

/** Active `/` query inside the current block, with document positions for replace. */
export function getBlockSlashRange(
  state: EditorState,
): { from: number; to: number; query: string } | null {
  const ctx = getMentionBlockContext(state)
  if (!ctx) return null
  const slash = getSlashState(ctx.text, ctx.text.length)
  if (!slash) return null
  return {
    from: ctx.blockStart + slash.start,
    to: ctx.cursorPos,
    query: slash.query,
  }
}

export function slashCommandsForEditor(options?: {
  equation?: boolean
  image?: boolean
}): SlashCommand[] {
  return BASE_SLASH_COMMANDS.filter((cmd) => {
    if (cmd.id === 'image') return Boolean(options?.image)
    return true
  }).concat(options?.equation ? [EQUATION_COMMAND] : [])
}

export function filterSlashCommands(commands: SlashCommand[], query: string): SlashCommand[] {
  const q = query.trim().toLowerCase()
  if (!q) return commands
  return commands.filter((cmd) => slashCommandMatchesQuery(cmd, q))
}

function slashCommandMatchesQuery(cmd: SlashCommand, q: string): boolean {
  if (cmd.id === q || cmd.id.startsWith(q) || q.startsWith(cmd.id)) return true
  const label = cmd.label.toLowerCase()
  const description = cmd.description.toLowerCase()
  if (label.includes(q) || description.includes(q)) return true
  return cmd.keywords.some((kw) => keywordMatchesQuery(kw, q))
}

function keywordMatchesQuery(keyword: string, q: string): boolean {
  if (keyword === q) return true
  if (keyword.length < 2 || q.length < 2) return false
  return keyword.startsWith(q) || q.startsWith(keyword)
}

export function applySlashCommand(
  editor: Editor,
  command: SlashCommand,
  range: { from: number; to: number },
  options?: { onEquation?: () => void; onImage?: () => void },
): void {
  if (command.id === 'equation') {
    editor.chain().focus().deleteRange({ from: range.from, to: range.to }).run()
    options?.onEquation?.()
    return
  }
  if (command.id === 'image') {
    editor.chain().focus().deleteRange({ from: range.from, to: range.to }).run()
    options?.onImage?.()
    return
  }

  const chain = editor.chain().focus().deleteRange({ from: range.from, to: range.to })

  switch (command.id) {
    case 'heading1':
      chain.setHeading({ level: 1 }).run()
      break
    case 'heading2':
      chain.setHeading({ level: 2 }).run()
      break
    case 'heading3':
      chain.setHeading({ level: 3 }).run()
      break
    case 'paragraph':
      chain.setParagraph().run()
      break
    case 'bulletList':
      chain.toggleBulletList().run()
      break
    case 'orderedList':
      chain.toggleOrderedList().run()
      break
    case 'codeBlock':
      chain.toggleCodeBlock().run()
      break
    case 'blockquote':
      chain.toggleBlockquote().run()
      break
    case 'horizontalRule':
      chain.setHorizontalRule().run()
      break
    default: {
      const _exhaustive: never = command.id
      void _exhaustive
    }
  }
}
