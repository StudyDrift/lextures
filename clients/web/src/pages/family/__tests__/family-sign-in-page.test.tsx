import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it } from 'vitest'
import FamilySignInPage from '../family-sign-in-page'
import { clearFamilySessionLocally, clearStoredFamilyDevice, storeFamilyDevice } from '../../../lib/family-device'
import { server } from '../../../test/mocks/server'

const API = 'http://localhost:8080'

function renderPage() {
  return render(
    <MemoryRouter initialEntries={['/family']}>
      <FamilySignInPage />
    </MemoryRouter>,
  )
}

describe('FamilySignInPage', () => {
  afterEach(() => {
    clearFamilySessionLocally()
    clearStoredFamilyDevice()
  })

  it('explains how to set up a device that is not a family device', async () => {
    renderPage()
    expect(await screen.findByText(/isn't set up for learner sign-in/i)).toBeInTheDocument()
  })

  it("shows Who's learning? and asks the chosen learner for their PIN", async () => {
    storeFamilyDevice({ id: 'dev-1', token: 'secret', label: 'Family device' })
    let signInBody: unknown = null
    server.use(
      http.post(`${API}/api/v1/auth/family-device/learners`, () =>
        HttpResponse.json({
          learners: [
            { id: 'kid-1', displayName: 'Alex', signInMethod: 'pin' },
            { id: 'kid-2', displayName: 'Sam', signInMethod: 'none' },
          ],
          pictureChoices: [],
        }),
      ),
      http.post(`${API}/api/v1/auth/family-device/sessions`, async ({ request }) => {
        signInBody = await request.json()
        return HttpResponse.json(
          { error: { code: 'INVALID_CREDENTIALS', message: 'no' }, attemptsRemaining: 4 },
          { status: 401 },
        )
      }),
    )
    const user = userEvent.setup()
    renderPage()
    expect(await screen.findByRole('heading', { name: "Who's learning?" })).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: /Alex/ }))
    expect(screen.getByRole('heading', { name: 'Hi, Alex!' })).toBeInTheDocument()
    for (const d of ['1', '2', '3', '4']) {
      await user.click(screen.getByRole('button', { name: d }))
    }
    await user.click(screen.getByRole('button', { name: 'Start learning' }))
    await waitFor(() => expect(screen.getByText(/4 tries left/)).toBeInTheDocument())
    expect(signInBody).toEqual({ deviceToken: 'secret', learnerId: 'kid-1', pin: '1234' })
  })

  it('tells a learner without sign-in to ask a parent', async () => {
    storeFamilyDevice({ id: 'dev-1', token: 'secret', label: 'Family device' })
    server.use(
      http.post(`${API}/api/v1/auth/family-device/learners`, () =>
        HttpResponse.json({ learners: [{ id: 'kid-2', displayName: 'Sam', signInMethod: 'none' }], pictureChoices: [] }),
      ),
    )
    const user = userEvent.setup()
    renderPage()
    await user.click(await screen.findByRole('button', { name: /Sam/ }))
    expect(screen.getByText(/Ask a parent to set up your PIN/)).toBeInTheDocument()
  })
})
