/* eslint-disable react-refresh/only-export-components -- context module exports provider + hook */
import { createContext, useContext, type ReactNode } from 'react'

type WorkflowCanvasContextValue = {
  readOnly: boolean
  onNodeLabelChange: (nodeId: string, label: string | null) => void
  renameRequestNodeId: string | null
  requestNodeRename: (nodeId: string) => void
  clearRenameRequest: () => void
}

const WorkflowCanvasContext = createContext<WorkflowCanvasContextValue | null>(null)

export function WorkflowCanvasProvider({
  readOnly,
  onNodeLabelChange,
  renameRequestNodeId,
  requestNodeRename,
  clearRenameRequest,
  children,
}: WorkflowCanvasContextValue & { children: ReactNode }) {
  return (
    <WorkflowCanvasContext.Provider
      value={{ readOnly, onNodeLabelChange, renameRequestNodeId, requestNodeRename, clearRenameRequest }}
    >
      {children}
    </WorkflowCanvasContext.Provider>
  )
}

export function useWorkflowCanvas(): WorkflowCanvasContextValue {
  const value = useContext(WorkflowCanvasContext)
  if (!value) {
    throw new Error('useWorkflowCanvas must be used within WorkflowCanvasProvider')
  }
  return value
}