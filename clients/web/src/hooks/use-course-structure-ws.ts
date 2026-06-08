import { useEffect, useState } from 'react'
import { getAccessToken } from '../lib/auth'
import { wsUrl } from '../lib/api'

/** Increments when the server broadcasts structure_changed for a course (e.g. Canvas import). */
export function useCourseStructureRevision(courseCode: string | undefined): number {
  const [revision, setRevision] = useState(0)

  useEffect(() => {
    if (!courseCode) return
    const token = getAccessToken()
    if (!token) return

    const ws = new WebSocket(wsUrl(`/api/v1/courses/${encodeURIComponent(courseCode)}/structure/ws`))
    ws.onopen = () => {
      ws.send(JSON.stringify({ authToken: token }))
    }
    ws.onmessage = (ev) => {
      try {
        const data = JSON.parse(String(ev.data)) as { type?: string }
        if (data.type === 'structure_changed') {
          setRevision((r) => r + 1)
        }
      } catch {
        /* ignore malformed */
      }
    }
    return () => {
      ws.close()
    }
  }, [courseCode])

  return revision
}
