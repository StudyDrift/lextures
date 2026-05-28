import { useState } from 'react'

export function useSimplifyDialog() {
  const [open, setOpen] = useState(false)
  const [original, setOriginal] = useState('')
  const [simplified, setSimplified] = useState('')
  const [targetFkgl, setTargetFkgl] = useState(4)
  const [computedFkgl, setComputedFkgl] = useState<number | undefined>()
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  return {
    open,
    original,
    simplified,
    targetFkgl,
    computedFkgl,
    loading,
    error,
    setOpen,
    setOriginal,
    setSimplified,
    setTargetFkgl,
    setComputedFkgl,
    setLoading,
    setError,
  }
}
