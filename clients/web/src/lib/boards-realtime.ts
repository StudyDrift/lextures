/**
 * Board realtime — Y.js CRDT + awareness over the Go WebSocket relay (VC.4).
 * REST mutations also push JSON `board.changed` text frames so peers refetch
 * post bodies without relying solely on the mutating client's CRDT publish.
 */
import { useEffect, useReducer, useRef, useState } from 'react'
import * as Y from 'yjs'
import { WebsocketProvider } from 'y-websocket'
import type { Awareness } from 'y-protocols/awareness'
import { wsUrl } from './api'
import { getAccessToken, getJwtSubject } from './auth'
import type { BoardPost, BoardPostPosition, ArrangeBoardPostInput } from './boards-api'
import { colorForUser } from '../components/collab/collab-utils'

export const BOARD_POSTS_MAP = 'posts'

export type BoardConnState = 'connecting' | 'connected' | 'disconnected' | 'offline'

export type BoardPresenceUser = {
  userId: string
  displayName: string
  color: string
  cursor?: { x: number; y: number } | null
  selectionPostId?: string | null
}

export type BoardArrangement = {
  id: string
  sectionId?: string | null
  sortIndex?: number
  position?: BoardPostPosition | null
  eventDate?: string | null
  lat?: number | null
  lng?: number | null
  deleted?: boolean
  /** ISO timestamp — newer CRDT/REST arrangement wins when merging. */
  updatedAt?: string
}

export type BoardChangedEvent = {
  type: 'board.changed'
  reason: string
  postId?: string
}

/** Base WebSocket URL; y-websocket appends `/{roomname}` (use roomname `ws`). */
export function boardWsUrl(courseCode: string, boardId: string): string {
  // Route through the shared helper so local Vite `/api` proxy upgrades work
  // (direct :8080 WS fails when that port is SSH-tunneled without upgrades).
  return wsUrl(`/api/v1/courses/${encodeURIComponent(courseCode)}/boards/${encodeURIComponent(boardId)}`)
}

export function parseBoardChangedEvent(raw: string): BoardChangedEvent | null {
  try {
    const msg = JSON.parse(raw) as { type?: string; reason?: string; postId?: string }
    if (msg.type !== 'board.changed' || typeof msg.reason !== 'string') return null
    return {
      type: 'board.changed',
      reason: msg.reason,
      postId: typeof msg.postId === 'string' ? msg.postId : undefined,
    }
  } catch {
    return null
  }
}

function buildAuthWebSocket(
  token: string,
  onTextMessage: (data: string) => void,
): typeof WebSocket {
  return class AuthWebSocket extends WebSocket {
    constructor(url: string | URL, protocols?: string | string[]) {
      super(url, protocols)
      this.addEventListener('open', () => {
        this.send(JSON.stringify({ authToken: token }))
      })
      this.addEventListener('message', (ev: MessageEvent) => {
        if (typeof ev.data === 'string') {
          onTextMessage(ev.data)
        }
      })
    }
  }
}

export function clonePosition(position: BoardPostPosition | null | undefined): BoardPostPosition | null | undefined {
  if (position == null) return position
  return { x: position.x, y: position.y, w: position.w, h: position.h }
}

export function arrangementFromPost(post: BoardPost): BoardArrangement {
  return {
    id: post.id,
    sectionId: post.sectionId ?? null,
    sortIndex: post.sortIndex,
    position: clonePosition(post.position) ?? null,
    eventDate: post.eventDate ?? null,
    lat: post.lat ?? null,
    lng: post.lng ?? null,
    updatedAt: post.updatedAt,
  }
}

export function applyArrangementToPost(post: BoardPost, arr: BoardArrangement): BoardPost {
  return {
    ...post,
    sectionId: arr.sectionId === undefined ? post.sectionId : (arr.sectionId ?? undefined),
    sortIndex: arr.sortIndex ?? post.sortIndex,
    position: arr.position === undefined ? post.position : (clonePosition(arr.position) ?? undefined),
    eventDate: arr.eventDate === undefined ? post.eventDate : (arr.eventDate ?? undefined),
    lat: arr.lat === undefined ? post.lat : (arr.lat ?? undefined),
    lng: arr.lng === undefined ? post.lng : (arr.lng ?? undefined),
    updatedAt: arr.updatedAt && (!post.updatedAt || arr.updatedAt > post.updatedAt) ? arr.updatedAt : post.updatedAt,
  }
}

/** Prefer CRDT arrangement when it is at least as new as the REST post. */
export function shouldApplyCrdtArrangement(post: BoardPost, arr: BoardArrangement): boolean {
  if (arr.deleted) return true
  if (!arr.updatedAt) return true
  if (!post.updatedAt) return true
  return arr.updatedAt >= post.updatedAt
}

