import { useEffect, useRef, useState } from 'react'
import {
  ContentToolRevisionConflictError,
  getContentToolState,
  putContentToolState,
  submitContentToolState,
  type ContentToolScore,
  type ContentToolState,
} from '../../../lib/courses-api'
import {
  resolveConflictState,
  type ConflictPolicy,
  type MergeReducer,
} from './conflict-policy'

export type ToolSyncStatus = 'idle' | 'saving' | 'saved' | 'unsynced' | 'error'

export type UseToolStateOptions = {
  courseCode: string
  instanceId: string
  /** Prefetched envelope from the page batch (`withState=1`). */
  initialEnvelope?: ContentToolState | null
  debounceMs?: number
  conflictPolicy?: ConflictPolicy
  mergeFn?: MergeReducer
  readOnly?: boolean
  announce?: (message: string) => void
  savedAnnouncement?: string
  scope?: 'enrollment' | 'preview'
}

export type UseToolStateResult = {
  state: Record<string, unknown>
  status: string
  score: ContentToolScore | null
  revision: number
  syncStatus: ToolSyncStatus
  loading: boolean
  error: string | null
  save: (patch: Record<string, unknown>) => void
  submit: (patch: Record<string, unknown>) => Promise<void>
  flush: () => Promise<void>
  applyEnvelope: (envelope: ContentToolState) => void
}

const DEFAULT_DEBOUNCE_MS = 1500

function clampDebounce(ms: number | undefined): number {
  if (ms == null || !Number.isFinite(ms)) return DEFAULT_DEBOUNCE_MS
  return Math.min(10000, Math.max(500, Math.round(ms)))
}

function emptyEnvelope(instanceId: string): ContentToolState {
  return {
    instanceId,
    revision: 0,
    status: 'not_started',
    state: {},
    stateJson: {},
    score: null,
    updatedAt: null,
    resetCount: 0,
    lastResetAt: null,
  }
}

