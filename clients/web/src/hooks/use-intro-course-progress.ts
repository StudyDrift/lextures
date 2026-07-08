import { useCallback, useEffect, useState } from 'react'
import { usePlatformFeatures } from '../context/platform-features-context'
import {
  fetchIntroCourseProgress,
  type IntroCourseProgress,
} from '../lib/intro-course-api'

export type UseIntroCourseProgressResult = {
  progress: IntroCourseProgress | null
  loading: boolean
  error: boolean
  refresh: () => void
}

export function useIntroCourseProgress(enabled = true): UseIntroCourseProgressResult {
  const { introCourseEnabled, loading: featuresLoading } = usePlatformFeatures()
  const [progress, setProgress] = useState<IntroCourseProgress | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(false)
  const [reloadToken, setReloadToken] = useState(0)

  const refresh = useCallback(() => {
    setReloadToken((n) => n + 1)
  }, [])

  useEffect(() => {
    if (!enabled || featuresLoading || !introCourseEnabled) {
      setProgress(null)
      setLoading(false)
      setError(false)
      return
    }

    let cancelled = false
    setLoading(true)
    setError(false)

    void fetchIntroCourseProgress()
      .then((data) => {
        if (!cancelled) setProgress(data)
      })
      .catch(() => {
        if (!cancelled) {
          setProgress(null)
          setError(true)
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })

    return () => {
      cancelled = true
    }
  }, [enabled, featuresLoading, introCourseEnabled, reloadToken])

  return { progress, loading, error, refresh }
}