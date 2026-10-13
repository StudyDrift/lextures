import { afterEach, describe, expect, it } from 'vitest'
import { clearImpersonationToken, setImpersonationToken } from '../auth'
import { isManagedLearnerSession, jwtTokenType } from '../managed-learner-session'
import { folderEmptyCopy, inboxComposeCopy, recipientLabel } from '../inbox-copy'
import { notificationSoundsEnabled, setNotificationSoundsEnabled, shouldChimeForUnread } from '../notification-sounds'
import type { MailboxFolder } from '../communication-api'

function fakeJwt(payload: Record<string, unknown>): string {
  const b64 = btoa(JSON.stringify(payload)).replace(/=+$/, '').replace(/\+/g, '-').replace(/\//g, '_')
  return `h.${b64}.s`
}

describe('managed learner session (#775)', () => {
  afterEach(() => clearImpersonationToken())

  it('reads the JWT typ claim', () => {
    expect(jwtTokenType(fakeJwt({ typ: 'managed_learner' }))).toBe('managed_learner')
    expect(jwtTokenType('not-a-jwt')).toBeNull()
    expect(jwtTokenType(null)).toBeNull()
  })

  it('is true only for a managed_learner bearer, not admin view-as', () => {
    expect(isManagedLearnerSession()).toBe(false)
    setImpersonationToken(fakeJwt({ typ: 'impersonation' }))
    expect(isManagedLearnerSession()).toBe(false)
    setImpersonationToken(fakeJwt({ typ: 'managed_learner' }))
    expect(isManagedLearnerSession()).toBe(true)
  })
})

describe('inbox copy (#775)', () => {
  const folders: MailboxFolder[] = ['inbox', 'starred', 'sent', 'drafts', 'trash']

  it('never mentions campus, for kids or adults', () => {
    for (const kid of [true, false]) {
      for (const f of folders) {
        for (const searching of [true, false]) {
          const c = folderEmptyCopy(f, searching, kid)
          expect(`${c.title} ${c.body}`.toLowerCase()).not.toContain('campus')
        }
      }
    }
  })

  it('uses "Message your parent" in kid sessions', () => {
    expect(inboxComposeCopy(true).composeButton).toBe('Message your parent')
    expect(inboxComposeCopy(true).dialogTitle).toBe('Message your parent')
    expect(folderEmptyCopy('inbox', false, true).body).toMatch(/message your parent/i)
    expect(inboxComposeCopy(false).composeButton).toBe('Compose')
  })

  it('labels recipients by role', () => {
    expect(recipientLabel({ id: '1', displayName: 'Mom', kind: 'parent' })).toBe('Mom (parent)')
    expect(recipientLabel({ id: '2', displayName: 'Ms. Lee', kind: 'teacher', courseTitles: ['Science'] })).toBe(
      'Ms. Lee (teacher, Science)',
    )
  })
})

describe('notification sounds (#775)', () => {
  afterEach(() => setNotificationSoundsEnabled(false))

  it('is off by default and chimes only when unread goes up', () => {
    expect(notificationSoundsEnabled()).toBe(false)
    expect(shouldChimeForUnread(0, 1)).toBe(false)
    setNotificationSoundsEnabled(true)
    expect(shouldChimeForUnread(null, 3)).toBe(false)
    expect(shouldChimeForUnread(2, 2)).toBe(false)
    expect(shouldChimeForUnread(2, 1)).toBe(false)
    expect(shouldChimeForUnread(2, 3)).toBe(true)
  })
})
