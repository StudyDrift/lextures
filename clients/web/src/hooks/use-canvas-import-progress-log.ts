import { useCallback, useRef, useState } from 'react'

export type CanvasImportProgressEntry = {
  id: number
  text: string
}

export function useCanvasImportProgressLog() {
  const idRef = useRef(0)
  const [entries, setEntries] = useState<CanvasImportProgressEntry[]>([])

  const append = useCallback((text: string) => {
    const trimmed = text.trim()
    if (!trimmed) return
    setEntries((prev) => [...prev, { id: ++idRef.current, text: trimmed }])
  }, [])

  const clear = useCallback(() => {
    idRef.current = 0
    setEntries([])
  }, [])

  return { entries, append, clear }
}
