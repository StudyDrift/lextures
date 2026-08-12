export type MarketingContentEvent =
  | { event: 'marketing_content.list_viewed' }
  | { event: 'marketing_content.filter_applied'; filter: string }
  | { event: 'marketing_content.row_action'; action: string }

export function emitMarketingContentEvent(event: MarketingContentEvent) {
  if (typeof window === 'undefined' || navigator.doNotTrack === '1') return
  if (localStorage.getItem('lextures.analytics.opt-out') === '1') return
  window.dispatchEvent(new CustomEvent('lextures:analytics', { detail: event }))
}
