import type { Editor } from '@tiptap/core'
import type { Node as ProseMirrorNode } from '@tiptap/pm/model'

const DEFAULT_COL_WIDTH = 120
const MIN_COL_WIDTH = 48
const MAX_COL_WIDTH = 640

function tableInfoAtSelection(editor: Editor): {
  tablePos: number
  tableNode: ProseMirrorNode
  colIndex: number
} | null {
  const { state } = editor
  const $from = state.selection.$from
  let tableDepth = -1
  let cellDepth = -1
  for (let d = $from.depth; d > 0; d--) {
    const name = $from.node(d).type.name
    if ((name === 'tableCell' || name === 'tableHeader') && cellDepth < 0) {
      cellDepth = d
    }
    if (name === 'table') {
      tableDepth = d
      break
    }
  }
  if (tableDepth < 0 || cellDepth < 0) return null

  const tablePos = $from.before(tableDepth)
  const tableNode = $from.node(tableDepth)
  const cellPos = $from.before(cellDepth)

  // Column index = number of cells before this one in the same row.
  const $cell = state.doc.resolve(cellPos)
  const row = $cell.parent
  let colIndex = 0
  for (let i = 0; i < $cell.index($cell.depth); i++) {
    const cell = row.child(i)
    colIndex += cell.attrs.colspan ?? 1
  }

  return { tablePos, tableNode, colIndex }
}

/** Grow or shrink the selected column via `colwidth` on every cell in that column. */
export function adjustSelectedColumnWidth(editor: Editor, delta: number): boolean {
  const info = tableInfoAtSelection(editor)
  if (!info) return false

  const { tablePos, tableNode, colIndex } = info
  let tr = editor.state.tr
  let changed = false
  let pos = tablePos + 1

  for (let r = 0; r < tableNode.childCount; r++) {
    const row = tableNode.child(r)
    pos += 1 // enter row
    let col = 0
    for (let c = 0; c < row.childCount; c++) {
      const cell = row.child(c)
      const span = cell.attrs.colspan ?? 1
      if (col === colIndex && span === 1) {
        const prev: number[] | null = Array.isArray(cell.attrs.colwidth)
          ? (cell.attrs.colwidth as number[])
          : null
        const current = prev?.[0] && prev[0] > 0 ? prev[0] : DEFAULT_COL_WIDTH
        const next = Math.min(MAX_COL_WIDTH, Math.max(MIN_COL_WIDTH, current + delta))
        if (next !== current) {
          tr = tr.setNodeMarkup(pos, undefined, {
            ...cell.attrs,
            colwidth: [next],
          })
          changed = true
        }
      }
      pos += cell.nodeSize
      col += span
    }
    pos += 1 // leave row
  }

  if (!changed) return false
  editor.view.dispatch(tr)
  return true
}

export function insertDefaultTable(editor: Editor): boolean {
  return editor.chain().focus().insertTable({ rows: 3, cols: 3, withHeaderRow: true }).run()
}
