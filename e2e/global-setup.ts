import { seedE2EPlatformFeatures } from './fixtures/platform-features.js'

export default async function globalSetup(): Promise<void> {
  try {
    await seedE2EPlatformFeatures()
  } catch (err) {
    const msg = err instanceof Error ? err.message : String(err)
    // Local dev stacks may lack the e2e global-admin bootstrap; tests that need
    // platform flags still run under e2e-local / CI where seed succeeds.
    if (msg.includes('403') || msg.includes('FORBIDDEN')) {
      console.warn('[e2e] Platform settings seed skipped (no global admin on this API).')
      return
    }
    throw err
  }
}
