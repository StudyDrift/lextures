/// <reference lib="webworker" />
import { precacheAndRoute, cleanupOutdatedCaches, createHandlerBoundToURL } from 'workbox-precaching'
import { registerRoute, NavigationRoute } from 'workbox-routing'
import { CacheFirst, NetworkFirst } from 'workbox-strategies'
import { ExpirationPlugin } from 'workbox-expiration'
import { BackgroundSyncPlugin, Queue } from 'workbox-background-sync'

declare const self: ServiceWorkerGlobalScope

// Skip waiting immediately so new SW activates on update
self.addEventListener('message', (event) => {
  if (event.data?.type === 'SKIP_WAITING') {
    void self.skipWaiting()
  }
})

self.addEventListener('activate', (event) => {
  // Take control of open tabs so the new precache (new chunk hashes) is used.
  event.waitUntil(self.clients.claim())
})

// Precache the app shell (list injected by vite-plugin-pwa)
precacheAndRoute(self.__WB_MANIFEST)
cleanupOutdatedCaches()

// SPA navigation fallback: serve index.html for all navigation requests
registerRoute(new NavigationRoute(createHandlerBoundToURL('/index.html')))

/**
 * Never cache HTML (or other non-matching types) under script/style URLs.
 * After a deploy, nginx SPA fallback used to return index.html for missing
 * hashed chunks; CacheFirst would then serve that HTML forever as "JS".
 */
function onlyMatchingContentType(expected: RegExp) {
  return {
    cacheWillUpdate: async ({ response }: { response: Response }) => {
      if (!response || response.status !== 200) return null
      const ct = response.headers.get('content-type') || ''
      return expected.test(ct) ? response : null
    },
  }
}

// Fonts + images: cache-first (not fingerprinted the same way as Vite chunks).
// Scripts and styles are already in the Workbox precache; do NOT runtime-cache
// them — a transient HTML 200 during deploy would poison the cache.
registerRoute(
  ({ request, url }) =>
    !url.pathname.startsWith('/api/') &&
    (request.destination === 'font' || request.destination === 'image'),
  new CacheFirst({
    cacheName: 'static-assets-v3',
    plugins: [
      new ExpirationPlugin({ maxEntries: 150, maxAgeSeconds: 30 * 24 * 60 * 60 }),
      onlyMatchingContentType(/^(image|font)\//i),
    ],
  }),
)

// Network-first for all API calls — always serve fresh data when online,
// fall back to cache only when offline (offline-first UX via IndexedDB handles the rest).
registerRoute(
  ({ url }) =>
    url.pathname.startsWith('/api/') &&
    !url.pathname.includes('/push') &&
    !url.pathname.includes('/vapid'),
  new NetworkFirst({
    cacheName: 'api-cache-v2',
    networkTimeoutSeconds: 10,
    plugins: [
      new ExpirationPlugin({ maxEntries: 200, maxAgeSeconds: 10 * 60 }),
    ],
  }),
)

// Background sync queues for offline submissions
const quizSyncQueue = new Queue('quiz-sync-queue', {
  maxRetentionTime: 24 * 60,
  onSync: async ({ queue }) => {
    let entry
    while ((entry = await queue.shiftRequest())) {
      try {
        await fetch(entry.request)
      } catch {
        await queue.unshiftRequest(entry)
        throw new Error('quiz-sync: network unavailable, retrying later')
      }
    }
  },
})

const discussionSyncQueue = new Queue('discussion-sync-queue', {
  maxRetentionTime: 24 * 60,
  onSync: async ({ queue }) => {
    let entry
    while ((entry = await queue.shiftRequest())) {
      try {
        await fetch(entry.request)
      } catch {
        await queue.unshiftRequest(entry)
        throw new Error('discussion-sync: network unavailable, retrying later')
      }
    }
  },
})

// Intercept failed quiz submission requests and add to sync queue
self.addEventListener('fetch', (event) => {
  const { request } = event
  if (
    request.method === 'POST' &&
    (request.url.includes('/quiz-attempts') || request.url.includes('/quiz_attempts'))
  ) {
    const bgSyncPlugin = new BackgroundSyncPlugin('quiz-sync-queue', {
      maxRetentionTime: 24 * 60,
    })
    void bgSyncPlugin
    const handler = async () => {
      try {
        return await fetch(request.clone())
      } catch {
        await quizSyncQueue.pushRequest({ request })
        return new Response(JSON.stringify({ queued: true }), {
          status: 202,
          headers: { 'Content-Type': 'application/json' },
        })
      }
    }
    event.respondWith(handler())
    return
  }

  if (
    request.method === 'POST' &&
    request.url.includes('/discussions')
  ) {
    const handler = async () => {
      try {
        return await fetch(request.clone())
      } catch {
        await discussionSyncQueue.pushRequest({ request })
        return new Response(JSON.stringify({ queued: true }), {
          status: 202,
          headers: { 'Content-Type': 'application/json' },
        })
      }
    }
    event.respondWith(handler())
  }
})

// Push notification handling (preserved from original sw.js)
self.addEventListener('push', (event) => {
  let data: { title: string; body: string; url?: string } = { title: 'New notification', body: '' }
  if (event.data) {
    try {
      data = event.data.json() as typeof data
    } catch {
      data.body = event.data.text()
    }
  }
  const options: NotificationOptions = {
    body: data.body ?? '',
    icon: '/favicon.svg',
    badge: '/favicon.svg',
    data: { url: data.url ?? '/' },
    requireInteraction: false,
  }
  event.waitUntil(self.registration.showNotification(data.title, options))
})

self.addEventListener('notificationclick', (event) => {
  event.notification.close()
  const targetUrl: string =
    (event.notification.data as { url?: string } | null)?.url ?? '/'
  event.waitUntil(
    self.clients
      .matchAll({ type: 'window', includeUncontrolled: true })
      .then((windowClients) => {
        for (const client of windowClients) {
          if (client.url === targetUrl && 'focus' in client) {
            return client.focus()
          }
        }
        if (self.clients.openWindow) {
          return self.clients.openWindow(targetUrl)
        }
      }),
  )
})
