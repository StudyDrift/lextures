/** Trimmed instructor prompt from workflow node data. */
export function workflowPromptText(data: Record<string, unknown>): string {
  return typeof data.prompt === 'string' ? data.prompt.trim() : ''
}

/** True when the prompt has substantive content (not blank or punctuation-only). */
export function workflowPromptIsPresent(data: Record<string, unknown>): boolean {
  const text = workflowPromptText(data)
  if (!text) return false
  return /[\p{L}\p{N}]/u.test(text)
}