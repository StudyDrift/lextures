import { useEffect, useState } from 'react'
import { fetchMessagingPolicy, type MessagingPolicy } from '../lib/communication-api'
import { useManagedLearnerSession } from './use-managed-learner-session'

export type MessagingPolicyState = MessagingPolicy & { loading: boolean; error: string | null }

const UNRESTRICTED: MessagingPolicy = { restricted: false, recipients: [] }

/**
 * Who Compose may address (#775). Adults are unrestricted (free-text To). In a managed-learner
 * session the server returns the parent and approved teachers; until it answers, the kid
 * session is treated as restricted so no free-text address field ever flashes.
 */
export function useMessagingPolicy(revision = 0): MessagingPolicyState {
  const kidSession = useManagedLearnerSession()
  const [state, setState] = useState<MessagingPolicyState>({
    ...UNRESTRICTED,
    restricted: kidSession,
    loading: true,
    error: null,
  })

  useEffect(() => {
    let cancelled = false
    void (async () => {
      try {
        const policy = await fetchMessagingPolicy()
        if (!cancelled) setState({ ...policy, restricted: policy.restricted || kidSession, loading: false, error: null })
      } catch {
        if (!cancelled) {
          setState({
            ...UNRESTRICTED,
            restricted: kidSession,
            loading: false,
            error: kidSession ? 'Could not load who you can message.' : null,
          })
        }
      }
    })()
    return () => {
      cancelled = true
    }
  }, [kidSession, revision])

  return state
}
