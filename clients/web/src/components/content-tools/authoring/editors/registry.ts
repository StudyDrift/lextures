import type { ComponentType } from 'react'
import { InlineQuestionsEditor, type InlineQuestionsEditorProps } from '../editors/inline-questions-editor'
import { PredictRevealEditor, type PredictRevealEditorProps } from '../editors/predict-reveal-editor'

export type CustomEditorProps = InlineQuestionsEditorProps | PredictRevealEditorProps

const EDITORS: Record<string, ComponentType<CustomEditorProps>> = {
  inline_questions: InlineQuestionsEditor,
  predict_reveal: PredictRevealEditor,
}

export function resolveCustomEditor(
  editorId: string | undefined | null,
): ComponentType<CustomEditorProps> | null {
  const id = (editorId ?? '').trim()
  if (!id) return null
  return EDITORS[id] ?? null
}
