import { NodeViewWrapper, type NodeViewProps } from '@tiptap/react'
import { ToolBlockCard } from '../../content-tools/authoring/tool-block-card'
import { useContentToolAuthoring } from '../../content-tools/authoring/content-tool-authoring-context'

export function ContentToolNodeView(props: NodeViewProps) {
  const instanceId = String(props.node.attrs.instanceId ?? '')
  const toolId = String(props.node.attrs.toolId ?? '')
  const courseCodeAttr = String(props.node.attrs.courseCode ?? '')
  const authoring = useContentToolAuthoring()

  const instance = authoring?.instances[instanceId]
  const catalogTool = authoring?.catalog.find((t) => t.id === toolId)
  const courseCode = authoring?.courseCode || courseCodeAttr

  return (
    <NodeViewWrapper
      as="div"
      className="lex-content-tool-block my-3"
      contentEditable={false}
      data-type="content-tool-block"
      data-instance-id={instanceId}
      data-tool-id={toolId}
      data-course-code={courseCode}
    >
      <ToolBlockCard
        instanceId={instanceId}
        toolId={toolId}
        instance={instance}
        catalogTool={catalogTool}
        onConfigure={() => authoring?.onConfigure(instanceId)}
        onPreview={() => authoring?.onPreview(instanceId)}
        onDuplicate={() => authoring?.onDuplicate(instanceId)}
        onDelete={() => authoring?.onDelete(instanceId)}
      />
    </NodeViewWrapper>
  )
}
