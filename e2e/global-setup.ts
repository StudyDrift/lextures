import { seedE2EPlatformFeatures } from './fixtures/platform-features.js'

export default async function globalSetup(): Promise<void> {
  await seedE2EPlatformFeatures()
}
