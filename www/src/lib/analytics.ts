/**
 * Deferred analytics loader (SEO.4 FR-7, FR-16).
 * GA4 is never on the critical path; loads via requestIdleCallback.
 */

const GA_ID = 'G-JX182Q6KKX'
const FIRST_TOUCH_KEY = 'lextures_first_touch'
const FIRST_TOUCH_MAX_AGE = 60 * 60 * 24 * 90

export type FirstTouch = {
  channel: string
  source: string
  medium: string
  campaign: string
  capturedAt: string
}

const AI_HOSTS = new Set([
  'chatgpt.com', 'chat.openai.com', 'perplexity.ai', 'gemini.google.com', 'claude.ai',
  'copilot.microsoft.com', 'you.com', 'grok.com', 'poe.com',
])

type GtagFn = (...args: unknown[]) => void

declare global {
  interface Window {
    dataLayer?: unknown[]
    gtag?: GtagFn
  }
}

let loaded = false

function hostname(value: string): string {
  try { return new URL(value).hostname.toLowerCase().replace(/^www\./, '') } catch { return '' }
}

export function classifyFirstTouch(url: URL, referrer = ''): FirstTouch {
  const source = url.searchParams.get('utm_source')?.trim().toLowerCase() || hostname(referrer)
  const medium = url.searchParams.get('utm_medium')?.trim().toLowerCase() || (source ? 'referral' : 'none')
  const campaign = url.searchParams.get('utm_campaign')?.trim() || ''
  const ai = medium === 'ai' || AI_HOSTS.has(source) || (source === 'bing.com' && (() => { try { return new URL(referrer).pathname.startsWith('/chat') } catch { return false } })())
  const organic = medium === 'organic' || /(^|\.)(google|bing|yahoo|duckduckgo)\./.test(source)
  return {
    channel: ai ? 'ai_assistant' : organic ? 'organic_search' : source ? 'referral' : 'direct',
    source: source || 'direct', medium, campaign, capturedAt: new Date().toISOString(),
  }
}

export function getFirstTouch(): FirstTouch | null {
  if (typeof document === 'undefined') return null
  const value = document.cookie.split('; ').find(row => row.startsWith(`${FIRST_TOUCH_KEY}=`))?.slice(FIRST_TOUCH_KEY.length + 1)
  if (!value) return null
  try { return JSON.parse(decodeURIComponent(value)) as FirstTouch } catch { return null }
}

export function captureFirstTouch(): FirstTouch | null {
  if (typeof window === 'undefined' || typeof document === 'undefined') return null
  const existing = getFirstTouch()
  if (existing) return existing
  const touch = classifyFirstTouch(new URL(window.location.href), document.referrer)
  document.cookie = `${FIRST_TOUCH_KEY}=${encodeURIComponent(JSON.stringify(touch))}; Path=/; Max-Age=${FIRST_TOUCH_MAX_AGE}; SameSite=Lax; Secure`
  return touch
}

export function trackEvent(name: string, parameters: Record<string, string | number | boolean> = {}): void {
  if (typeof window === 'undefined') return
  window.gtag?.('event', name, parameters)
}

export function loadAnalytics(): void {
  if (typeof window === 'undefined' || loaded) return
  captureFirstTouch()
  if (document.documentElement.dataset.analytics === 'off') return

  const boot = () => {
    if (loaded) return
    loaded = true

    window.dataLayer = window.dataLayer || []
    window.gtag = function gtag(...args: unknown[]) {
      window.dataLayer!.push(args)
    }
    window.gtag('js', new Date())
    window.gtag('config', GA_ID, { send_page_view: true })

    const s = document.createElement('script')
    s.async = true
    s.src = `https://www.googletagmanager.com/gtag/js?id=${GA_ID}`
    document.head.appendChild(s)
  }

  const ric = (
    window as Window & {
      requestIdleCallback?: (cb: () => void, opts?: { timeout: number }) => number
    }
  ).requestIdleCallback

  if (typeof ric === 'function') {
    ric(boot, { timeout: 5000 })
  } else {
    setTimeout(boot, 2000)
  }
}