export function useToolState(opts: UseToolStateOptions): UseToolStateResult {
  const {
    courseCode,
    instanceId,
    initialEnvelope,
    debounceMs,
    conflictPolicy = 'server_wins',
    mergeFn,
    readOnly = false,
    announce,
    savedAnnouncement,
    scope,
  } = opts

  const [envelope, setEnvelope] = useState<ContentToolState>(
    () => initialEnvelope ?? emptyEnvelope(instanceId),
  )
  const [syncStatus, setSyncStatus] = useState<ToolSyncStatus>('idle')
  const [loading, setLoading] = useState(!initialEnvelope)
  const [error, setError] = useState<string | null>(null)

  const envelopeRef = useRef(envelope)
  envelopeRef.current = envelope
  const pendingRef = useRef<Record<string, unknown> | null>(null)
  const dirtyRef = useRef(false)
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const savingRef = useRef(false)
  const mountedRef = useRef(true)

  const debounce = clampDebounce(debounceMs)

  useEffect(() => {
    mountedRef.current = true
    return () => {
      mountedRef.current = false
    }
  }, [])

  useEffect(() => {
    if (initialEnvelope) {
      setEnvelope(initialEnvelope)
      setLoading(false)
      return
    }
    let cancelled = false
    setLoading(true)
    void getContentToolState(courseCode, instanceId, scope ? { scope } : undefined)
      .then((st) => {
        if (cancelled || !mountedRef.current) return
        setEnvelope(st)
        setError(null)
      })
      .catch((e: unknown) => {
        if (cancelled || !mountedRef.current) return
        setError(e instanceof Error ? e.message : 'Failed to load tool state.')
      })
      .finally(() => {
        if (cancelled || !mountedRef.current) return
        setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [courseCode, instanceId, initialEnvelope, scope])

  function applyEnvelope(next: ContentToolState) {
    setEnvelope(next)
    envelopeRef.current = next
  }

  async function persist(nextState: Record<string, unknown>, mode: 'save' | 'submit') {
    if (readOnly) return
    if (savingRef.current) {
      pendingRef.current = nextState
      dirtyRef.current = true
      return
    }
    savingRef.current = true
    setSyncStatus('saving')
    setError(null)
    const revision = envelopeRef.current.revision
    try {
      const writer = mode === 'submit' ? submitContentToolState : putContentToolState
      const result = await writer(courseCode, instanceId, {
        revision,
        state: nextState,
        ...(mode === 'save' && scope ? { scope } : {}),
      })
      if (!mountedRef.current) return
      applyEnvelope(result)
      dirtyRef.current = false
      pendingRef.current = null
      setSyncStatus('saved')
      if (savedAnnouncement) announce?.(savedAnnouncement)
    } catch (e) {
      if (!mountedRef.current) return
      if (e instanceof ContentToolRevisionConflictError) {
        const server = e.current
        const resolved = resolveConflictState(
          conflictPolicy,
          nextState,
          server.state ?? server.stateJson ?? {},
          mergeFn,
        )
        applyEnvelope({
          ...server,
          state: resolved,
          stateJson: resolved,
        })
        if (conflictPolicy === 'server_wins') {
          dirtyRef.current = false
          pendingRef.current = null
          setSyncStatus('saved')
          if (savedAnnouncement) announce?.(savedAnnouncement)
        } else {
          // Retry once with the server revision and resolved document.
          try {
            const retry = await putContentToolState(courseCode, instanceId, {
              revision: server.revision,
              state: resolved,
              ...(scope ? { scope } : {}),
            })
            if (!mountedRef.current) return
            applyEnvelope(retry)
            dirtyRef.current = false
            pendingRef.current = null
            setSyncStatus('saved')
            if (savedAnnouncement) announce?.(savedAnnouncement)
          } catch (retryErr) {
            dirtyRef.current = true
            setSyncStatus('unsynced')
            setError(retryErr instanceof Error ? retryErr.message : 'Save conflict.')
          }
        }
      } else {
        dirtyRef.current = true
        setSyncStatus('unsynced')
        setError(e instanceof Error ? e.message : 'Save failed.')
      }
    } finally {
      savingRef.current = false
      if (dirtyRef.current && pendingRef.current && mountedRef.current) {
        const queued = pendingRef.current
        pendingRef.current = null
        void persist(queued, 'save')
      }
    }
  }

  function scheduleSave(nextState: Record<string, unknown>) {
    pendingRef.current = nextState
    dirtyRef.current = true
    setSyncStatus((s) => (s === 'saving' ? s : 'unsynced'))
    if (timerRef.current) clearTimeout(timerRef.current)
    timerRef.current = setTimeout(() => {
      timerRef.current = null
      const pending = pendingRef.current
      if (pending) void persist(pending, 'save')
    }, debounce)
  }

  async function flush() {
    if (timerRef.current) {
      clearTimeout(timerRef.current)
      timerRef.current = null
    }
    const pending = pendingRef.current
    if (pending && dirtyRef.current) {
      await persist(pending, 'save')
    }
  }

  useEffect(() => {
    function onVisibility() {
      if (document.visibilityState === 'hidden') void flush()
    }
    document.addEventListener('visibilitychange', onVisibility)
    return () => {
      document.removeEventListener('visibilitychange', onVisibility)
      if (timerRef.current) clearTimeout(timerRef.current)
      // Best-effort flush on unmount (fire-and-forget).
      const pending = pendingRef.current
      if (pending && dirtyRef.current && !readOnly) {
        void putContentToolState(courseCode, instanceId, {
          revision: envelopeRef.current.revision,
          state: pending,
          ...(scope ? { scope } : {}),
        }).catch(() => {})
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- flush on unmount/visibility only
  }, [courseCode, instanceId, readOnly, scope])

  function save(patch: Record<string, unknown>) {
    if (readOnly) return
    const base = pendingRef.current ?? envelopeRef.current.state ?? {}
    const next = { ...base, ...patch }
    applyEnvelope({
      ...envelopeRef.current,
      state: next,
      stateJson: next,
      status:
        envelopeRef.current.status === 'not_started' ? 'in_progress' : envelopeRef.current.status,
    })
    scheduleSave(next)
  }

  async function submit(patch: Record<string, unknown>) {
    if (readOnly) return
    if (timerRef.current) {
      clearTimeout(timerRef.current)
      timerRef.current = null
    }
    const base = pendingRef.current ?? envelopeRef.current.state ?? {}
    const next = { ...base, ...patch }
    applyEnvelope({
      ...envelopeRef.current,
      state: next,
      stateJson: next,
    })
    await persist(next, 'submit')
  }

  return {
    state: envelope.state ?? envelope.stateJson ?? {},
    status: envelope.status,
    score: envelope.score ?? null,
    revision: envelope.revision,
    syncStatus,
    loading,
    error,
    save,
    submit,
    flush,
    applyEnvelope,
  }
}
