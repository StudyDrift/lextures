import { describe, expect, it } from 'vitest'

describe('study-buddy-api', () => {
  it('exports fetch helpers', async () => {
    const mod = await import('./study-buddy-api')
    expect(typeof mod.fetchStudyBuddyPrompts).toBe('function')
    expect(typeof mod.sendStudyBuddyMessage).toBe('function')
  })
})
