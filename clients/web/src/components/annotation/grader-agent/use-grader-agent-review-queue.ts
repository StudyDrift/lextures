import { useCallback, useEffect, useState } from 'react'
import {
  fetchGraderAgentReviewQueue,
  fetchGraderAgentRuns,
  type GraderAgentReviewQueueResponse,
  type GraderAgentRunHistoryEntry,
} from '../../../lib/courses-api'

type UseGraderAgentReviewQueueArgs = {
  enabled: boolean
  courseCode: string
  itemId: string
}

export function useGraderAgentReviewQueue({
  enabled,
  courseCode,
  itemId,
}: UseGraderAgentReviewQueueArgs) {
  const [queue, setQueue] = useState<GraderAgentReviewQueueResponse | null>(null)
  const [runs, setRuns] = useState<GraderAgentRunHistoryEntry[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const refresh = useCallback(async () => {
    if (!enabled) return
    setLoading(true)
    setError(null)
    try {
      const [queueRes, runsRes] = await Promise.all([
        fetchGraderAgentReviewQueue(courseCode, itemId),
        fetchGraderAgentRuns(courseCode, itemId),
      ])
      setQueue(queueRes)
      setRuns(runsRes.runs)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load review queue.')
    } finally {
      setLoading(false)
    }
  }, [courseCode, enabled, itemId])

  useEffect(() => {
    if (!enabled) {
      setQueue(null)
      setRuns([])
      setError(null)
      return
    }
    void refresh()
  }, [enabled, refresh])

  return {
    held: queue?.held ?? [],
    flagged: queue?.flagged ?? [],
    totalCount: queue?.totalCount ?? 0,
    runs,
    loading,
    error,
    refresh,
  }
}