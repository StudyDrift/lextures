/** CT.3 — client-side revision conflict resolution. */

export type ConflictPolicy = 'server_wins' | 'client_wins' | 'merge'

export type MergeReducer = (
  client: Record<string, unknown>,
  server: Record<string, unknown>,
) => Record<string, unknown>

/** Default merge: server base with client keys winning on overlap. */
export function defaultMergeReducer(
  client: Record<string, unknown>,
  server: Record<string, unknown>,
): Record<string, unknown> {
  return { ...server, ...client }
}

export function resolveConflictState(
  policy: ConflictPolicy,
  client: Record<string, unknown>,
  server: Record<string, unknown>,
  mergeFn: MergeReducer = defaultMergeReducer,
): Record<string, unknown> {
  switch (policy) {
    case 'server_wins':
      return { ...server }
    case 'client_wins':
      return { ...client }
    case 'merge':
      return mergeFn(client, server)
    default: {
      const _exhaustive: never = policy
      return _exhaustive
    }
  }
}
