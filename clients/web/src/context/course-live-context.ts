import { createContext, useContext } from 'react'

export type CourseLiveValue = {
  /** Bumps when course modules/items change over WebSocket. */
  structureRevision: number
}

export const CourseLiveContext = createContext<CourseLiveValue>({ structureRevision: 0 })

export function useCourseLiveStructureRevision(): number {
  return useContext(CourseLiveContext).structureRevision
}
