export type NotebookTaskMeta = {
  id: string
  checked: boolean
  dueAt: string | null
}

const TASK_BLOCK_RE = /```task\s*\n([\s\S]*?)```/g

function parseMetaLine(line: string): NotebookTaskMeta | null {
  try {
    const raw = JSON.parse(line) as Record<string, unknown>
    const id = typeof raw.id === 'string' ? raw.id : ''
    if (!id) return null
    return {
      id,
      checked: raw.checked === true,
      dueAt: typeof raw.dueAt === 'string' ? raw.dueAt : null,
    }
  } catch {
    return null
  }
}

export type ParsedNotebookTask = {
  id: string
  text: string
  checked: boolean
  dueAt: string | null
}

/** Extract task blocks from notebook markdown. */
export function parseNotebookTasksFromMarkdown(contentMd: string): ParsedNotebookTask[] {
  const out: ParsedNotebookTask[] = []
  for (const match of contentMd.matchAll(TASK_BLOCK_RE)) {
    const inner = match[1] ?? ''
    const lines = inner.split('\n')
    const meta = parseMetaLine(lines[0] ?? '')
    if (!meta) continue
    out.push({
      id: meta.id,
      text: lines.slice(1).join('\n').trim(),
      checked: meta.checked,
      dueAt: meta.dueAt,
    })
  }
  return out
}

/** Mark a task block as checked in notebook markdown by task id. */
export function markTaskCheckedInMarkdown(contentMd: string, taskId: string): string {
  return contentMd.replace(TASK_BLOCK_RE, (block, inner: string) => {
    const lines = inner.split('\n')
    const metaLine = lines[0] ?? ''
    const meta = parseMetaLine(metaLine)
    if (!meta || meta.id !== taskId) return block
    const nextMeta = JSON.stringify({ ...meta, checked: true })
    const body = lines.slice(1).join('\n')
    return `\`\`\`task\n${nextMeta}\n${body}\n\`\`\``
  })
}
