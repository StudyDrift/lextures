import { useMemo, useState, type ReactElement } from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { flushSync } from 'react-dom'
import type { DefinedTool, ToolProps } from '../contract'

export type RenderToolOptions<
  TConfig extends Record<string, unknown> = Record<string, unknown>,
  TState extends Record<string, unknown> = Record<string, unknown>,
> = {
  config?: TConfig
  state?: TState
  readOnly?: boolean
  locale?: string
  onAction?: (name: string, input: Record<string, unknown>) => Promise<unknown> | unknown
}

export type RenderToolResult<TState extends Record<string, unknown> = Record<string, unknown>> = {
  container: HTMLElement
  getState: () => TState
  getActions: () => Array<{ name: string; input: Record<string, unknown> }>
  unmount: () => void
  announceLog: string[]
}

/**
 * Mounts a tool against an in-memory state store and mock action dispatcher (FR-2 / AC-1).
 * No server or database required.
 */
export function renderTool<
  TConfig extends Record<string, unknown> = Record<string, unknown>,
  TState extends Record<string, unknown> = Record<string, unknown>,
>(
  tool: DefinedTool<TConfig, TState>,
  options: RenderToolOptions<TConfig, TState> = {},
): RenderToolResult<TState> {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const actions: Array<{ name: string; input: Record<string, unknown> }> = []
  const announceLog: string[] = []
  let state = { ...(options.state ?? ({} as TState)) } as TState
  let root: Root | null = createRoot(container)

  function Harness() {
    const [current, setCurrent] = useState(state)
    const props: ToolProps<TConfig, TState> = useMemo(
      () => ({
        instanceId: 'harness-instance',
        toolId: tool.manifest.id,
        config: (options.config ?? {}) as TConfig,
        state: current,
        status: 'in_progress',
        readOnly: Boolean(options.readOnly),
        save: (patch: Partial<TState> & Record<string, unknown>) => {
          state = { ...state, ...patch } as TState
          setCurrent(state)
        },
        submit: (patch: Partial<TState> & Record<string, unknown>) => {
          state = { ...state, ...patch } as TState
          setCurrent(state)
        },
        runAction: async (name: string, input: Record<string, unknown>) => {
          actions.push({ name, input })
          if (options.onAction) return options.onAction(name, input)
          return { ok: true }
        },
        t: (key: string) => key,
        announce: (message: string) => {
          announceLog.push(message)
        },
      }),
      [current],
    )
    return tool.Renderer(props) as ReactElement
  }

  flushSync(() => {
    root!.render(<Harness />)
  })

  return {
    container,
    getState: () => state,
    getActions: () => actions,
    announceLog,
    unmount: () => {
      root?.unmount()
      root = null
      container.remove()
    },
  }
}
