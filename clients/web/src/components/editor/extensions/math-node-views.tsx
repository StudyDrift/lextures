import type React from 'react'
import { NodeViewWrapper, type NodeViewProps } from '@tiptap/react'
import { isEquationEditorEnabled } from '../../../lib/math'
import { KatexExpression } from '../../math/katex-expression'
import { useEquationEditor } from '../equation-editor-context'

function MathNodeViewBody({
  props,
  displayMode,
  dataAttr,
}: {
  props: NodeViewProps
  displayMode: boolean
  dataAttr: string
}) {
  const latex = String(props.node.attrs.latex ?? '')
  const equationEditor = useEquationEditor()
  const canEdit = isEquationEditorEnabled() && equationEditor != null
  const pos = props.getPos()
  const nodeType = displayMode ? 'math_block' : 'math_inline'

  return (
    <NodeViewWrapper
      as={displayMode ? 'div' : 'span'}
      className={displayMode ? 'lex-math-block-root' : 'inline'}
      contentEditable={false}
      data-math-inline={dataAttr === 'inline' ? '' : undefined}
      data-math-block={dataAttr === 'block' ? '' : undefined}
      onDoubleClick={
        canEdit && typeof pos === 'number'
          ? (e: React.MouseEvent) => {
              e.preventDefault()
              e.stopPropagation()
              equationEditor.openEdit({
                editor: props.editor,
                pos,
                nodeType,
                latex,
              })
            }
          : undefined
      }
      title={canEdit ? 'Double-click to edit equation' : undefined}
    >
      <KatexExpression latex={latex} displayMode={displayMode} />
    </NodeViewWrapper>
  )
}

export function MathInlineNodeView(props: NodeViewProps) {
  return <MathNodeViewBody props={props} displayMode={false} dataAttr="inline" />
}

export function MathBlockNodeView(props: NodeViewProps) {
  return <MathNodeViewBody props={props} displayMode dataAttr="block" />
}
