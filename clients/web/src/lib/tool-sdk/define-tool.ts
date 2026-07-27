import type { ReactNode } from 'react'
import type { DefinedTool, ToolManifestLike, ToolProps } from './contract'
import { TOOL_SDK_CONTRACT_VERSION } from './contract'

export function defineTool<
  TConfig extends Record<string, unknown> = Record<string, unknown>,
  TState extends Record<string, unknown> = Record<string, unknown>,
>(
  manifest: ToolManifestLike,
  Renderer: (props: ToolProps<TConfig, TState>) => ReactNode,
): DefinedTool<TConfig, TState> {
  return {
    manifest: {
      ...manifest,
      contract: manifest.contract ?? TOOL_SDK_CONTRACT_VERSION,
    },
    Renderer,
  }
}
