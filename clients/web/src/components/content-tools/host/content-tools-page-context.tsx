/* eslint-disable react-refresh/only-export-components -- context module exports provider + hook */
import {
  createContext,
  useContext,
  useEffect,
  useRef,
  useState,
  type ReactNode,
} from 'react'
import {
  fetchContentToolsInstances,
  type ContentToolInstance,
} from '../../../lib/courses-api'

export type ContentToolsPageContextValue = {
  courseCode: string
  itemId?: string
  hostKind?: string
  loading: boolean
  error: string | null
  getInstance: (instanceId: string) => ContentToolInstance | undefined
  refresh: () => Promise<void>
}

const ContentToolsPageContext = createContext<ContentToolsPageContextValue | null>(null)

export type ContentToolsPageProviderProps = {
  courseCode: string
  itemId?: string
  hostKind?: string
  children: ReactNode
}

export function ContentToolsPageProvider({
  courseCode,
  itemId,
  hostKind,
  children,
}: ContentToolsPageProviderProps) {
  const [instances, setInstances] = useState<ContentToolInstance[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const mapRef = useRef<Map<string, ContentToolInstance>>(new Map())

  async function load() {
    setLoading(true)
    setError(null)
    try {
      const list = await fetchContentToolsInstances(courseCode, {
        itemId,
        hostKind,
        withState: true,
      })
      setInstances(list)
      const next = new Map<string, ContentToolInstance>()
      for (const inst of list) next.set(inst.id, inst)
      mapRef.current = next
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load content tools.')
      setInstances([])
      mapRef.current = new Map()
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void load()
    // Re-fetch when the page identity changes.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [courseCode, itemId, hostKind])

  const value: ContentToolsPageContextValue = {
    courseCode,
    itemId,
    hostKind,
    loading,
    error,
    getInstance(instanceId: string) {
      return mapRef.current.get(instanceId) ?? instances.find((i) => i.id === instanceId)
    },
    refresh: load,
  }

  return (
    <ContentToolsPageContext.Provider value={value}>{children}</ContentToolsPageContext.Provider>
  )
}

export function useContentToolsPage(): ContentToolsPageContextValue | null {
  return useContext(ContentToolsPageContext)
}
