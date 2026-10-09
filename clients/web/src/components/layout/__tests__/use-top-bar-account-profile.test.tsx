import { act, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { clearImpersonationToken, setAccessToken, setImpersonationToken } from '../../../lib/auth'
import { profileName } from '../top-bar-utils'
import { useTopBarAccountProfile } from '../use-top-bar-account-profile'

const fetchMock = vi.fn()

vi.mock('../../../lib/api', () => ({
  authorizedFetch: (...args: unknown[]) => fetchMock(...args),
}))

function jsonResponse(body: unknown, ok = true): Response {
  return {
    ok,
    json: async () => body,
  } as Response
}

function NameProbe() {
  const profile = useTopBarAccountProfile()
  return <span>{profileName(profile)}</span>
}

describe('useTopBarAccountProfile', () => {
  beforeEach(() => {
    fetchMock.mockReset()
    clearImpersonationToken()
    setAccessToken('parent-token')
  })

  it('clears the cached name when the Learn-as token ends, before the parent profile returns', async () => {
    let releaseParent: (value: Response) => void = () => {}
    fetchMock.mockImplementation(async () => {
      const calls = fetchMock.mock.calls.length
      if (calls === 1) {
        return jsonResponse({ email: 'kid@example.com', displayName: 'QA Test Kid' })
      }
      return new Promise<Response>((resolve) => {
        releaseParent = resolve
      })
    })

    render(<NameProbe />)
    await waitFor(() => {
      expect(screen.getByText('QA Test Kid')).toBeInTheDocument()
    })

    act(() => {
      setImpersonationToken('learn-as-token')
      clearImpersonationToken()
    })

    await waitFor(() => {
      expect(screen.getByText('Profile')).toBeInTheDocument()
    })
    expect(screen.queryByText('QA Test Kid')).not.toBeInTheDocument()

    releaseParent(jsonResponse({ email: 'parent@example.com', displayName: 'Parent User' }))
    await waitFor(() => {
      expect(screen.getByText('Parent User')).toBeInTheDocument()
    })
  })

  it('keeps the current name while an access token refresh refetches the same session', async () => {
    let releaseRefresh: (value: Response) => void = () => {}
    fetchMock.mockImplementation(async () => {
      if (fetchMock.mock.calls.length === 1) {
        return jsonResponse({ email: 'parent@example.com', displayName: 'Parent User' })
      }
      return new Promise<Response>((resolve) => {
        releaseRefresh = resolve
      })
    })

    render(<NameProbe />)
    await waitFor(() => {
      expect(screen.getByText('Parent User')).toBeInTheDocument()
    })

    act(() => {
      setAccessToken('refreshed-parent-token')
    })
    expect(screen.getByText('Parent User')).toBeInTheDocument()

    releaseRefresh(jsonResponse({ email: 'parent@example.com', displayName: 'Parent User' }))
    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledTimes(2)
    })
    expect(screen.getByText('Parent User')).toBeInTheDocument()
  })
})
