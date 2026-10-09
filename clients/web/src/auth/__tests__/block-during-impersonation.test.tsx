import { render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom'
import { beforeEach, describe, expect, it } from 'vitest'
import { clearImpersonationToken, setImpersonationToken } from '../../lib/auth'
import { BlockDuringImpersonation } from '../block-during-impersonation'

function LocationProbe() {
  const location = useLocation()
  return <div>{location.pathname}</div>
}

function renderAt(path: string) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route path="/" element={<LocationProbe />} />
        <Route
          path="/learners"
          element={
            <BlockDuringImpersonation>
              <h1>Learners</h1>
            </BlockDuringImpersonation>
          }
        />
        <Route
          path="/me/billing"
          element={
            <BlockDuringImpersonation>
              <h1>Billing</h1>
            </BlockDuringImpersonation>
          }
        />
      </Routes>
    </MemoryRouter>,
  )
}

describe('BlockDuringImpersonation', () => {
  beforeEach(() => {
    clearImpersonationToken()
  })

  it('renders the page when no impersonation token is active', () => {
    renderAt('/learners')
    expect(screen.getByRole('heading', { name: 'Learners' })).toBeInTheDocument()
  })

  it('redirects Learners while a Learn-as token is active', () => {
    setImpersonationToken('learn-as-token')
    renderAt('/learners')
    expect(screen.queryByRole('heading', { name: 'Learners' })).not.toBeInTheDocument()
    expect(screen.getByText('/')).toBeInTheDocument()
  })

  it('redirects Billing while a Learn-as token is active', () => {
    setImpersonationToken('learn-as-token')
    renderAt('/me/billing')
    expect(screen.queryByRole('heading', { name: 'Billing' })).not.toBeInTheDocument()
    expect(screen.getByText('/')).toBeInTheDocument()
  })
})
