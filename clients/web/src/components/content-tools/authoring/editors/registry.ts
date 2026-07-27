import type { ComponentType } from 'react'
import {
  DiagramHotspotEditor,
  type DiagramHotspotEditorProps,
} from '../editors/diagram-hotspot-editor'
import {
  HighlightAnnotateEditor,
  type HighlightAnnotateEditorProps,
} from '../editors/highlight-annotate-editor'
import { InlineQuestionsEditor, type InlineQuestionsEditorProps } from '../editors/inline-questions-editor'
import {
  ParameterExplorerEditor,
  type ParameterExplorerEditorProps,
} from '../editors/parameter-explorer-editor'
import { PredictRevealEditor, type PredictRevealEditorProps } from '../editors/predict-reveal-editor'
import { SortSequenceEditor, type SortSequenceEditorProps } from '../editors/sort-sequence-editor'

export type CustomEditorProps =
  | InlineQuestionsEditorProps
  | PredictRevealEditorProps
  | HighlightAnnotateEditorProps
  | SortSequenceEditorProps
  | DiagramHotspotEditorProps
  | ParameterExplorerEditorProps

const EDITORS: Record<string, ComponentType<CustomEditorProps>> = {
  diagram_hotspot: DiagramHotspotEditor as ComponentType<CustomEditorProps>,
  highlight_annotate: HighlightAnnotateEditor as ComponentType<CustomEditorProps>,
  inline_questions: InlineQuestionsEditor,
  parameter_explorer: ParameterExplorerEditor as ComponentType<CustomEditorProps>,
  predict_reveal: PredictRevealEditor,
  sort_sequence: SortSequenceEditor as ComponentType<CustomEditorProps>,
}

export function resolveCustomEditor(
  editorId: string | undefined | null,
): ComponentType<CustomEditorProps> | null {
  const id = (editorId ?? '').trim()
  if (!id) return null
  return EDITORS[id] ?? null
}
