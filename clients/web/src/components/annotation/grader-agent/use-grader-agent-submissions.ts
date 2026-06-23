import { useEffect, useState } from 'react'
import {
  fetchModuleAssignmentSubmissions,
  type ModuleAssignmentSubmissionApi,
} from '../../../lib/courses-api'
import {
  defaultSubmissionIndex,
  sortSubmissionsByStudentLabel,
} from '../submission-navigator-utils'

type UseGraderAgentSubmissionsArgs = {
  open: boolean
  courseCode: string
  itemId: string
  initialSubmissionId: string | null
  enabled?: boolean
}

export function useGraderAgentSubmissions({
  open,
  courseCode,
  itemId,
  initialSubmissionId,
  enabled = true,
}: UseGraderAgentSubmissionsArgs) {
  const [submissions, setSubmissions] = useState<ModuleAssignmentSubmissionApi[]>([])
  const [index, setIndex] = useState(0)
  const [loading, setLoading] = useState(false)
  const [loadError, setLoadError] = useState<string | null>(null)

  useEffect(() => {
    if (!open || !enabled || !itemId.trim()) {
      setSubmissions([])
      setIndex(0)
      setLoadError(null)
      setLoading(false)
      return
    }

    let cancelled = false
    setLoading(true)
    setLoadError(null)

    void fetchModuleAssignmentSubmissions(courseCode, itemId, { graded: 'all' })
      .then((list) => {
        if (cancelled) return
        const sorted = sortSubmissionsByStudentLabel(list)
        setSubmissions(sorted)
        if (initialSubmissionId) {
          const found = sorted.findIndex((s) => s.id === initialSubmissionId)
          setIndex(found >= 0 ? found : defaultSubmissionIndex(sorted))
          return
        }
        setIndex(defaultSubmissionIndex(sorted))
      })
      .catch((e) => {
        if (cancelled) return
        setSubmissions([])
        setIndex(0)
        setLoadError(e instanceof Error ? e.message : 'Could not load submissions.')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })

    return () => {
      cancelled = true
    }
  }, [open, enabled, courseCode, itemId, initialSubmissionId])

  const selectedSubmission = submissions[index] ?? null
  const selectedSubmissionId = selectedSubmission?.id ?? null

  return {
    submissions,
    index,
    setIndex,
    selectedSubmission,
    selectedSubmissionId,
    loading,
    loadError,
  }
}