import { lazy, type ComponentType, type LazyExoticComponent } from 'react'
import type { ContentToolRendererProps } from '../runtime-contract'

type RendererComponent = ComponentType<ContentToolRendererProps>

const RENDERERS: Record<string, LazyExoticComponent<RendererComponent>> = {
  ask_questions: lazy(() => import('../../tools/ask_questions/renderer')),
  code_sandbox: lazy(() => import('../../tools/code_sandbox/renderer')),
  diagram_hotspot: lazy(() => import('../../tools/diagram_hotspot/renderer')),
  explain_it_back: lazy(() => import('../../tools/explain_it_back/renderer')),
  highlight_annotate: lazy(() => import('../../tools/highlight_annotate/renderer')),
  inline_questions: lazy(() => import('../../tools/inline_questions/renderer')),
  media_checkpoints: lazy(() => import('../../tools/media_checkpoints/renderer')),
  noop_probe: lazy(() => import('../../tools/noop_probe/renderer')),
  parameter_explorer: lazy(() => import('../../tools/parameter_explorer/renderer')),
  predict_reveal: lazy(() => import('../../tools/predict_reveal/renderer')),
  sort_sequence: lazy(() => import('../../tools/sort_sequence/renderer')),
  worked_example: lazy(() => import('../../tools/worked_example/renderer')),
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
