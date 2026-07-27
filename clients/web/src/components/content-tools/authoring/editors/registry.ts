import type { ComponentType } from 'react'
import {
  HighlightAnnotateEditor,
  type HighlightAnnotateEditorProps,
} from '../editors/highlight-annotate-editor'
import { InlineQuestionsEditor, type InlineQuestionsEditorProps } from '../editors/inline-questions-editor'
import { PredictRevealEditor, type PredictRevealEditorProps } from '../editors/predict-reveal-editor'
import { SortSequenceEditor, type SortSequenceEditorProps } from '../editors/sort-sequence-editor'

export type CustomEditorProps =
  | InlineQuestionsEditorProps
  | PredictRevealEditorProps
  | HighlightAnnotateEditorProps
  | SortSequenceEditorProps

const EDITORS: Record<string, ComponentType<CustomEditorProps>> = {
  highlight_annotate: HighlightAnnotateEditor as ComponentType<CustomEditorProps>,
  inline_questions: InlineQuestionsEditor,
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
