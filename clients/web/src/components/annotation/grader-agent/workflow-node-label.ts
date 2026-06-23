/** Optional display label persisted on workflow node `data.label`. */

export function workflowNodeLabel(data: Record<string, unknown>): string | null {
  if (typeof data.label !== 'string') return null
  const trimmed = data.label.trim()
  return trimmed || null
}

export function workflowNodeDisplayLabel(data: Record<string, unknown>, defaultLabel: string): string {
  return workflowNodeLabel(data) ?? defaultLabel
}

export function patchWorkflowNodeLabel(
  data: Record<string, unknown>,
  label: string | null,
): Record<string, unknown> {
  if (!label) {
    const { label: _removed, ...rest } = data
    return rest
  }
  return { ...data, label }
}