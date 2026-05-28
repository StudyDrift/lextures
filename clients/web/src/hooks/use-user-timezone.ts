import { useContext } from 'react'
import { UserTimezoneContext } from '../context/user-timezone-context'

export function useUserTimezone() {
  const ctx = useContext(UserTimezoneContext)
  if (!ctx) {
    return {
      timezone: null,
      loading: false,
      setTimezone: async () => {},
      refresh: async () => {},
    }
  }
  return ctx
}
