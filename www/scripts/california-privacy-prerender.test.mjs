import assert from 'node:assert/strict'
import { createElement } from 'react'
import { renderToString } from 'react-dom/server'
import { describe, it } from 'node:test'
import { createServer } from 'vite'

describe('California privacy rights prerender', () => {
  it('ships the CCPA copy instead of a loading spinner', async () => {
    const server = await createServer({
      server: { middlewareMode: true },
      appType: 'custom',
      logLevel: 'error',
    })
    try {
      const mod = await server.ssrLoadModule('/src/pages/california-privacy-rights-page.tsx')
      const html = renderToString(createElement(mod.CaliforniaPrivacyRightsPage))
      assert.match(html, /Your California Privacy Rights/)
      assert.match(html, /California Consumer Privacy Act/)
      assert.match(html, /Submit a California Privacy Rights Request/)
      assert.doesNotMatch(html, /aria-label="Loading"/)
      assert.doesNotMatch(html, /animate-spin/)
    } finally {
      await server.close()
    }
  })
})
