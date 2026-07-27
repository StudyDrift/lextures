import { useCallback, useRef, useState } from 'react'
import type { ToolProps } from './contract'

/** Local state helper that mirrors host `save` semantics for in-process tools. */
export function useToolState<TState extends Record<string, unknown>>(
  props: Pick<ToolProps<Record<string, unknown>, TState>, 'state' | 'save' | 'readOnly'>,
) {
  const [local, setLocal] = useState<TState>(props.state)
  const save = useCallback(
    (patch: Partial<TState> & Record<string, unknown>) => {
      if (props.readOnly) return
      setLocal((prev: TState) => {
        const next = { ...prev, ...patch } as TState
        void props.save(next)
        return next
      })
    },
    [props],
  )
  return { state: local, setState: setLocal, save }
}

export function useToolAction(
  props: Pick<ToolProps, 'runAction' | 'readOnly'>,
) {
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const runAction = useCallback(
    async (name: string, input: Record<string, unknown> = {}) => {
      if (props.readOnly) throw new Error('read-only')
      setBusy(true)
      setError(null)
      try {
        return await props.runAction(name, input)
      } catch (e) {
        const msg = e instanceof Error ? e.message : 'action failed'
        setError(msg)
        throw e
      } finally {
        setBusy(false)
      }
    },
    [props],
  )
  return { runAction, busy, error }
}

export function useToolAnnounce(props: Pick<ToolProps, 'announce'>) {
  const announce = useCallback(
    (message: string, assertive?: boolean) => {
      props.announce(message, assertive)
    },
    [props],
  )
  return announce
}

export function useToolI18n(props: Pick<ToolProps, 't'>, namespace?: string) {
  const ns = namespace
  return useCallback(
    (key: string, options?: Record<string, unknown>) => {
      const full = ns && !key.startsWith(ns) ? `${ns}.${key}` : key
      return props.t(full, options)
    },
    [props, ns],
  )
}

/** Batches bridge traffic to the next animation frame (NFR performance). */
export function useRafBatch<T>(flush: (items: T[]) => void) {
  const queue = useRef<T[]>([])
  const scheduled = useRef(false)
  return useCallback(
    (item: T) => {
      queue.current.push(item)
      if (scheduled.current) return
      scheduled.current = true
      requestAnimationFrame(() => {
        scheduled.current = false
        const items = queue.current
        queue.current = []
        flush(items)
      })
    },
    [flush],
  )
}
