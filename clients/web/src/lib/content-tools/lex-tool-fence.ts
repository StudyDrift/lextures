/** CT.2 — ```lex-tool fence payload helpers (stable key order). */

export type LexToolFencePayload = {
  instanceId: string
  toolId: string
  v: 1
}

const FENCE_LANG = 'lex-tool'

/** Serialize fence body JSON with stable key order: instanceId, toolId, v. */
export function serializeFence(payload: {
  instanceId: string
  toolId: string
  v?: 1
}): string {
  const instanceId = String(payload.instanceId ?? '').trim()
  const toolId = String(payload.toolId ?? '').trim()
  const v = 1 as const
  // Stable key order — do not use JSON.stringify on an object literal that could reorder.
  return `{"instanceId":${JSON.stringify(instanceId)},"toolId":${JSON.stringify(toolId)},"v":${v}}`
}

/** Full fenced block for Markdown insertion. */
export function serializeLexToolFenceBlock(payload: {
  instanceId: string
  toolId: string
  v?: 1
}): string {
  return `\`\`\`${FENCE_LANG}\n${serializeFence(payload)}\n\`\`\``
}

export function parseFencePayload(json: string): LexToolFencePayload | null {
  const trimmed = json.trim()
  if (!trimmed) return null
  let raw: unknown
  try {
    raw = JSON.parse(trimmed) as unknown
  } catch {
    return null
  }
  if (!raw || typeof raw !== 'object' || Array.isArray(raw)) return null
  const obj = raw as Record<string, unknown>
  const instanceId = typeof obj.instanceId === 'string' ? obj.instanceId.trim() : ''
  const toolId = typeof obj.toolId === 'string' ? obj.toolId.trim() : ''
  const v = obj.v
  if (!instanceId || !toolId) return null
  if (v !== 1 && v !== '1') return null
  return { instanceId, toolId, v: 1 }
}

export { FENCE_LANG as LEX_TOOL_FENCE_LANG }