export function mergePostsWithCrdt(posts: BoardPost[], arrangements: Map<string, BoardArrangement>): BoardPost[] {
  const byId = new Map(posts.map((p) => [p.id, p]))
  for (const [id, arr] of arrangements) {
    if (arr.deleted) {
      byId.delete(id)
      continue
    }
    const existing = byId.get(id)
    if (existing && shouldApplyCrdtArrangement(existing, arr)) {
      byId.set(id, applyArrangementToPost(existing, arr))
    }
  }
  return Array.from(byId.values())
}

function readArrangements(ydoc: Y.Doc): Map<string, BoardArrangement> {
  const posts = ydoc.getMap<BoardArrangement>(BOARD_POSTS_MAP)
  const out = new Map<string, BoardArrangement>()
  posts.forEach((value, key) => {
    if (value && typeof value === 'object') {
      out.set(key, { ...value, id: value.id || key })
    }
  })
  return out
}

type Session = {
  ydoc: Y.Doc
  provider: WebsocketProvider
}

type SessionAction = { type: 'set'; session: Session } | { type: 'clear' }

function sessionReducer(_state: Session | null, action: SessionAction): Session | null {
  if (action.type === 'set') return action.session
  return null
}

export type UseBoardRealtimeOptions = {
  courseCode: string
  boardId: string
  enabled: boolean
  displayName: string
  posts: BoardPost[]
  onRemoteCardAdded?: () => void
  /** Called when CRDT introduces a post id not yet in local REST state (refetch bodies). */
  onUnknownPostIds?: (ids: string[]) => void
  /** Called when the server pushes a REST-originated board.changed event. */
  onBoardChanged?: (event: BoardChangedEvent) => void
}

export type UseBoardRealtimeResult = {
  connState: BoardConnState
  awareness: Awareness | null
  arrangements: Map<string, BoardArrangement>
  mergedPosts: BoardPost[]
  publishArrangement: (postId: string, input: ArrangeBoardPostInput | BoardArrangement) => void
  publishPostCreated: (post: BoardPost) => void
  publishPostDeleted: (postId: string) => void
  setCursor: (cursor: { x: number; y: number } | null) => void
  setSelectionPostId: (postId: string | null) => void
}

