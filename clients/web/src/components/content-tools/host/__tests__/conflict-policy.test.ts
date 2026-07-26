import { describe, expect, it } from 'vitest'
import { defaultMergeReducer, resolveConflictState } from '../conflict-policy'

describe('resolveConflictState', () => {
  const client = { response: 'client', attempts: 2 }
  const server = { response: 'server', attempts: 1, extra: true }

  it('server_wins returns a copy of the server document', () => {
    const next = resolveConflictState('server_wins', client, server)
    expect(next).toEqual(server)
    expect(next).not.toBe(server)
  })

  it('client_wins returns a copy of the client document', () => {
    const next = resolveConflictState('client_wins', client, server)
    expect(next).toEqual(client)
    expect(next).not.toBe(client)
  })

  it('merge uses the default reducer (client keys win)', () => {
    expect(resolveConflictState('merge', client, server)).toEqual({
      response: 'client',
      attempts: 2,
      extra: true,
    })
  })

  it('merge accepts a custom reducer', () => {
    const next = resolveConflictState('merge', client, server, (c, s) => ({
      ...defaultMergeReducer(c, s),
      attempts: Math.max(Number(c.attempts) || 0, Number(s.attempts) || 0),
    }))
    expect(next.attempts).toBe(2)
    expect(next.response).toBe('client')
  })
})
