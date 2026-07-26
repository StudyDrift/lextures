import { lazy, type ComponentType, type LazyExoticComponent } from 'react'
import type { ContentToolRendererProps } from '../runtime-contract'

type RendererComponent = ComponentType<ContentToolRendererProps>

const RENDERERS: Record<string, LazyExoticComponent<RendererComponent>> = {
  noop_probe: lazy(() => import('../../tools/noop_probe/renderer')),
}

export function resolveRenderer(
  toolId: string,
): LazyExoticComponent<RendererComponent> | null {
  const id = toolId.trim()
  if (!id) return null
  return RENDERERS[id] ?? null
}

export function isRendererRegistered(toolId: string): boolean {
  return resolveRenderer(toolId) != null
}
