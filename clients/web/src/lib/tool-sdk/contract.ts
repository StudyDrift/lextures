import type { ReactNode } from 'react'

/** Runtime contract version spoken by @lextures/tool-sdk and the host (FR-17). */
export const TOOL_SDK_CONTRACT_VERSION = 1 as const

export type ToolProps<
  TConfig extends Record<string, unknown> = Record<string, unknown>,
  TState extends Record<string, unknown> = Record<string, unknown>,
> = {
  instanceId: string
  toolId: string
  config: TConfig
  state: TState
  status: string
  readOnly: boolean
  save: (patch: Partial<TState> & Record<string, unknown>) => void | Promise<void>
  submit: (patch: Partial<TState> & Record<string, unknown>) => void | Promise<void>
  runAction: (name: string, input: Record<string, unknown>) => Promise<unknown>
  t: (key: string, options?: Record<string, unknown>) => string
  announce: (message: string, assertive?: boolean) => void
}

export type ToolManifestLike = {
  id: string
  version: string
  contract?: number
  sandbox?: 'inprocess' | 'iframe'
  deprecated?: boolean
  sunsetAt?: string
  stateSchemaVersion?: number
  configSchemaVersion?: number
}

export type DefinedTool<
  TConfig extends Record<string, unknown> = Record<string, unknown>,
  TState extends Record<string, unknown> = Record<string, unknown>,
> = {
  manifest: ToolManifestLike
  Renderer: (props: ToolProps<TConfig, TState>) => ReactNode
}
