import { useCallback, useMemo, useState } from 'react'
import type { GraderWorkflowGraph } from './types'
import { isRubricNodeType } from './types'
import { rubricLibraryAssignmentItemId, rubricSourceMode } from './rubric-node-data'

/** Tracks library-mode rubric availability for workflow validation. */
export function useRubricLibraryRubrics(graph: GraderWorkflowGraph | null | undefined) {
  const [libraryRubrics, setLibraryRubrics] = useState<Record<string, boolean>>({})

  const libraryItemIds = useMemo(() => {
    if (!graph) return []
    const ids = new Set<string>()
    for (const node of graph.nodes) {
      if (!isRubricNodeType(node.type)) continue
      if (rubricSourceMode(node.data) !== 'library') continue
      const itemId = rubricLibraryAssignmentItemId(node.data)
      if (itemId) ids.add(itemId)
    }
    return [...ids]
  }, [graph])

  const setLibraryRubricAvailability = useCallback((itemId: string, hasRubric: boolean) => {
    setLibraryRubrics((current) => {
      if (current[itemId] === hasRubric) return current
      return { ...current, [itemId]: hasRubric }
    })
  }, [])

  const libraryRubricsResolved = useMemo(() => {
    if (libraryItemIds.length === 0) return libraryRubrics
    const next = { ...libraryRubrics }
    for (const itemId of libraryItemIds) {
      if (!(itemId in next)) next[itemId] = true
    }
    return next
  }, [libraryItemIds, libraryRubrics])

  return { libraryRubrics: libraryRubricsResolved, setLibraryRubricAvailability }
}