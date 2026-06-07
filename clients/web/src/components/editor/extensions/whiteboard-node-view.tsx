import { useCallback } from 'react'
import { NodeViewWrapper, type NodeViewProps } from '@tiptap/react'
import { EmbeddedWhiteboard } from '../../whiteboard/embedded-whiteboard'
import { parseWhiteboardElements, serializeWhiteboardElements } from '../../../lib/whiteboard/serialize'
import type { DrawEl } from '../../../lib/whiteboard/types'

export function WhiteboardNodeView(props: NodeViewProps) {
  const elements = parseWhiteboardElements(props.node.attrs.elements)
  const editable = props.editor.isEditable

  const onElementsChange = useCallback(
    (next: DrawEl[]) => {
      props.updateAttributes({ elements: serializeWhiteboardElements(next) })
    },
    [props.updateAttributes],
  )

  return (
    <NodeViewWrapper
      as="div"
      className="lex-whiteboard-block my-4"
      contentEditable={false}
      data-type="whiteboard-block"
    >
      <EmbeddedWhiteboard elements={elements} onElementsChange={onElementsChange} disabled={!editable} />
    </NodeViewWrapper>
  )
}
