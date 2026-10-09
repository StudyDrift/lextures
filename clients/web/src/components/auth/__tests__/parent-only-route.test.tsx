import { render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, describe, expect, it } from 'vitest'
import { clearImpersonationToken, setImpersonationToken } from '../../../lib/auth'
import { ParentOnlyRoute } from '../parent-only-route'

function jwtWithTyp(typ: string): string {
  const b64 = (o: object) => btoa(JSON.stringify(o)).replace(/=+$/, '')
  return `${b64({ alg: 'none' })}.${b64({ typ, sub: 'u1' })}.sig`
}

function renderRoute() {
  return render(
    <MemoryRouter initialEntries={['/learners']}>
      <Routes>
        <Route path="/" element={<p>Dashboard</p>} />
        <Route
          path="/learners"
          element={
            <ParentOnlyRoute>
              <p>Learners page</p>
            </ParentOnlyRoute>
          }
        />
      </Routes>
    </MemoryRouter>,
  )
}

describe('ParentOnlyRoute', () => {
  afterEach(() => {
    clearImpersonationToken()
  })

  it('renders the page for the parent', () => {
    renderRoute()
    expect(screen.getByText('Learners page')).toBeInTheDocument()
  })

  it('sends a managed learner to the dashboard instead of an error page', () => {
    setImpersonationToken(jwtWithTyp('managed_learner'))
    renderRoute()
    expect(screen.getByText('Dashboard')).toBeInTheDocument()
    expect(screen.queryByText('Learners page')).not.toBeInTheDocument()
  })
})
