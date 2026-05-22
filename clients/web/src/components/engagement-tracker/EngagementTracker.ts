import { postEngagementEvents, type EngagementEvent } from '../../lib/engagement-api'

const HEARTBEAT_INTERVAL_MS = 30_000
const FLUSH_INTERVAL_MS = 60_000
const SCROLL_THROTTLE_MS = 500

type TrackerOptions = {
  courseId?: string
  itemId?: string
  itemType?: EngagementEvent['itemType']
}

/**
 * EngagementTracker collects heartbeat, scroll-depth, and video-progress events
 * for a content page view. Call start() when the page mounts and stop() when it
 * unmounts.
 *
 * Usage:
 *   const tracker = new EngagementTracker({ courseId, itemId, itemType: 'content_page' })
 *   tracker.start()
 *   // ...
 *   tracker.stop()
 */
export class EngagementTracker {
  private readonly opts: TrackerOptions
  private queue: EngagementEvent[] = []
  private heartbeatTimer: ReturnType<typeof setInterval> | null = null
  private flushTimer: ReturnType<typeof setInterval> | null = null
  private scrollThrottleTimer: ReturnType<typeof setTimeout> | null = null
  private maxScrollDepth = 0
  private active = false

  constructor(opts: TrackerOptions = {}) {
    this.opts = opts
  }

  start(): void {
    if (this.active) return
    this.active = true

    this.heartbeatTimer = setInterval(() => {
      if (document.visibilityState === 'visible') {
        this.enqueue({ eventType: 'heartbeat' })
      }
    }, HEARTBEAT_INTERVAL_MS)

    this.flushTimer = setInterval(() => void this.flush(), FLUSH_INTERVAL_MS)

    document.addEventListener('visibilitychange', this.handleVisibilityChange)
    window.addEventListener('scroll', this.handleScroll, { passive: true })
    window.addEventListener('beforeunload', this.handleUnload)
  }

  stop(): void {
    if (!this.active) return
    this.active = false

    if (this.heartbeatTimer !== null) clearInterval(this.heartbeatTimer)
    if (this.flushTimer !== null) clearInterval(this.flushTimer)
    if (this.scrollThrottleTimer !== null) clearTimeout(this.scrollThrottleTimer)

    document.removeEventListener('visibilitychange', this.handleVisibilityChange)
    window.removeEventListener('scroll', this.handleScroll)
    window.removeEventListener('beforeunload', this.handleUnload)

    this.recordScrollDepth()
    void this.flush()
  }

  /** Call from video player on timeupdate events. */
  recordVideoProgress(currentSecond: number, durationSecond: number): void {
    if (durationSecond <= 0) return
    const pct = Math.min(100, (currentSecond / durationSecond) * 100)
    this.enqueue({ eventType: 'video_progress', value: pct })
  }

  private enqueue(partial: Omit<EngagementEvent, 'courseId' | 'itemId' | 'itemType'>): void {
    this.queue.push({
      ...partial,
      courseId: this.opts.courseId,
      itemId: this.opts.itemId,
      itemType: this.opts.itemType,
      occurredAt: new Date().toISOString(),
    })
  }

  private recordScrollDepth(): void {
    const scrollY = window.scrollY
    const docHeight = document.documentElement.scrollHeight - window.innerHeight
    if (docHeight <= 0) return
    const depth = Math.min(100, Math.round((scrollY / docHeight) * 100))
    if (depth > this.maxScrollDepth) {
      this.maxScrollDepth = depth
      this.enqueue({ eventType: 'scroll_depth', value: depth })
    }
  }

  private async flush(): Promise<void> {
    if (this.queue.length === 0) return
    const batch = this.queue.splice(0)
    try {
      await postEngagementEvents(batch)
    } catch {
      // Failures are acceptable; events are approximate.
      // Re-queue up to 50 events on failure to retry on next flush.
      this.queue.unshift(...batch.slice(0, 50 - this.queue.length))
    }
  }

  private handleVisibilityChange = (): void => {
    if (document.visibilityState === 'hidden') {
      this.recordScrollDepth()
      void this.flush()
    }
  }

  private handleScroll = (): void => {
    if (this.scrollThrottleTimer !== null) return
    this.scrollThrottleTimer = setTimeout(() => {
      this.scrollThrottleTimer = null
      this.recordScrollDepth()
    }, SCROLL_THROTTLE_MS)
  }

  private handleUnload = (): void => {
    this.recordScrollDepth()
    // Best-effort synchronous flush via sendBeacon.
    if (this.queue.length > 0 && typeof navigator.sendBeacon === 'function') {
      const token = getAccessTokenSync()
      if (token) {
        const blob = new Blob([JSON.stringify(this.queue)], { type: 'application/json' })
        // sendBeacon doesn't support custom headers; include token as query param for unload only.
        navigator.sendBeacon(`/api/v1/analytics/events?_token=${encodeURIComponent(token)}`, blob)
        this.queue = []
      }
    }
  }
}

function getAccessTokenSync(): string | null {
  try {
    // Mirror the pattern from lib/auth.ts getAccessToken().
    return sessionStorage.getItem('access_token') ?? localStorage.getItem('access_token')
  } catch {
    return null
  }
}
