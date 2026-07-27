import type { ComponentType } from 'react'
import { InlineQuestionsEditor, type InlineQuestionsEditorProps } from '../editors/inline-questions-editor'

export type CustomEditorProps = InlineQuestionsEditorProps

const EDITORS: Record<string, ComponentType<CustomEditorProps>> = {
  inline_questions: InlineQuestionsEditor,
}

export function resolveCustomEditor(
  editorId: string | undefined | null,
): ComponentType<CustomEditorProps> | null {
  const id = (editorId ?? '').trim()
  if (!id) return null
  return EDITORS[id] ?? null
}
