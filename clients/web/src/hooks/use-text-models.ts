import { useCallback, useEffect, useState } from 'react'
import type { ImageModelOption } from '../components/image-model-picker-utils'
import { FALLBACK_TEXT_MODEL_OPTIONS } from '../lib/ai-models'
import { authorizedFetch } from '../lib/api'
import { readApiErrorMessage } from '../lib/errors'

function fallbackTextModels(): ImageModelOption[] {
  return FALLBACK_TEXT_MODEL_OPTIONS.map((o) => ({
    id: o.id,
    name: o.label,
    contextLength: null,
    inputPricePerMillionUsd: null,
    outputPricePerMillionUsd: null,
    modalitiesSummary: null,
  }))
}

export function useTextModels(enabled = true) {
  const [models, setModels] = useState<ImageModelOption[]>(fallbackTextModels)
  const [loading, setLoading] = useState(false)

  const refresh = useCallback(async () => {
    setLoading(true)
    try {
      const res = await authorizedFetch('/api/v1/settings/ai/models?kind=text')
      const raw: unknown = await res.json().catch(() => ({}))
      if (!res.ok) {
        throw new Error(readApiErrorMessage(raw))
      }
      const list = raw as { models?: ImageModelOption[] }
      const apiModels = list.models ?? []
      setModels(apiModels.length > 0 ? apiModels : fallbackTextModels())
    } catch {
      setModels(fallbackTextModels())
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (!enabled) return
    void refresh()
  }, [enabled, refresh])

  return { models, loading, refresh }
}
