import { beforeEach, describe, expect, it, vi } from 'vitest'

const authorizedFetch = vi.fn()
vi.mock('../../../lib/api', () => ({ authorizedFetch: (...args: unknown[]) => authorizedFetch(...args) }))

import {
  fetchParentKidMessages,
  fetchParentKidMessagingContacts,
  setParentKidTeacherApproval,
} from '../../../lib/parent-messaging-api'
import { fetchLearnerSettings, updateLearnerSettings } from '../../../lib/learner-settings-api'
import { fetchMessagingPolicy, sendMessage, SendMessageError } from '../../../lib/communication-api'

const ok = (body: unknown) => ({ ok: true, status: 200, json: async () => body })

describe('kid-safe messaging API clients (#775)', () => {
  beforeEach(() => authorizedFetch.mockReset())

  it('loads a kid folder for the parent', async () => {
    authorizedFetch.mockResolvedValue(
      ok({ messages: [{ id: 'm1', from: { name: 'Kid', email: '' }, to: 'mom@e.com', subject: 'Hi', snippet: '', body: 'x', sentAt: '2026-10-12T00:00:00Z', read: true, starred: false, folder: 'sent', hasAttachment: false }] }),
    )
    const msgs = await fetchParentKidMessages('kid-1', 'sent')
    expect(authorizedFetch).toHaveBeenCalledWith('/api/v1/parent/students/kid-1/messages?folder=sent')
    expect(msgs[0].sent_at).toBe('2026-10-12T00:00:00Z')
    expect(msgs[0].has_attachment).toBe(false)
  })

  it('approves and removes teachers with PUT / DELETE', async () => {
    authorizedFetch.mockResolvedValue(ok({ parents: [], teachers: [] }))
    await setParentKidTeacherApproval('kid-1', 't-1', true)
    expect(authorizedFetch.mock.calls[0][0]).toBe('/api/v1/parent/students/kid-1/messaging/contacts/t-1')
    expect(authorizedFetch.mock.calls[0][1].method).toBe('PUT')
    await setParentKidTeacherApproval('kid-1', 't-1', false)
    expect(authorizedFetch.mock.calls[1][1].method).toBe('DELETE')
  })

  it('defaults missing contact lists', async () => {
    authorizedFetch.mockResolvedValue(ok({}))
    await expect(fetchParentKidMessagingContacts('kid-1')).resolves.toEqual({ parents: [], teachers: [] })
  })

  it('surfaces the server message on failure', async () => {
    authorizedFetch.mockResolvedValue({ ok: false, status: 404, json: async () => ({ error: { message: 'Messages are only available for learners you manage.' } }) })
    await expect(fetchParentKidMessages('kid-1', 'inbox')).rejects.toThrow('Messages are only available for learners you manage.')
  })

  it('reads the messaging policy', async () => {
    authorizedFetch.mockResolvedValue(ok({ restricted: true, recipients: [{ id: 'p', displayName: 'Mom', kind: 'parent' }] }))
    const p = await fetchMessagingPolicy()
    expect(authorizedFetch).toHaveBeenCalledWith('/api/v1/communication/recipients')
    expect(p.restricted).toBe(true)
    expect(p.recipients).toHaveLength(1)
  })

  it('sends by user id and reports a 403 recipient rejection', async () => {
    authorizedFetch.mockResolvedValue({ ok: false, status: 403, json: async () => ({ error: { message: 'You can only message your parent or a teacher they approved.' } }) })
    const err = await sendMessage({ toUserId: 'stranger', subject: 's', body: 'b' }).catch((e: unknown) => e)
    expect(err).toBeInstanceOf(SendMessageError)
    expect((err as SendMessageError).status).toBe(403)
    expect((err as SendMessageError).message).toMatch(/only message your parent/)
    expect(JSON.parse(authorizedFetch.mock.calls[0][1].body)).toMatchObject({ toUserId: 'stranger' })
  })

  it('loads and patches learner settings', async () => {
    authorizedFetch.mockResolvedValue(ok({ displayName: 'Kiddo', avatarUrl: null, uiTheme: 'dark' }))
    await expect(fetchLearnerSettings()).resolves.toEqual({ displayName: 'Kiddo', avatarUrl: null, uiTheme: 'dark' })
    await updateLearnerSettings({ uiTheme: 'dark' })
    expect(authorizedFetch.mock.calls[1][0]).toBe('/api/v1/me/learner-settings')
    expect(authorizedFetch.mock.calls[1][1].method).toBe('PATCH')
  })
})
