import { useCallback, useMemo } from 'react'
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  addEdge,
  type Connection,
  type Edge,
  type Node,
  type OnEdgesChange,
  type OnNodesChange,
  applyEdgeChanges,
  applyNodeChanges,
} from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import { useTranslation } from 'react-i18next'
import type { GraderAgentWorkflowState } from './use-grader-agent-workflow'
import { connectionIsValid } from './validation'
import type { GraderWorkflowGraph } from './types'
import { graderAgentNodeTypes } from './workflow-node-types'

type CanvasViewProps = {
  workflow: GraderAgentWorkflowState
  readOnly?: boolean
}

function graphToFlow(graph: GraderWorkflowGraph): { nodes: Node[]; edges: Edge[] } {
  return {
    nodes: graph.nodes.map((n) => ({
      id: n.id,
      type: n.type,
      position: n.position,
      data: n.data,
      deletable: n.type !== 'output' && n.type !== 'submission',
      selectable: n.type !== 'output' && n.type !== 'submission',
      draggable: n.type !== 'output' && n.type !== 'submission',
    })),
    edges: graph.edges.map((e) => ({
      id: e.id,
      source: e.source,
      sourceHandle: e.sourceHandle,
      target: e.target,
      targetHandle: e.targetHandle,
    })),
  }
}

function flowToGraph(nodes: Node[], edges: Edge[], version: number): GraderWorkflowGraph {
  return {
    version,
    nodes: nodes.map((n) => ({
      id: n.id,
      type: n.type as GraderWorkflowGraph['nodes'][0]['type'],
      position: n.position,
      data: (n.data ?? {}) as Record<string, unknown>,
    })),
    edges: edges.map((e) => ({
      id: e.id,
      source: e.source,
      sourceHandle: e.sourceHandle ?? undefined,
      target: e.target,
      targetHandle: e.targetHandle ?? undefined,
    })),
  }
}

export function CanvasView({ workflow, readOnly = false }: CanvasViewProps) {
  const { t } = useTranslation('common')
  const { graph, updateGraph, setSelectedNodeId } = workflow
  const { nodes, edges } = useMemo(() => (graph ? graphToFlow(graph) : { nodes: [], edges: [] }), [graph])

  const onNodesChange: OnNodesChange = useCallback(
    (changes) => {
      if (!graph || readOnly) return
      const nextNodes = applyNodeChanges(changes, nodes)
      updateGraph(flowToGraph(nextNodes, edges, graph.version))
    },
    [graph, nodes, edges, readOnly, updateGraph],
  )

  const onEdgesChange: OnEdgesChange = useCallback(
    (changes) => {
      if (!graph || readOnly) return
      const nextEdges = applyEdgeChanges(changes, edges)
      updateGraph(flowToGraph(nodes, nextEdges, graph.version))
    },
    [graph, nodes, edges, readOnly, updateGraph],
  )

  const onConnect = useCallback(
    (connection: Connection) => {
      if (!graph || readOnly) return
      const { source, sourceHandle, target, targetHandle } = connection
      if (!source || !target) return
      if (!connectionIsValid(graph, source, sourceHandle, target, targetHandle)) return
      const nextEdges = addEdge(
        { ...connection, id: `e${Date.now()}` },
        edges,
      )
      updateGraph(flowToGraph(nodes, nextEdges, graph.version))
    },
    [graph, nodes, edges, readOnly, updateGraph],
  )

  if (!graph) return null

  return (
    <div className="h-full min-h-[320px] w-full rounded-xl border border-slate-200 bg-slate-50 dark:border-neutral-700 dark:bg-neutral-950">
      <ReactFlow
        nodes={nodes}
        edges={edges}
        nodeTypes={graderAgentNodeTypes}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onConnect={onConnect}
        onNodeClick={(_, node) => setSelectedNodeId(node.id)}
        nodesFocusable
        edgesFocusable
        fitView
        proOptions={{ hideAttribution: true }}
        className="grader-agent-canvas"
      >
        <Background gap={16} />
        <Controls />
        <MiniMap pannable zoomable />
      </ReactFlow>
      {readOnly ? (
        <p className="px-3 py-2 text-xs text-slate-500 dark:text-neutral-400">
          {t('gradingAgent.canvas.readOnlyHint')}
        </p>
      ) : null}
    </div>
  )
}
