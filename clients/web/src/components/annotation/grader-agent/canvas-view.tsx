import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
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
  readOnly: boolean,
  quizQuestionSlots: import('./quiz-question-slots').QuizQuestionSlot[] | undefined,
): { nodes: Node[]; edges: Edge[] } {
  const gradeSlotUsesRubric = workflowHasAttachedRubric(graph)
  const quizSlots = quizQuestionSlots ?? []
  return {
    nodes: graph.nodes.map((n) => ({
      id: n.id,
      type: n.type,
      position: n.position,
      data: {
        ...(n.type === 'output' || n.type === 'quizResponses'
          ? { ...n.data, quizQuestionSlots: quizSlots }
          : n.data),
        ...(n.type === 'output' ? { gradeSlotUsesRubric } : {}),
        executionStatus: 'idle',
      },
      selected: false,
      deletable: n.type !== 'output' && n.type !== 'quizResponses',
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

function applyNodePresentation(
  nodes: Node[],
  selectedNodeId: string | null,
  nodeExecutionStates: Record<string, string>,
  quizQuestionSlots: import('./quiz-question-slots').QuizQuestionSlot[] | undefined,
  gradeSlotUsesRubric: boolean,
): Node[] {
  const quizSlots = quizQuestionSlots ?? []
  return nodes.map((node) => {
    const isQuizSlotNode = node.type === 'output' || node.type === 'quizResponses'
    return {
      ...node,
      selected: node.id === selectedNodeId,
      data: {
        ...(node.data ?? {}),
        ...(isQuizSlotNode ? { quizQuestionSlots: quizSlots } : {}),
        ...(node.type === 'output' ? { gradeSlotUsesRubric } : {}),
        executionStatus: nodeExecutionStates[node.id] ?? 'idle',
      },
    }
  })
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
    quizQuestionSlots,
    navStack,
    enterGroup,
    exitToDepth,
    groupSelection,
    ungroup,
  } = workflow
  const [nodes, setNodes] = useNodesState<Node>([])
  const [edges, setEdges] = useEdgesState<Edge>([])
  const [contextMenu, setContextMenu] = useState<WorkflowCanvasContextMenuState>(null)
  const [renameRequestNodeId, setRenameRequestNodeId] = useState<string | null>(null)
  const [selectedNodeIds, setSelectedNodeIds] = useState<string[]>([])
  const nodesRef = useRef(nodes)
  const edgesRef = useRef(edges)
  const isDraggingNodeRef = useRef(false)

  graphRef.current = graph
  nodesRef.current = nodes
  edgesRef.current = edges

  const graphStructureKey = useMemo(() => {
    if (!graph) return ''
    return JSON.stringify({
      version: graph.version,
      nodes: graph.nodes.map((node) => ({
        id: node.id,
        type: node.type,
        position: node.position,
        data: node.data,
      })),
      edges: graph.edges,
    })
  }, [graph])

  const gradeSlotUsesRubric = graph ? workflowHasAttachedRubric(graph) : false

  useEffect(() => {
    if (!graph) return
    const flow = graphToFlow(graph, readOnly, quizQuestionSlots)
    nodesRef.current = flow.nodes
    edgesRef.current = flow.edges
    setNodes(flow.nodes)
    setEdges(flow.edges)
  }, [graph, graphStructureKey, readOnly, quizQuestionSlots, setNodes, setEdges])

  useEffect(() => {
    if (isDraggingNodeRef.current) return
    setNodes((current) =>
      applyNodePresentation(
        current,
        selectedNodeId,
        nodeExecutionStates,
        quizQuestionSlots,
        gradeSlotUsesRubric,
      ),
    )
  }, [gradeSlotUsesRubric, nodeExecutionStates, quizQuestionSlots, selectedNodeId, setNodes])

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

      if (
        actionable.some(
          (change) => change.type === 'position' && 'dragging' in change && change.dragging === false,
        )
      ) {
        isDraggingNodeRef.current = false
      } else if (
        actionable.some(
          (change) => change.type === 'position' && 'dragging' in change && change.dragging === true,
        )
      ) {
        isDraggingNodeRef.current = true
      }

      const currentNodes = nodesRef.current
      const nextNodes = applyNodeChanges(actionable, currentNodes)
      nodesRef.current = nextNodes
      setNodes(nextNodes)

      if (!readOnly) {
        const persistable = actionable.filter(shouldPersistNodeChange)
        if (persistable.length > 0) {
          isDraggingNodeRef.current = false
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
      setSelectedNodeIds(selectedNodes.map((n) => n.id))
    },
    [setSelectedNodeId],
  )

  const groupableSelection = useMemo(() => {
    if (readOnly || selectedNodeIds.length < 2) return []
    const types = new Map(nodes.map((n) => [n.id, n.type]))
    if (selectedNodeIds.some((id) => types.get(id) === 'output')) return []
    return selectedNodeIds
  }, [nodes, readOnly, selectedNodeIds])

  const onNodeDoubleClick = useCallback(
    (_event: ReactMouseEvent, node: Node) => {
      if (node.type === 'group') enterGroup(node.id)
    },
    [enterGroup],
  )

  const handleGroupSelection = useCallback(() => {
    if (groupableSelection.length < 2) return
    if (groupSelection(groupableSelection, t('gradingAgent.canvas.nodes.group.defaultLabel'))) {
      setSelectedNodeIds([])
    }
  }, [groupSelection, groupableSelection, t])

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
        if (item.kind === 'openGroup') enterGroup(contextMenu.nodeId)
        if (item.kind === 'ungroup') ungroup(contextMenu.nodeId)
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
      enterGroup,
      graph,
      removeEdge,
      removeNode,
      requestNodeRename,
      setSelectedNodeId,
      ungroup,
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
      className="relative h-full min-h-0 w-full overflow-hidden rounded-xl border border-slate-200 bg-slate-50 dark:border-neutral-700 dark:bg-neutral-950"
      onDragEnter={onDragOver}
      onDragOver={onDragOver}
      onDrop={onDrop}
    >
      {navStack.length > 0 ? (
        <nav
          aria-label={t('gradingAgent.canvas.group.breadcrumbLabel')}
          className="pointer-events-auto absolute left-3 top-3 z-10 flex flex-wrap items-center gap-1 rounded-lg border border-slate-200 bg-white/95 px-2 py-1 text-xs shadow-sm backdrop-blur dark:border-neutral-700 dark:bg-neutral-900/95"
        >
          <button
            type="button"
            onClick={() => exitToDepth(0)}
            className="rounded px-1.5 py-0.5 font-medium text-indigo-600 hover:bg-indigo-50 dark:text-indigo-300 dark:hover:bg-indigo-950/40"
          >
            {t('gradingAgent.canvas.group.rootCrumb')}
          </button>
          {navStack.map((entry, idx) => (
            <span key={`${entry.groupId}-${idx}`} className="flex items-center gap-1">
              <span aria-hidden className="text-slate-400">
                /
              </span>
              <button
                type="button"
                onClick={() => exitToDepth(idx + 1)}
                disabled={idx === navStack.length - 1}
                className="rounded px-1.5 py-0.5 font-medium text-slate-700 enabled:hover:bg-slate-100 disabled:font-semibold disabled:text-slate-900 dark:text-neutral-300 dark:enabled:hover:bg-neutral-800 dark:disabled:text-neutral-50"
              >
                {entry.label}
              </button>
            </span>
          ))}
        </nav>
      ) : null}
      {groupableSelection.length >= 2 ? (
        <button
          type="button"
          onClick={handleGroupSelection}
          className="absolute right-3 top-3 z-10 inline-flex items-center gap-1.5 rounded-lg bg-fuchsia-600 px-3 py-1.5 text-xs font-semibold text-white shadow-lg hover:bg-fuchsia-700"
        >
          {t('gradingAgent.canvas.group.groupSelection', { count: groupableSelection.length })}
        </button>
      ) : null}
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
          onNodeDoubleClick={onNodeDoubleClick}
          nodesDraggable={!readOnly}
          nodesConnectable={!readOnly}
          elementsSelectable={!readOnly}
          panOnDrag={[1, 2]}
          panActivationKeyCode="Space"
          panOnScroll
          selectionOnDrag
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
