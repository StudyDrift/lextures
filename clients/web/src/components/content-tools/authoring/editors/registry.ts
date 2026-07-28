import type { ComponentType } from 'react'
import {
  CodeSandboxEditor,
  type CodeSandboxEditorProps,
} from '../editors/code-sandbox-editor'
import {
  DiagramHotspotEditor,
  type DiagramHotspotEditorProps,
} from '../editors/diagram-hotspot-editor'
import {
  ExplainItBackEditor,
  type ExplainItBackEditorProps,
} from '../editors/explain-it-back-editor'
import {
  FlashcardsEditor,
  type FlashcardsEditorProps,
} from '../editors/flashcards-editor'
import {
  HighlightAnnotateEditor,
  type HighlightAnnotateEditorProps,
} from '../editors/highlight-annotate-editor'
import { InlineQuestionsEditor, type InlineQuestionsEditorProps } from '../editors/inline-questions-editor'
import {
  MediaCheckpointsEditor,
  type MediaCheckpointsEditorProps,
} from '../editors/media-checkpoints-editor'
import {
  ParameterExplorerEditor,
  type ParameterExplorerEditorProps,
} from '../editors/parameter-explorer-editor'
import { PredictRevealEditor, type PredictRevealEditorProps } from '../editors/predict-reveal-editor'
import { SortSequenceEditor, type SortSequenceEditorProps } from '../editors/sort-sequence-editor'
import {
  WorkedExampleEditor,
  type WorkedExampleEditorProps,
} from '../editors/worked-example-editor'

export type CustomEditorProps =
  | InlineQuestionsEditorProps
  | PredictRevealEditorProps
  | HighlightAnnotateEditorProps
  | SortSequenceEditorProps
  | DiagramHotspotEditorProps
  | ParameterExplorerEditorProps
  | CodeSandboxEditorProps
  | WorkedExampleEditorProps
  | MediaCheckpointsEditorProps
  | ExplainItBackEditorProps
  | FlashcardsEditorProps

const EDITORS: Record<string, ComponentType<CustomEditorProps>> = {
  code_sandbox: CodeSandboxEditor as ComponentType<CustomEditorProps>,
  diagram_hotspot: DiagramHotspotEditor as ComponentType<CustomEditorProps>,
  explain_it_back: ExplainItBackEditor as ComponentType<CustomEditorProps>,
  flashcards: FlashcardsEditor as ComponentType<CustomEditorProps>,
  highlight_annotate: HighlightAnnotateEditor as ComponentType<CustomEditorProps>,
  inline_questions: InlineQuestionsEditor,
  media_checkpoints: MediaCheckpointsEditor as ComponentType<CustomEditorProps>,
  parameter_explorer: ParameterExplorerEditor as ComponentType<CustomEditorProps>,
  predict_reveal: PredictRevealEditor,
  sort_sequence: SortSequenceEditor as ComponentType<CustomEditorProps>,
  worked_example: WorkedExampleEditor as ComponentType<CustomEditorProps>,
}

export function resolveCustomEditor(
  editorId: string | undefined | null,
): ComponentType<CustomEditorProps> | null {
  const id = (editorId ?? '').trim()
  if (!id) return null
  return EDITORS[id] ?? null
}
