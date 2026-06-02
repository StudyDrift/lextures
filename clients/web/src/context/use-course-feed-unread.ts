import { useContext } from 'react'
import { CourseFeedUnreadContext } from './course-feed-unread-context'

const noop = () => {}

export function useCourseFeedUnread() {
  const ctx = useContext(CourseFeedUnreadContext)
  if (!ctx) {
    return {
      feedUnreadForChannel: () => 0,
      clearFeedChannelUnread: noop,
      clearAllFeedUnread: noop,
      setViewedFeedChannel: noop,
      totalFeedUnread: 0,
    } as const
  }
  return ctx
}
