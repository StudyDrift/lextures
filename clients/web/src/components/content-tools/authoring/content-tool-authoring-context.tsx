/* eslint-disable react-refresh/only-export-components -- context module */
import { createContext, useContext, type ReactNode } from 'react'
import type {
  ContentToolInstance,
  ContentToolManifest,
  ContentToolsCatalogTool,
} from '../../../lib/courses-api'

export type ContentToolAuthoringContextValue = {
  courseCode: string
  instances: Record<string, ContentToolInstance>
  catalog: ContentToolsCatalogTool[]
  manifests: Record<string, ContentToolManifest>
  onConfigure: (instanceId: string) => void
  onPreview: (instanceId: string) => void
  onDuplicate: (instanceId: string) => void
  onDelete: (instanceId: string) => void
  upsertInstance: (instance: ContentToolInstance) => void
  removeInstance: (instanceId: string) => void
  cacheManifest: (manifest: ContentToolManifest) => void
}

const ContentToolAuthoringContext = createContext<ContentToolAuthoringContextValue | null>(null)

export function ContentToolAuthoringProvider({
  value,
  children,
}: {
  value: ContentToolAuthoringContextValue
  children: ReactNode
}) {
  return (
    <ContentToolAuthoringContext.Provider value={value}>
      {children}
    </ContentToolAuthoringContext.Provider>
  )
}

export function useContentToolAuthoring(): ContentToolAuthoringContextValue | null {
  return useContext(ContentToolAuthoringContext)
}

export function useContentToolAuthoringRequired(): ContentToolAuthoringContextValue {
  const ctx = useContext(ContentToolAuthoringContext)
  if (!ctx) {
    throw new Error('useContentToolAuthoringRequired requires ContentToolAuthoringProvider')
  }
  return ctx
}