export function useBoardRealtime(opts: UseBoardRealtimeOptions): UseBoardRealtimeResult {
  const {
    courseCode,
    boardId,
    enabled,
    displayName,
    posts,
    onRemoteCardAdded,
    onUnknownPostIds,
    onBoardChanged,
  } = opts
  const [session, dispatchSession] = useReducer(sessionReducer, null)
  const [connState, setConnState] = useState<BoardConnState>('connecting')
  const [arrangements, setArrangements] = useState<Map<string, BoardArrangement>>(new Map())
  const seededRef = useRef(false)
  const prevIdsRef = useRef<Set<string>>(new Set())
  const postsRef = useRef(posts)
  postsRef.current = posts
  const announceRef = useRef(onRemoteCardAdded)
  announceRef.current = onRemoteCardAdded
  const unknownRef = useRef(onUnknownPostIds)
  unknownRef.current = onUnknownPostIds
  const changedRef = useRef(onBoardChanged)
  changedRef.current = onBoardChanged

  const wsBase = boardWsUrl(courseCode, boardId)
  const userId = getJwtSubject() ?? ''

  useEffect(() => {
    if (!enabled || !courseCode || !boardId) {
      dispatchSession({ type: 'clear' })
      setConnState('offline')
      return
    }
    seededRef.current = false
    const ydoc = new Y.Doc()
    const token = getAccessToken() ?? ''
    const provider = new WebsocketProvider(wsBase, 'ws', ydoc, {
      connect: true,
      params: { token },
      WebSocketPolyfill: buildAuthWebSocket(token, (raw) => {
        const ev = parseBoardChangedEvent(raw)
        if (ev) changedRef.current?.(ev)
      }),
    })

    const rafId = requestAnimationFrame(() => {
      dispatchSession({ type: 'set', session: { ydoc, provider } })
    })

    provider.on('status', ({ status }: { status: string }) => {
      setConnState(status === 'connected' ? 'connected' : status === 'connecting' ? 'connecting' : 'disconnected')
    })

    const postsMap = ydoc.getMap(BOARD_POSTS_MAP)
    const syncArrangements = () => {
      const next = readArrangements(ydoc)
      setArrangements(next)
      const ids = new Set(next.keys())
      const known = new Set(postsRef.current.map((p) => p.id))
      const unknown: string[] = []
      for (const id of ids) {
        const arr = next.get(id)
        if (arr?.deleted) continue
        if (!prevIdsRef.current.has(id)) {
          announceRef.current?.()
        }
        if (!known.has(id)) unknown.push(id)
      }
      if (unknown.length > 0) unknownRef.current?.(unknown)
      prevIdsRef.current = ids
    }
    postsMap.observe(syncArrangements)
    syncArrangements()

    return () => {
      cancelAnimationFrame(rafId)
      postsMap.unobserve(syncArrangements)
      dispatchSession({ type: 'clear' })
      provider.destroy()
      ydoc.destroy()
    }
  }, [enabled, courseCode, boardId, wsBase])

  useEffect(() => {
    if (!session?.provider.awareness || !enabled) return
    const color = colorForUser(displayName || userId || 'anon')
    session.provider.awareness.setLocalStateField('user', {
      userId,
      displayName: displayName || 'Anonymous',
      color,
    } satisfies BoardPresenceUser)
  }, [session, enabled, displayName, userId])

  // Seed CRDT from REST posts once after connect when the map is empty.
  useEffect(() => {
    if (!session || !enabled || seededRef.current || connState !== 'connected') return
    const postsMap = session.ydoc.getMap(BOARD_POSTS_MAP)
    if (postsMap.size > 0) {
      seededRef.current = true
      return
    }
    if (posts.length === 0) {
      seededRef.current = true
      return
    }
    session.ydoc.transact(() => {
      for (const post of posts) {
        postsMap.set(post.id, arrangementFromPost(post))
      }
    })
    seededRef.current = true
  }, [session, enabled, connState, posts])

  // Heal stale CRDT arrangement from newer REST reads (e.g. after board.changed refetch).
  useEffect(() => {
    if (!session || !enabled || connState !== 'connected' || posts.length === 0) return
    const postsMap = session.ydoc.getMap<BoardArrangement>(BOARD_POSTS_MAP)
    session.ydoc.transact(() => {
      for (const post of posts) {
        const prev = postsMap.get(post.id)
        if (prev?.deleted) continue
        const fromPost = arrangementFromPost(post)
        if (!prev) {
          postsMap.set(post.id, fromPost)
          continue
        }
        if (!prev.updatedAt || (fromPost.updatedAt && fromPost.updatedAt > prev.updatedAt)) {
          postsMap.set(post.id, { ...fromPost, deleted: prev.deleted })
        }
      }
    })
  }, [session, enabled, connState, posts])

  function publishArrangement(postId: string, input: ArrangeBoardPostInput | BoardArrangement) {
    if (!session) return
    const postsMap = session.ydoc.getMap<BoardArrangement>(BOARD_POSTS_MAP)
    const prev = postsMap.get(postId) ?? { id: postId }
    const next: BoardArrangement = { ...prev, id: postId }
    if ('sectionId' in input && input.sectionId !== undefined) next.sectionId = input.sectionId
    if ('sortIndex' in input && input.sortIndex !== undefined) next.sortIndex = input.sortIndex
    if ('position' in input && input.position !== undefined) {
      next.position = clonePosition(input.position) ?? null
    }
    if ('eventDate' in input && input.eventDate !== undefined) {
      next.eventDate = input.eventDate
    }
    if ('lat' in input && input.lat !== undefined) next.lat = input.lat
    if ('lng' in input && input.lng !== undefined) next.lng = input.lng
    if ('clearGeo' in input && input.clearGeo) {
      next.lat = null
      next.lng = null
    }
    if ('deleted' in input && input.deleted !== undefined) next.deleted = input.deleted
    next.updatedAt =
      'updatedAt' in input && typeof input.updatedAt === 'string' && input.updatedAt
        ? input.updatedAt
        : new Date().toISOString()
    session.ydoc.transact(() => {
      postsMap.set(postId, next)
    })
  }

  function publishPostCreated(post: BoardPost) {
    if (!session) return
    session.ydoc.transact(() => {
      session.ydoc.getMap(BOARD_POSTS_MAP).set(post.id, arrangementFromPost(post))
    })
  }

  function publishPostDeleted(postId: string) {
    if (!session) return
    session.ydoc.transact(() => {
      const postsMap = session.ydoc.getMap<BoardArrangement>(BOARD_POSTS_MAP)
      const prev = postsMap.get(postId) ?? { id: postId }
      postsMap.set(postId, {
        ...prev,
        id: postId,
        deleted: true,
        updatedAt: new Date().toISOString(),
      })
    })
  }

  function setCursor(cursor: { x: number; y: number } | null) {
    session?.provider.awareness?.setLocalStateField('cursor', cursor)
  }

  function setSelectionPostId(postId: string | null) {
    session?.provider.awareness?.setLocalStateField('selectionPostId', postId)
  }

  return {
    connState: enabled ? connState : 'offline',
    awareness: session?.provider.awareness ?? null,
    arrangements,
    mergedPosts: mergePostsWithCrdt(posts, arrangements),
    publishArrangement,
    publishPostCreated,
    publishPostDeleted,
    setCursor,
    setSelectionPostId,
  }
}
