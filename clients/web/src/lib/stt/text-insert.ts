import type { Editor } from '@tiptap/core'

export type InterimRange = { from: number; to: number } | null

export function insertTipTapDictationInterim(editor: Editor, text: string, range: InterimRange): InterimRange {
  let chain = editor.chain().focus()
  if (range) {
    chain = chain.deleteRange(range)
  }
  const from = editor.state.selection.from
  const escaped = text.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
  chain
    .insertContentAt(from, `<span data-dictation-interim="true" style="color:#94a3b8">${escaped}</span>`)
    .run()
  return { from, to: editor.state.selection.to }
}

export function commitTipTapDictationFinal(editor: Editor, text: string, range: InterimRange): void {
  let chain = editor.chain().focus()
  if (range) {
    chain = chain.deleteRange(range)
  }
  const from = editor.state.selection.from
  chain.insertContentAt(from, text).run()
}

export function insertTextAtSelection(
  el: HTMLTextAreaElement | HTMLInputElement,
  text: string,
  interimStart: number | null,
  interimLength: number,
): { value: string; caret: number; interimStart: number; interimLength: number } {
  const current = el.value
  let start = el.selectionStart ?? current.length
  let end = el.selectionEnd ?? current.length

  if (interimStart != null && interimLength > 0) {
    current.slice(interimStart, interimStart + interimLength)
    start = interimStart
    end = interimStart + interimLength
  }

  const next = current.slice(0, start) + text + current.slice(end)
  const caret = start + text.length
  return {
    value: next,
    caret,
    interimStart: start,
    interimLength: text.length,
  }
}

export function commitTextAtSelection(
  el: HTMLTextAreaElement | HTMLInputElement,
  text: string,
  interimStart: number | null,
  interimLength: number,
): { value: string; caret: number } {
  const result = insertTextAtSelection(el, text, interimStart, interimLength)
  return { value: result.value, caret: result.caret }
}
