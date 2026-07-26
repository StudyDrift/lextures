/** CT.3 — versioned props contract for first-party Content Tool renderers. */

export const RUNTIME_CONTRACT_VERSION = 1 as const

export type ContentToolRendererProps = {
  instanceId: string
  toolId: string
  config: Record<string, unknown>
  state: Record<string, unknown>
  status: string
  readOnly: boolean
  save: (patch: Record<string, unknown>) => void | Promise<void>
  submit: (patch: Record<string, unknown>) => void | Promise<void>
  runAction: (name: string, input: Record<string, unknown>) => Promise<unknown>
  t: (key: string, options?: Record<string, unknown>) => string
  announce: (message: string) => void
}
