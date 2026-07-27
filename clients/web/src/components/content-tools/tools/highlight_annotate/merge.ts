import type { MergeReducer } from '../../host/conflict-policy'
import { defaultMergeReducer } from '../../host/conflict-policy'

type AnnotationLike = {
  id?: string
  createdAt?: string
  [key: string]: unknown
}

function asAnnotations(v: unknown): AnnotationLike[] {
  if (!Array.isArray(v)) return []
  return v.filter((a) => a && typeof a === 'object') as AnnotationLike[]
}

/** Merge annotation arrays by id (client edit wins; unique ids from both sides kept). */
export function mergeAnnotationsById(
  client: AnnotationLike[],
  server: AnnotationLike[],
): AnnotationLike[] {
  const byId = new Map<string, AnnotationLike>()
  const order: string[] = []

  for (const a of server) {
    const id = typeof a.id === 'string' ? a.id.trim() : ''
    if (!id) continue
    byId.set(id, a)
    order.push(id)
  }
  for (const a of client) {
    const id = typeof a.id === 'string' ? a.id.trim() : ''
    if (!id) continue
    if (!byId.has(id)) order.push(id)
    byId.set(id, a)
  }
  return order.map((id) => byId.get(id)!).filter(Boolean)
}

/** CT.13 merge reducer: shallow merge + annotations by id. */
export const highlightAnnotateMergeReducer: MergeReducer = (client, server) => {
  const base = defaultMergeReducer(client, server)
  return {
    ...base,
    v: 1,
    annotations: mergeAnnotationsById(
      asAnnotations(client.annotations),
      asAnnotations(server.annotations),
    ),
  }
}
