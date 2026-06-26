import { useCallback, useEffect, useRef, useState } from 'react'
import type { DragEvent, MouseEvent as ReactMouseEvent } from 'react'
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  ReactFlowProvider,
  addEdge,
  useEdgesState,
  useNodesState,
  useReactFlow,
  type Connection,
  type Edge,
  type Node,
  type NodeChange,
  type OnConnect,
  type OnEdgesChange,
  type OnNodesChange,
  type OnSelectionChangeFunc,
  applyEdgeChanges,
  applyNodeChanges,
} from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import './grader-agent-canvas.css'
import { useTranslation } from 'react-i18next'
import { useLmsDarkMode } from '../../../hooks/use-lms-dark-mode'
import type { GraderAgentWorkflowState } from './use-grader-agent-workflow'
import { GRADER_AGENT_DRAG_MIME } from './node-palette'
import { parsePaletteNodeType, consumePaletteDragType } from './palette-drag'
import { connectionIsValid } from './validation'
import type { GraderWorkflowGraph, PaletteNodeType } from './types'
import { graderAgentNodeTypes } from './workflow-node-types'
import { WorkflowCanvasProvider } from './workflow-canvas-context'
import {
  WorkflowCanvasContextMenu,
  type WorkflowCanvasContextMenuItem,
  type WorkflowCanvasContextMenuState,
} from './workflow-canvas-context-menu'
import { workflowHasAttachedRubric } from './workflow-grade-slot'

/** Cap zoom-in so a lone output node does not fill the canvas on first paint. */
const INITIAL_FIT_VIEW = { padding: 0.45, maxZoom: 0.85 } as const

type CanvasViewProps = {
  workflow: GraderAgentWorkflowState
  readOnly?: boolean
}

