import { createContext } from 'react'

export type UserTimezoneContextValue = {
  timezone: string | null
  loading: boolean
  setTimezone: (tz: string | null) => Promise<void>
  refresh: () => Promise<void>
}

export const UserTimezoneContext = createContext<UserTimezoneContextValue | null>(null)