function graphToFlow(
  graph: GraderWorkflowGraph,
  selectedNodeId: string | null,
  readOnly: boolean,
  nodeExecutionStates: Record<string, string>,
): { nodes: Node[]; edges: Edge[] } {
  const gradeSlotUsesRubric = workflowHasAttachedRubric(graph)
  return {
    nodes: graph.nodes.map((n) => ({
      id: n.id,
      type: n.type,
      position: n.position,
      data: {
        ...(n.type === 'output' ? { ...n.data, gradeSlotUsesRubric } : n.data),
        executionStatus: nodeExecutionStates[n.id] ?? 'idle',
      },
      selected: n.id === selectedNodeId,
      deletable: n.type !== 'output',
      selectable: true,
      draggable: !readOnly,
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

function flowNodeData(node: Node): Record<string, unknown> {
  const data = { ...((node.data ?? {}) as Record<string, unknown>) }
  if (node.type === 'output') delete data.gradeSlotUsesRubric
  delete data.executionStatus

  return data
}

function flowToGraph(nodes: Node[], edges: Edge[], version: number): GraderWorkflowGraph {
  return {
    version,
    nodes: nodes.map((n) => ({
      id: n.id,
      type: n.type as GraderWorkflowGraph['nodes'][0]['type'],
      position: n.position,
      data: flowNodeData(n),
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

function shouldPersistNodeChange(change: NodeChange): boolean {
  if (change.type === 'remove') return true
  if (change.type === 'position' && 'dragging' in change && change.dragging === false) return true
  return false
}

function resolveDroppedPaletteType(event: DragEvent): PaletteNodeType | null {
  const raw = consumePaletteDragType() ?? event.dataTransfer.getData(GRADER_AGENT_DRAG_MIME)
  return parsePaletteNodeType(raw)
}

function CanvasFlow({ workflow, readOnly = false }: CanvasViewProps) {
  const { t } = useTranslation('common')
  const isDark = useLmsDarkMode()
  const didFitView = useRef(false)
  const graphRef = useRef<GraderWorkflowGraph | null>(null)
  const { screenToFlowPosition, fitView } = useReactFlow()
  const {
    graph,
    updateGraph,
    setSelectedNodeId,
    selectedNodeId,
    addPaletteNode,
    updateNodeLabel,
    removeNode,
    removeEdge,
    nodeExecutionStates,
  } = workflow
  const [nodes, setNodes] = useNodesState<Node>([])
  const [edges, setEdges] = useEdgesState<Edge>([])
  const [contextMenu, setContextMenu] = useState<WorkflowCanvasContextMenuState>(null)
  const [renameRequestNodeId, setRenameRequestNodeId] = useState<string | null>(null)
  const nodesRef = useRef(nodes)
  const edgesRef = useRef(edges)

  graphRef.current = graph
  nodesRef.current = nodes
  edgesRef.current = edges

  useEffect(() => {
    if (!graph) return
    const flow = graphToFlow(graph, selectedNodeId, readOnly, nodeExecutionStates)
    nodesRef.current = flow.nodes
    edgesRef.current = flow.edges
    setNodes(flow.nodes)
    setEdges(flow.edges)
  }, [graph, readOnly, selectedNodeId, nodeExecutionStates, setNodes, setEdges])

  useEffect(() => {
    if (!graph || didFitView.current) return
    didFitView.current = true
    requestAnimationFrame(() => {
      void fitView(INITIAL_FIT_VIEW)
    })
  }, [graph, fitView])

  const persistGraph = useCallback(
    (nextNodes: Node[], nextEdges: Edge[]) => {
      const g = graphRef.current
      if (!g) return
      updateGraph(flowToGraph(nextNodes, nextEdges, g.version))
    },
    [updateGraph],
  )

  const onNodesChange: OnNodesChange = useCallback(
    (changes) => {
      const actionable = changes.filter((change) => change.type !== 'replace')
      if (actionable.length === 0) return

      const currentNodes = nodesRef.current
      const nextNodes = applyNodeChanges(actionable, currentNodes)
      nodesRef.current = nextNodes
      setNodes(nextNodes)

      if (!readOnly) {
        const persistable = actionable.filter(shouldPersistNodeChange)
        if (persistable.length > 0) {
          persistGraph(applyNodeChanges(persistable, currentNodes), edgesRef.current)
        }
      }
    },
    [persistGraph, readOnly, setNodes],
  )

  const onEdgesChange: OnEdgesChange = useCallback(
    (changes) => {
      const currentEdges = edgesRef.current
      const nextEdges = applyEdgeChanges(changes, currentEdges)
      edgesRef.current = nextEdges
      setEdges(nextEdges)

      if (!readOnly && changes.some((change) => change.type === 'remove')) {
        persistGraph(nodesRef.current, nextEdges)
      }
    },
    [persistGraph, readOnly, setEdges],
  )

  const onConnect: OnConnect = useCallback(
    (connection: Connection) => {
      if (!graph || readOnly) return
      const { source, sourceHandle, target, targetHandle } = connection
      if (!source || !target) return
      if (!connectionIsValid(graph, source, sourceHandle, target, targetHandle)) return

      const nextEdges = addEdge({ ...connection, id: `e${Date.now()}` }, edgesRef.current)
      edgesRef.current = nextEdges
      setEdges(nextEdges)
      persistGraph(nodesRef.current, nextEdges)
    },
    [graph, persistGraph, readOnly, setEdges],
  )

  const onDragOver = useCallback((event: DragEvent) => {
    event.preventDefault()
    event.stopPropagation()
    event.dataTransfer.dropEffect = 'move'
  }, [])

  const onSelectionChange: OnSelectionChangeFunc = useCallback(
    ({ nodes: selectedNodes }) => {
      const nextId = selectedNodes[0]?.id ?? null
      setSelectedNodeId((current) => (current === nextId ? current : nextId))
    },
    [setSelectedNodeId],
  )

  const closeContextMenu = useCallback(() => setContextMenu(null), [])

  const requestNodeRename = useCallback((nodeId: string) => {
    setRenameRequestNodeId(nodeId)
  }, [])

  const clearRenameRequest = useCallback(() => {
    setRenameRequestNodeId(null)
  }, [])

  const onNodeContextMenu = useCallback(
    (event: ReactMouseEvent, node: Node) => {
      event.preventDefault()
      setSelectedNodeId(node.id)
      if (readOnly) return
      setContextMenu({ kind: 'node', nodeId: node.id, top: event.clientY, left: event.clientX })
    },
    [readOnly, setSelectedNodeId],
  )

  const onEdgeContextMenu = useCallback((event: ReactMouseEvent, edge: Edge) => {
    event.preventDefault()
    setContextMenu({
      kind: 'edge',
      edgeId: edge.id,
      sourceId: edge.source,
      targetId: edge.target,
      top: event.clientY,
      left: event.clientX,
    })
  }, [])

  const handleContextMenuSelect = useCallback(
    (item: WorkflowCanvasContextMenuItem) => {
      if (!contextMenu || !graph) return
      if (contextMenu.kind === 'node') {
        if (item.kind === 'rename') requestNodeRename(contextMenu.nodeId)
        if (item.kind === 'deleteNode') removeNode(contextMenu.nodeId)
      } else {
        if (item.kind === 'selectSource') setSelectedNodeId(contextMenu.sourceId)
        if (item.kind === 'selectTarget') setSelectedNodeId(contextMenu.targetId)
        if (item.kind === 'deleteEdge') removeEdge(contextMenu.edgeId)
      }
      closeContextMenu()
    },
    [
      closeContextMenu,
      contextMenu,
      graph,
      removeEdge,
      removeNode,
      requestNodeRename,
      setSelectedNodeId,
    ],
  )

  const onDrop = useCallback(
    (event: DragEvent) => {
      event.preventDefault()
      event.stopPropagation()
      if (!graph || readOnly) return

      const type = resolveDroppedPaletteType(event)
      if (!type) return

      let position = screenToFlowPosition({ x: event.clientX, y: event.clientY })
      if (!Number.isFinite(position.x) || !Number.isFinite(position.y)) {
        position = { x: 0, y: 0 }
      }
      addPaletteNode(type, position)
    },
    [addPaletteNode, graph, readOnly, screenToFlowPosition],
  )

  if (!graph) return null

  return (
    <div
      className="h-full min-h-0 w-full overflow-hidden rounded-xl border border-slate-200 bg-slate-50 dark:border-neutral-700 dark:bg-neutral-950"
      onDragEnter={onDragOver}
      onDragOver={onDragOver}
      onDrop={onDrop}
    >
      <WorkflowCanvasProvider
        readOnly={readOnly}
        onNodeLabelChange={updateNodeLabel}
        renameRequestNodeId={renameRequestNodeId}
        requestNodeRename={requestNodeRename}
        clearRenameRequest={clearRenameRequest}
      >
        <ReactFlow
          nodes={nodes}
          edges={edges}
          nodeTypes={graderAgentNodeTypes}
          onNodesChange={onNodesChange}
          onEdgesChange={onEdgesChange}
          onConnect={onConnect}
          onSelectionChange={onSelectionChange}
          onNodeClick={() => {
            closeContextMenu()
          }}
          onEdgeClick={() => closeContextMenu()}
          onPaneClick={() => {
            closeContextMenu()
            setSelectedNodeId(null)
          }}
          onNodeContextMenu={onNodeContextMenu}
          onEdgeContextMenu={onEdgeContextMenu}
          nodesFocusable
          edgesFocusable
          proOptions={{ hideAttribution: true }}
          className={`grader-agent-canvas${isDark ? ' dark' : ''}`}
          style={{ width: '100%', height: '100%' }}
        >
          <Background gap={16} />
          <Controls />
          <MiniMap pannable zoomable />
        </ReactFlow>
        {contextMenu ? (
          <WorkflowCanvasContextMenu
            menu={contextMenu}
            graph={graph}
            readOnly={readOnly}
            onClose={closeContextMenu}
            onSelect={handleContextMenuSelect}
          />
        ) : null}
      </WorkflowCanvasProvider>
      {readOnly ? (
        <p className="px-3 py-2 text-xs text-slate-500 dark:text-neutral-400">
          {t('gradingAgent.canvas.readOnlyHint')}
        </p>
      ) : null}
    </div>
  )
}

export function CanvasView(props: CanvasViewProps) {
  return (
    <ReactFlowProvider>
      <CanvasFlow {...props} />
    </ReactFlowProvider>
  )
}
